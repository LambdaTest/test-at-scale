package config

import "github.com/LambdaTest/synapse/pkg/lumber"

// Model definition for configuration

// NucleusConfig is the application's configuration
type NucleusConfig struct {
	Config         string
	Port           string
	PayloadAddress string `json:"payloadAddress" yaml:"payloadAddress"`
	LogFile        string
	LogConfig      lumber.LoggingConfig
	CoverageMode   bool   `json:"coverage" yaml:"coverageOnly"`
	DiscoverMode   bool   `json:"discover" yaml:"discoverOnly"`
	ExecuteMode    bool   `json:"execute" yaml:"executeOnly"`
	TaskID         string `json:"taskID" env:"TASK_ID"`
	BuildID        string `json:"buildID" env:"BUILD_ID"`
	Locators       string `json:"locators"`
	LocatorAddress string `json:"locatorAddress"`
	Env            string
	Verbose        bool
	Azure          Azure  `env:"AZURE"`
	LocalRunner    bool   `env:"local"`
	SynapseHost    string `env:"synapsehost"`
}

// Azure providers the storage configuration.
type Azure struct {
	ContainerName      string `env:"CONTAINER_NAME"`
	StorageAccountName string `env:"STORAGE_ACCOUNT"`
	StorageAccessKey   string `env:"STORAGE_ACCESS_KEY"`
}
