package docker

import (
	"context"
	"os"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/secrets"
	"github.com/LambdaTest/test-at-scale/pkg/tests"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var cfg *config.SynapseConfig
var secretsManager core.SecretsManager
var runner core.DockerRunner

func createNetworkIfNotExists(dockerClient *client.Client, networkName string) error {
	opts := types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	}
	networkList, err := dockerClient.NetworkList(context.TODO(), opts)
	if err != nil {
		return err
	}
	for idx := 0; idx < len(networkList); idx++ {
		if networkList[idx].Name == networkName {
			return nil
		}
	}
	if _, err := dockerClient.NetworkCreate(context.TODO(), networkName, types.NetworkCreate{
		Internal: true,
	}); err != nil {
		return err
	}
	return nil
}

func deletNetworkIfExists(dockerClient *client.Client, networkName string) error {
	ctx := context.TODO()
	opts := types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	}
	networkList, err := dockerClient.NetworkList(ctx, opts)
	if err != nil {
		return err
	}
	for idx := 0; idx < len(networkList); idx++ {
		if networkList[idx].Name == networkName {
			return dockerClient.NetworkRemove(ctx, networkName)
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	networkName := "dummy-network"
	os.Setenv(global.NetworkEnvName, networkName)
	cfg = tests.MockConfig()

	logger, err := lumber.NewLogger(cfg.LogConfig, cfg.Verbose, lumber.InstanceZapLogger)
	// TODO: check proper way to collect error
	if err != nil {
		os.Exit(1)
	}
	cl, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		os.Exit(1)
	}

	if errC := createNetworkIfNotExists(cl, networkName); errC != nil {
		logger.Errorf("Error in creating network %s", networkName)
		os.Exit(1)
	}
	secretsManager = secrets.New(cfg, logger)
	runner, err = New(secretsManager, logger, cfg)
	if err != nil {
		logger.Errorf("error in configuring docker client")
		os.Exit(1)
	}
	exitCode := m.Run()
	if err := deletNetworkIfExists(cl, networkName); err != nil {
		logger.Errorf("Error in deleting network %s", networkName)
		os.Exit(1)
	}
	os.Exit(exitCode)
}
