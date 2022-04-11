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

func createNetworkIfNotExists(client *client.Client, networkName string) error {
	opts := types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	}
	networkList, err := client.NetworkList(context.TODO(), opts)
	if err != nil {
		return err
	}
	for _, network := range networkList {
		if network.Name == networkName {
			return nil
		}
	}
	if _, err := client.NetworkCreate(context.TODO(), networkName, types.NetworkCreate{
		Internal: true,
	}); err != nil {
		return err
	}
	return nil
}

func deletNetworkIfExists(client *client.Client, networkName string) error {
	ctx := context.TODO()
	opts := types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkName)),
	}
	networkList, err := client.NetworkList(ctx, opts)
	if err != nil {
		return err
	}
	for _, network := range networkList {
		if network.Name == networkName {
			return client.NetworkRemove(ctx, networkName)
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
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		os.Exit(1)
	}

	if err := createNetworkIfNotExists(client, networkName); err != nil {
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
	if err := deletNetworkIfExists(client, networkName); err != nil {
		logger.Errorf("Error in deleting network %s", networkName)
		os.Exit(1)
	}
	os.Exit(exitCode)
}
