package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	errs "github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

type secertManager struct {
	logger lumber.Logger
	cfg    *config.SynapseConfig
}

// New returns new secretManager
func New(cfg *config.SynapseConfig, logger lumber.Logger) core.SecretsManager {
	return &secertManager{
		logger: logger,
		cfg:    cfg,
	}
}

func (s *secertManager) GetLambdatestSecrets() *config.LambdatestConfig {
	return &s.cfg.Lambdatest
}

// GetSynapseName returns the name of synapse if mentioned in config
func (s *secertManager) GetSynapseName() string {
	return s.cfg.Name
}

func (s *secertManager) WriteGitSecrets(path string) error {
	gitSecrets := core.Secret{
		"access_token":  s.cfg.Git.Token,
		"expiry":        "0001-01-01T00:00:00Z",
		"refresh_token": "",
		"token_type":    s.cfg.Git.TokenType,
	}
	gitSecretsJSON, err := json.Marshal(gitSecrets)
	if err != nil {
		return errs.ERR_JSON_MAR(err.Error())
	}

	if err := utils.CreateDirectory(path); err != nil {
		return err
	}

	if err := utils.WriteFileToDirectory(path, global.GitConfigFileName, gitSecretsJSON); err != nil {
		return err
	}

	return nil
}

func (s *secertManager) WriteRepoSecrets(repo string, path string) error {
	val, ok := s.cfg.RepoSecrets[repo]
	if !ok {
		return errors.New("no secrets found in configuration file")
	}

	repoSecretsJSON, err := json.Marshal(val)
	if err != nil {
		return errs.ERR_JSON_MAR(err.Error())
	}

	if err := utils.CreateDirectory(path); err != nil {
		return err
	}

	if err := utils.WriteFileToDirectory(path, global.RepoSecretsFileName, repoSecretsJSON); err != nil {
		return err
	}

	return nil
}

func (s *secertManager) GetDockerSecrets(r *core.RunnerOptions) (core.ContainerImageConfig, error) {
	containerImageConfig := core.ContainerImageConfig{}
	containerImageConfig.Mode = s.cfg.ContainerRegistry.Mode
	containerImageConfig.Image = r.DockerImage
	containerImageConfig.PullPolicy = s.cfg.ContainerRegistry.PullPolicy
	/*
		In parsing mode use default public container
	*/
	if r.PodType != core.NucleusPod {
		return containerImageConfig, nil
	}
	/*
			1. if mode is public then no need to build AuthRegistry
		 	2. PullPolicy is set to never, then we assume docker image is being pulled manually by user
	*/
	if s.cfg.ContainerRegistry.Mode == config.PublicMode || s.cfg.ContainerRegistry.PullPolicy == config.PullNever {
		return containerImageConfig, nil
	}
	// for private repo check whether creds are empty
	if s.cfg.ContainerRegistry.Username == "" || s.cfg.ContainerRegistry.Password == "" {
		return containerImageConfig, errs.CR_AUTH_NF
	}
	jsonBytes, _ := json.Marshal(map[string]string{
		"username": s.cfg.ContainerRegistry.Username,
		"password": s.cfg.ContainerRegistry.Password,
	})
	containerImageConfig.AuthRegistry = base64.StdEncoding.EncodeToString(jsonBytes)
	return containerImageConfig, nil
}
