package docker

import (
	"context"
	"fmt"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	mb = 1048576
)

type docker struct {
	client            *client.Client
	logger            lumber.Logger
	cfg               *config.SynapseConfig
	secretsManager    core.SecretsManager
	cpu               float32
	ram               int64
	RunningContainers []*core.RunnerOptions
}

// newDockerClient creates a new docker client
func newDockerClient(secretsManager core.SecretsManager) (*docker, error) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	dockerInfo, err := client.Info(context.TODO())
	if err != nil {
		return nil, err
	}

	return &docker{
		client:         client,
		cpu:            float32(dockerInfo.NCPU),
		ram:            dockerInfo.MemTotal / mb,
		secretsManager: secretsManager,
	}, nil
}

// New initialize a new docker configuration
func New(secretsManager core.SecretsManager,
	logger lumber.Logger,
	cfg *config.SynapseConfig) (core.DockerRunner, error) {
	dockerConfig, err := newDockerClient(secretsManager)
	if err != nil {
		return nil, err
	}
	dockerConfig.logger = logger
	dockerConfig.cfg = cfg

	logger.Infof("available cpu: %f", dockerConfig.cpu)
	logger.Infof("available memory: %d", dockerConfig.ram)

	return dockerConfig, nil
}

func (d *docker) Create(ctx context.Context, r *core.RunnerOptions) core.ContainerStatus {
	containerStatus := core.ContainerStatus{Done: true}
	container, err := d.getContainerConfiguration(r)
	if err != nil {
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(err.Error())
	}
	hostConfig := d.getContainerHostConfiguration(r)
	networkConfig, err := d.getContainerNetworkConfiguration()
	if err != nil {
		d.logger.Errorf("error retriving network: %v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(err.Error())
		return containerStatus
	}

	resp, err := d.client.ContainerCreate(ctx, container, hostConfig, networkConfig, nil, fmt.Sprintf("%s-%s", r.ContainerName, r.PodType))
	r.ContainerID = resp.ID
	if err != nil {
		d.logger.Errorf("error creating container: %v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(err.Error())
		return containerStatus
	}
	d.logger.Debugf("container created with name: %s, updating status %+v",
		fmt.Sprintf("%s-%s", r.ContainerName, r.PodType), containerStatus)
	return containerStatus
}

func (d *docker) Destroy(ctx context.Context, r *core.RunnerOptions) error {
	if err := d.client.ContainerStop(ctx, r.ContainerID, nil); err != nil {
		d.logger.Errorf("error stopping container %v", err)
		return err
	}
	err := d.client.ContainerRemove(ctx, r.ContainerID, types.ContainerRemoveOptions{})
	if err != nil {
		d.logger.Errorf("error removing container %v", err)
		return err
	}
	return nil
}

func (d *docker) Run(ctx context.Context, r *core.RunnerOptions) core.ContainerStatus {
	containerStatus := core.ContainerStatus{Done: true}
	d.logger.Debugf("running container %s", r.ContainerID)
	if err := d.client.ContainerStart(ctx, r.ContainerID, types.ContainerStartOptions{}); err != nil {
		d.logger.Errorf("error starting the container: %s", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_STRT(err.Error())
		return containerStatus
	}
	d.RunningContainers = append(d.RunningContainers, r)

	err := d.waitForRunning(ctx, r)
	if err != nil {
		d.logger.Errorf("error while waiting for the running container: %v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_RUN(err.Error())
		d.RunningContainers = removeContainerID(d.RunningContainers, r)
		return containerStatus
	}
	d.RunningContainers = removeContainerID(d.RunningContainers, r)

	d.logger.Debugf("Updating status %+v", containerStatus)

	return containerStatus
}

// removing element from slice of string
func removeContainerID(slice []*core.RunnerOptions, r *core.RunnerOptions) []*core.RunnerOptions {
	index := -1
	for i, val := range slice {
		if val.ContainerID == r.ContainerID {
			index = i
			break
		}
	}
	if index == -1 {
		return slice
	}
	newSlice := make([]*core.RunnerOptions, 0)
	newSlice = append(newSlice, slice[:index]...)
	if index != len(slice)-1 {
		newSlice = append(newSlice, slice[index+1:]...)
	}

	return newSlice
}

func (d *docker) waitForRunning(ctx context.Context, r *core.RunnerOptions) error {
	d.logger.Infof("waiting for  container %s compeletion", r.ContainerID)
	statusCh, errCh := d.client.ContainerWait(ctx, r.ContainerID, container.WaitConditionRemoved)

	select {
	case err := <-errCh:
		if err != nil {
			d.logger.Debugf("%s container terminated with exit code: %d, reason %s", r.ContainerID, err)

			return err
		}
	case status := <-statusCh:
		d.logger.Debugf("status code: %d", status.StatusCode)
		if status.StatusCode != 0 {
			msg := fmt.Sprintf("Received non zero status code %v", status.StatusCode)
			return errs.ERR_DOCKER_RUN(msg)

		}
		return nil
	}
	return nil
}

func (d *docker) GetInfo(ctx context.Context) (float32, int64) {
	return d.cpu, d.ram
}

func (d *docker) Initiate(ctx context.Context, r *core.RunnerOptions, statusChan chan core.ContainerStatus) {
	// creating the docker contaienr
	if status := d.Create(ctx, r); !status.Done {
		d.logger.Errorf("error creating container: %v", status.Error)
		d.logger.Infof("Update error status after creation")
		statusChan <- status
		return
	}
	if status := d.Run(ctx, r); !status.Done {
		d.logger.Errorf("error running container: %v", status.Error)
		d.logger.Infof("Update error status after running")

		statusChan <- status
		return
	}
	d.logger.Infof("container %+s executuion successful", r.ContainerID)
	statusChan <- core.ContainerStatus{Done: true}
}

func (d *docker) KillRunningDocker(ctx context.Context) {
	for _, r := range d.RunningContainers {
		d.logger.Infof("Destroying container %s", r.ContainerID)
		if err := d.Destroy(ctx, r); err != nil {
			d.logger.Errorf("Error occur while destroying container ID %s , err %+v", r.ContainerID, err)
		}
	}
}
