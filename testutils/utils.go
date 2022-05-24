package testutils

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

// getCurrentWorkingDir give the file path of this file
func getCurrentWorkingDir() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", errs.New("runtime.Calller(1) was unable to recover information")
	}
	filepath := path.Join(path.Dir(filename), "../")
	return filepath, nil
}

// GetConfig returns a dummy NucleusConfig using the json file pointed by ApplicationConfigPath
func GetConfig() (*config.NucleusConfig, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	configJSON, err := os.ReadFile(cwd + ApplicationConfigPath) // AplicationConfigPath points to dummy config file for NucleusConfig
	if err != nil {
		return nil, err
	}
	var tasConfig *config.NucleusConfig
	err = json.Unmarshal(configJSON, &tasConfig)
	if err != nil {
		return nil, err
	}
	return tasConfig, nil
}

// GetTaskPayload returns a dummy core.TaskPayload using the json file pointed by TaskPayloadPath
func GetTaskPayload() (*core.TaskPayload, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	payloadJSON, err := os.ReadFile(cwd + TaskPayloadPath) // TaskPayloadPath points to json file containing dummy TaskPayload
	if err != nil {
		return nil, err
	}
	var p *core.TaskPayload
	err = json.Unmarshal(payloadJSON, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetLogger returns a dummy lumber.Logger.
func GetLogger() (lumber.Logger, error) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// GetPayload returns a dummy core.Payload using the json file pointed by PayloadPath.
func GetPayload() (*core.Payload, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	payloadJSON, err := os.ReadFile(cwd + PayloadPath) // PayloadPath points to json file containing dummy PayloadPath
	if err != nil {
		return nil, err
	}
	var p *core.Payload
	err = json.Unmarshal(payloadJSON, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetGitlabCommitDiff returns a dummy GitlabCommitDiff as slice of byte data.
func GetGitlabCommitDiff() ([]byte, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(cwd + GitlabCommitDiff) // GitLabCommitDiff points to json file containing dummy GitLabCommitDiff
	if err != nil {
		return nil, err
	}
	return data, err
}

func LoadFile(relativePath string) ([]byte, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	absPath := fmt.Sprintf("%s/%s", cwd, relativePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return data, err
}
