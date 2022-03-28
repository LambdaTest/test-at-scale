package config

import "github.com/LambdaTest/synapse/pkg/lumber"

// Model definition for configuration

// SynapseConfig the application's configuration
type SynapseConfig struct {
	Config            string
	LogFile           string
	LogConfig         lumber.LoggingConfig
	Env               string
	Verbose           bool
	Lambdatest        LambdatestConfig
	Git               GitConfig
	ContainerRegistry ContainerRegistryConfig
	RepoSecrets       map[string]map[string]string
}

// LambdatestConfig contains credentials for lambdatest
type LambdatestConfig struct {
	SecretKey string
}

// GitConfig contains git token
type GitConfig struct {
	Token     string
	TokenType string
}

// PullPolicyType defines when to pull docker image
type PullPolicyType string

// ModeType define type of container repo
type ModeType string

// ContainerRegistryConfig contains repo configuration if private repo is used
type ContainerRegistryConfig struct {
	PullPolicy PullPolicyType
	Mode       ModeType
	Username   string
	Password   string
}

// defines constant for docker config
const (
	PullAlways  PullPolicyType = "always"
	PullNever   PullPolicyType = "never"
	PrivateMode ModeType       = "private"
	PublicMode  ModeType       = "public"
)
