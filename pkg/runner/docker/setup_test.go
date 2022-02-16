package docker

import (
	"os"
	"testing"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/secrets"
	"github.com/LambdaTest/synapse/pkg/tests"
)

var cfg *config.SynapseConfig
var secretsManager core.SecretsManager
var runner core.DockerRunner

func TestMain(m *testing.M) {
	cfg = tests.MockConfig()
	logger, err := lumber.NewLogger(cfg.LogConfig, cfg.Verbose, lumber.InstanceZapLogger)
	// TODO: check proper way to collect error
	if err != nil {
		return
	}

	secretsManager = secrets.New(cfg, logger)
	runner, err = New(secretsManager, logger, cfg)
	if err != nil {
		logger.Errorf("error in configuring docker client")
	}
	os.Exit(m.Run())
}
