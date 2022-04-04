package core

import "github.com/LambdaTest/test-at-scale/config"

// Secret struct for holding secret data
type Secret map[string]string

// VaultSecret holds secrets in vault format
type VaultSecret struct {
	Secrets Secret `json:"data"`
}

// SecretsManager defines operation for secrets
type SecretsManager interface {
	// GetLambdatestSecrets returns lambdatest config
	GetLambdatestSecrets() *config.LambdatestConfig

	// WriteGitSecrets writes git secrets to file
	WriteGitSecrets(path string) error

	// WriteRepoSecrets writes repo secrets to file
	WriteRepoSecrets(repo string, path string) error

	// GetDockerSecrets returns Mode , RegistryAuth, and URL for pulling remote docker image
	GetDockerSecrets(r *RunnerOptions) (ContainerImageConfig, error)
	// GetSynapseName returns synapse name mentioned in config
	GetSynapseName() string
}
