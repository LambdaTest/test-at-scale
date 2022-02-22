package docker

import (
	"context"
	"io"
	"os"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
)

const (
	networkName                = "test-at-scale"
	defaultContainerVolumePath = "/home/nucleus"
	defaultVaultPath           = "/vault/secrets"
	repoSourcePath             = "/tmp/synapse/nucleus"
	nanoCPUUnit                = 1e9
	// GB defines number of bytes in 1 GB
	GB int64 = 1e+9
)

func (d *docker) getContainerConfiguration(r *core.RunnerOptions) (*container.Config, error) {
	containerImageConfig, err := d.secretsManager.GetDockerSecrets(r)
	if err != nil {
		d.logger.Errorf("Something went wrong while seeking container config %+v", err)
	}

	r.ContainerArgs = append(r.ContainerArgs, "--local", "true")
	localIp := utils.GetOutboundIP()
	r.ContainerArgs = append(r.ContainerArgs, "--synapsehost", localIp)
	if containerImageConfig.PullPolicy == config.PullNever && r.PodType == core.NucleusPod {
		d.logger.Infof("pull policy %s, not pulling any image", containerImageConfig.PullPolicy)
		return &container.Config{
			Image:   r.DockerImage,
			Env:     r.Env,
			Tty:     false,
			Cmd:     r.ContainerArgs,
			Volumes: make(map[string]struct{}),
		}, nil
	}
	if err := d.PullImage(&containerImageConfig); err != nil {
		d.logger.Errorf("Something went wrong while pulling image %s", err)
		return nil, err
	}

	return &container.Config{
		Image:   r.DockerImage,
		Env:     r.Env,
		Tty:     false,
		Cmd:     r.ContainerArgs,
		Volumes: make(map[string]struct{}),
	}, nil
}

func (d *docker) PullImage(containerImageConfig *core.ContainerImageConfig) error {
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
		if err := reader.Close(); err != nil {
			d.logger.Errorf(err.Error())
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

func (d *docker) getContainerHostConfiguration(r *core.RunnerOptions) *container.HostConfig {
	if err := utils.CreateDirectory(repoSourcePath); err != nil {
		d.logger.Errorf("error creating directory: %v", err)
	}
	specs := getSpces(r.Tier)
	/*
		https://pkg.go.dev/github.com/docker/docker@v20.10.12+incompatible/api/types/container#Resources
		AS per documentation , 1 core = 1e9 NanoCPUs
	*/
	nanoCPU := int64(specs.CPU * nanoCPUUnit)
	d.logger.Infof("Specs %+v", specs)
	return &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: repoSourcePath,
				Target: defaultContainerVolumePath,
			},
			{
				Type:   mount.TypeBind,
				Source: r.HostVolumePath,
				Target: defaultVaultPath,
			},
		},
		AutoRemove:  true,
		SecurityOpt: []string{"seccomp=unconfined"},
		Resources:   container.Resources{Memory: specs.RAM * GB, NanoCPUs: nanoCPU},
	}
}

func (d *docker) getContainerNetworkConfiguration() (*network.NetworkingConfig, error) {
	var networkResource types.NetworkResource
	opts := types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	}
	networkList, err := d.client.NetworkList(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	for _, network := range networkList {
		if network.Name == networkName {
			networkResource = network
		}
	}

	endpointSettings := network.EndpointSettings{
		NetworkID: networkResource.ID,
	}
	networkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	networkConfig.EndpointsConfig[networkName] = &endpointSettings

	return &networkConfig, nil
}

func getSpces(tier core.Tier) core.Specs {
	if val, ok := core.TierOpts[tier]; ok {
		return core.Specs{CPU: val.CPU, RAM: val.RAM}
	}
	return core.TierOpts[core.Small]
}
