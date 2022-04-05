package docker

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/synapse"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
)

const (
	defaultVaultPath = "/vault/secrets"
	repoSourcePath   = "/tmp/synapse/%s/nucleus"
	nanoCPUUnit      = 1e9
	// GB defines number of bytes in 1 GB
	GB int64 = 1e+9
)

func (d *docker) getContainerConfiguration(r *core.RunnerOptions) *container.Config {

	return &container.Config{
		Image:   r.DockerImage,
		Env:     r.Env,
		Tty:     false,
		Cmd:     r.ContainerArgs,
		Volumes: make(map[string]struct{}),
	}
}

func (d *docker) getContainerHostConfiguration(r *core.RunnerOptions) *container.HostConfig {
	specs := getSpces(r.Tier)
	/*
		https://pkg.go.dev/github.com/docker/docker@v20.10.12+incompatible/api/types/container#Resources
		AS per documentation , 1 core = 1e9 NanoCPUs
	*/
	nanoCPU := int64(specs.CPU * nanoCPUUnit)
	d.logger.Infof("Specs %+v", specs)
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: r.HostVolumePath,
			Target: defaultVaultPath,
		},
	}
	if r.PodType == core.NucleusPod || r.PodType == core.CoveragePod {
		repoBuildSourcePath := fmt.Sprintf(repoSourcePath, r.Label[synapse.BuildID])
		if err := utils.CreateDirectory(repoBuildSourcePath); err != nil {
			d.logger.Errorf("error creating directory: %v", err)
		}
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: repoBuildSourcePath,
			Target: global.WorkspaceCacheDir,
		})
	}
	hostConfig := container.HostConfig{
		Mounts:      mounts,
		AutoRemove:  true,
		SecurityOpt: []string{"seccomp=unconfined"},
		Resources:   container.Resources{Memory: specs.RAM * GB, NanoCPUs: nanoCPU},
	}

	autoRemove, err := strconv.ParseBool(os.Getenv(global.AutoRemoveEnv))
	if err != nil {
		d.logger.Errorf("Error reading os env AutoRemove with error: %v \n returning default host config", err)
		return &hostConfig

	}
	hostConfig.AutoRemove = autoRemove
	return &hostConfig
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
