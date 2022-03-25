package testutils

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"runtime"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/lumber"
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

// GetConfig returns a dummy NucleusConfig using the json file pointed by ApplicationConfigPath. It returns error if it is unable to ReadFile from the provided location or if it is unable to Unmarshal the file contents
func GetConfig() (*config.NucleusConfig, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	configJSON, err := ioutil.ReadFile(cwd + ApplicationConfigPath) // AplicationConfigPath points to dummy config file for NucleusConfig
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

// GetTaskPayload returns a dummy core.TaskPayload using the json file pointed by TaskPayloadPath. It returns error if unable to readfile or if unable to unmarshal file components
func GetTaskPayload() (*core.TaskPayload, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	payloadJSON, err := ioutil.ReadFile(cwd + TaskPayloadPath) // TaskPayloadPath points to json file containing dummy TaskPayload
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

// GetLogger returns a dummy lumber.Logger. It returns error if it is unable establish logger using lumber.NewLogger function
func GetLogger() (lumber.Logger, error) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// GetPayload returns a dummy core.Payload using the json file pointed by PayloadPath. It returns error if unable to readfile or if unable to unmarshal file components
func GetPayload() (*core.Payload, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	payloadJSON, err := ioutil.ReadFile(cwd + PayloadPath) // PayloadPath points to json file containing dummy PayloadPath
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

// GetGitlabCommitDiff returns a dummy GitlabCommitDiff as slice of byte data. Itreturns error if unable to readfile
func GetGitlabCommitDiff() ([]byte, error) {
	cwd, err := getCurrentWorkingDir()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(cwd + GitlabCommitDiff) // GitLabCommitDiff points to json file containing dummy GitLabCommitDiff
	if err != nil {
		return nil, err
	}
	return data, err
}
