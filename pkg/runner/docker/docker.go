package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/synapse"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
)

const (
	buildCacheExpiry time.Duration = 4 * time.Hour
	BuildID                        = "build-id"
)

var gracefulyContainerStopDuration = time.Second * 10

var networkName string

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
	networkName = os.Getenv(global.NetworkEnvName)
	return &docker{
		client:         client,
		cpu:            float32(dockerInfo.NCPU),
		ram:            dockerInfo.MemTotal / units.MiB,
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

func (d *docker) CreateVolume(ctx context.Context, r *core.RunnerOptions) error {
	volumeOptions := d.getVolumeConfiguration(r)
	isVolume, err := d.FindVolumes(volumeOptions.Name)
	if err != nil {
		return err
	}
	if !isVolume {
		if _, err := d.client.VolumeCreate(ctx, *volumeOptions); err != nil {
			return err
		}
	}
	return nil
}

func (d *docker) CopyFileToContainer(ctx context.Context, path, fileName, containerID string, content []byte) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	if err := tw.WriteHeader(&tar.Header{
		Name: fileName,
		Mode: 0777,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}

	if err := d.client.CopyToContainer(
		ctx,
		containerID,
		global.VaultSecretDir,
		&buf,
		types.CopyToContainerOptions{AllowOverwriteDirWithFile: true},
	); err != nil {
		return err
	}
	return nil
}

func (d *docker) Create(ctx context.Context, r *core.RunnerOptions) core.ContainerStatus {
	containerStatus := core.ContainerStatus{Done: true}
	containerImageConfig, err := d.secretsManager.GetDockerSecrets(r)
	if err != nil {
		d.logger.Errorf("Something went wrong while seeking docker secrets %+v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(err.Error())
		return containerStatus
	}

	if err = d.CreateVolume(ctx, r); err != nil {
		d.logger.Errorf("Error in creating docker volume: %+v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ErrDockerVolCrt(err.Error())
		return containerStatus
	}

	if errP := d.PullImage(&containerImageConfig, r); errP != nil {
		d.logger.Errorf("Something went wrong while pulling container image %+v", errP)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(errP.Error())
		return containerStatus
	}
	containerConfig := d.getContainerConfiguration(r)
	hostConfig := d.getContainerHostConfiguration(r)
	networkConfig, err := d.getContainerNetworkConfiguration()
	if err != nil {
		d.logger.Errorf("error retrieving network: %v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(err.Error())
		return containerStatus
	}
	containerName := fmt.Sprintf("%s-%s", r.ContainerName, r.PodType)
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	r.ContainerID = resp.ID
	if err != nil {
		d.logger.Errorf("error creating container: %v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_CRT(err.Error())
		return containerStatus
	}
	d.logger.Debugf("container created with name: %s, updating status %+v",
		fmt.Sprintf("%s-%s", r.ContainerName, r.PodType), containerStatus)

	gitSecretBytes, err := d.secretsManager.GetGitSecretBytes()
	if err != nil {
		d.logger.Errorf("Error in loading git secrets: %s", err.Error())
		containerStatus.Done = false
		containerStatus.Error = errs.ErrSecretLoad(err.Error())
		return containerStatus
	}
	if err = d.CopyFileToContainer(
		ctx,
		global.VaultSecretDir,
		global.GitConfigFileName,
		r.ContainerID,
		gitSecretBytes,
	); err != nil {
		containerStatus.Done = false
		containerStatus.Error = errs.ErrDockerCP(err.Error())
		return containerStatus
	}

	// copies repo secrets to container
	repoSecretBytes, err := d.secretsManager.GetRepoSecretBytes(r.Label["repo"])
	if err != nil {
		d.logger.Debugf("Error in loading repo secrets: %s", err.Error())
	} else {
		if err := d.CopyFileToContainer(
			ctx,
			global.VaultSecretDir,
			global.RepoSecretsFileName,
			r.ContainerID,
			repoSecretBytes,
		); err != nil {
			containerStatus.Done = false
			containerStatus.Error = errs.ErrDockerCP(err.Error())
			return containerStatus
		}
	}
	return containerStatus
}

func (d *docker) Destroy(ctx context.Context, r *core.RunnerOptions) error {
	if err := d.client.ContainerStop(ctx, r.ContainerID, &gracefulyContainerStopDuration); err != nil {
		d.logger.Errorf("error stopping container %v", err)
		return err
	}
	autoRemove, err := strconv.ParseBool(os.Getenv(global.AutoRemoveEnv))
	if err != nil {
		d.logger.Errorf("Error reading AutoRemove os env error: %v", err)
		return errors.New("error reading AutoRemove os env error")
	}
	if autoRemove {
		// if autoRemove is set then it docker container will be removed once it stopped or exited
		return nil
	}
	err = d.client.ContainerRemove(ctx, r.ContainerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
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

	if err := d.writeLogs(ctx, r); err != nil {
		d.logger.Errorf("error writing logs to stdout: %+v", err)
	}

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

func (d *docker) WaitForCompletion(ctx context.Context, r *core.RunnerOptions) error {
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

func (d *docker) GetInfo(ctx context.Context) (cpu float32, ram int64) {
	return d.cpu, d.ram
}

func (d *docker) Initiate(ctx context.Context, r *core.RunnerOptions, statusChan chan core.ContainerStatus) {
	// creating the docker contaienr
	r.ContainerArgs = append(r.ContainerArgs, "--local", os.Getenv(global.LocalEnv), "--synapsehost", os.Getenv(global.SynapseHostEnv))
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
	containerStatus := core.ContainerStatus{Done: true}

	if err := d.WaitForCompletion(ctx, r); err != nil {
		d.logger.Errorf("error while waiting for the completion of container: %v", err)
		containerStatus.Done = false
		containerStatus.Error = errs.ERR_DOCKER_RUN(err.Error())
		d.RunningContainers = removeContainerID(d.RunningContainers, r)
		statusChan <- containerStatus
		return
	}
	d.RunningContainers = removeContainerID(d.RunningContainers, r)
	d.logger.Infof("container %+s execution successful", r.ContainerID)
	statusChan <- containerStatus
}

func (d *docker) KillRunningDocker(ctx context.Context) {
	for _, r := range d.RunningContainers {
		d.logger.Infof("Destroying container %s", r.ContainerID)
		if err := d.Destroy(ctx, r); err != nil {
			d.logger.Errorf("Error occur while destroying container ID %s , err %+v", r.ContainerID, err)
		}
	}
}

func (d *docker) KillContainerForBuildID(buildID string) error {
	for _, r := range d.RunningContainers {
		if r.Label[BuildID] == buildID {
			if err := d.Destroy(context.Background(), r); err != nil {
				d.logger.Errorf("error while destroying container: %v", err)
				return err
			}
			return nil
		}
	}
	return nil
}

func (d *docker) PullImage(containerImageConfig *core.ContainerImageConfig, r *core.RunnerOptions) error {
	if containerImageConfig.PullPolicy == config.PullNever && r.PodType == core.NucleusPod {
		d.logger.Infof("pull policy %s pod type %s, not pulling any image",
			containerImageConfig.PullPolicy, r.PodType)
		return nil
	}
	dockerImage := containerImageConfig.Image

	d.logger.Infof("Pulling image : %s", dockerImage)
	ImagePullOptions := types.ImagePullOptions{}
	ImagePullOptions.RegistryAuth = containerImageConfig.AuthRegistry
	reader, err := d.client.ImagePull(context.TODO(), dockerImage, ImagePullOptions)
	defer func() {
		if reader == nil {
			d.logger.Errorf("Reader returned by docker pull is null")
			return
		}
		if errC := reader.Close(); errC != nil {
			d.logger.Errorf(errC.Error())
		}
	}()

	if err != nil {
		return err
	}
	if _, err := io.Copy(os.Stdout, reader); err != nil {
		return err
	}
	return nil
}

// writeLogs writes container logs to a file
func (d *docker) writeLogs(ctx context.Context, r *core.RunnerOptions) error {
	reader, err := d.client.ContainerLogs(ctx,
		r.ContainerID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
	if err != nil {
		return err
	}
	defer reader.Close()

	buildLogsPath := fmt.Sprintf("%s/%s", global.ExecutionLogsPath, r.Label[synapse.BuildID])

	if errDir := utils.CreateDirectory(buildLogsPath); err != nil {
		return errDir
	}

	f, err := os.Create(fmt.Sprintf("%s/%s-%s.log", buildLogsPath, r.ContainerName, r.PodType))
	if err != nil {
		return err
	}
	defer f.Close()

	if _, errCopy := stdcopy.StdCopy(f, f, reader); err != nil {
		return errCopy
	}

	return nil
}

func (d *docker) FindVolumes(volumeName string) (bool, error) {
	volumeFilter := filters.KeyValuePair{Key: "name", Value: volumeName}
	volumes, err := d.client.VolumeList(context.Background(), filters.NewArgs(volumeFilter))
	if err != nil {
		return false, err
	}
	for _, v := range volumes.Volumes {
		if v.Name == volumeName {
			return true, nil
		}
	}
	return false, nil
}

func (d *docker) RemoveVolume(ctx context.Context, volumeName string) error {
	if err := d.client.VolumeRemove(ctx, volumeName, true); err != nil {
		return err
	}
	return nil
}

func (d *docker) RemoveOldVolumes(ctx context.Context) {
	volumes, err := d.client.VolumeList(context.Background(), filters.NewArgs())
	if err != nil {
		d.logger.Errorf("error fetching volume lists: %v", err.Error())
	}
	for _, v := range volumes.Volumes {
		if strings.HasPrefix(v.Name, volumePrefix) {
			_, data, err := d.client.VolumeInspectWithRaw(context.Background(), v.Name)
			if err == nil {
				var volumeDetails core.VolumeDetails
				err = json.Unmarshal(data, &volumeDetails)
				if err != nil {
					d.logger.Errorf("error in unmarshaling volume details: %v", err.Error())
					continue
				}

				now := time.Now()
				diff := now.Sub(volumeDetails.CreatedAt)
				if diff > buildCacheExpiry {
					d.logger.Debugf("Deleting volume: %s", v.Name)
					if err = d.RemoveVolume(ctx, v.Name); err != nil {
						d.logger.Errorf("Error deleting volume: %v", err.Error())
					}
				}
			} else {
				d.logger.Errorf("error in fetching volume details: %v", err.Error())
			}
		}
	}
}
