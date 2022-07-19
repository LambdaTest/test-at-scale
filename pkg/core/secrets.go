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

	// GetDockerSecrets returns Mode , RegistryAuth, and URL for pulling remote docker image
	GetDockerSecrets(r *RunnerOptions) (ContainerImageConfig, error)

	// GetSynapseName returns synapse name mentioned in config
	GetSynapseName() string
	// GetOauthToken returns oauth token
	GetOauthToken() *Oauth

	// GetGitSecretBytes get git secrets in bytes
	GetGitSecretBytes() ([]byte, error)

	// GetRepoSecretBytes get repo secrets in bytes
	GetRepoSecretBytes(repo string) ([]byte, error)
}
