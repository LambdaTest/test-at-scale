package secrets

import (
	"os"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/tests"
)

var cfg *config.SynapseConfig
var secretsManager core.SecretsManager

const testdDataDir = "./testdata"

func TestMain(m *testing.M) {
	cfg = tests.MockConfig()
	logger, err := lumber.NewLogger(cfg.LogConfig, cfg.Verbose, lumber.InstanceZapLogger)
	// TODO: check proper way to collect error
	if err != nil {
		return
	}

	secretsManager = New(cfg, logger)
	os.Exit(m.Run())
}
