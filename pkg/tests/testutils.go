package tests

import (
	"github.com/LambdaTest/test-at-scale/config"
)

// MockConfig creates new dummy config
func MockConfig() *config.SynapseConfig {
	cfg := config.SynapseConfig{
		LogFile: "./synapsetest.go",
		Verbose: true,
		Lambdatest: config.LambdatestConfig{
			SecretKey: "dummysecretkey",
		},
		Git: config.GitConfig{
			Token:     "dummytoken",
			TokenType: "Bearer",
		},
		ContainerRegistry: config.ContainerRegistryConfig{
			Mode:       config.PublicMode,
			PullPolicy: config.PullAlways,
		},
	}
	return &cfg
}
