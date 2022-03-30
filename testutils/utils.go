package testutils

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

// getCurrentWorkingDir give the file path of this file
func getCurrentWorkingDir() string {
	_, filename, _, _ := runtime.Caller(1)
	filepath := path.Join(path.Dir(filename), "../")
	fmt.Println(filepath)
	return filepath
}

// GetConfig returns a dummy NucleusConfig using the json file pointed by ApplicationConfigPath. It returns error if it is unable to ReadFile from the provided location or if it is unable to Unmarshal the file contents
func GetConfig() (*config.NucleusConfig, error) {
	cwd := getCurrentWorkingDir()
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

// GetTaskPayload returns a dummy core.TaskPayload using the json file pointed by TaskPayloadPath. It returns error if unable to readfile or if unable to unmarshal file components
func GetTaskPayload() (*core.TaskPayload, error) {
	cwd := getCurrentWorkingDir()
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
	cwd := getCurrentWorkingDir()
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

// GetGitDiff returns a dummy map[string]int for testing purpose.
func GetGitDiff() map[string]int {
	m := make(map[string]int)
	m["src/steps/resource.ts"] = 3
	return m
}

// GetGitlabCommitDiff returns a dummy GitlabCommitDiff as slice of byte data. Itreturns error if unable to readfile
func GetGitlabCommitDiff() ([]byte, error) {
	cwd := getCurrentWorkingDir()
	data, err := os.ReadFile(cwd + GitlabCommitDiff) // GitLabCommitDiff points to json file containing dummy GitLabCommitDiff
	if err != nil {
		return nil, err
	}
	return data, err
}

func LoadFile(relativePath string) ([]byte, error) {
	absPath := fmt.Sprintf("%s/%s", getCurrentWorkingDir(), relativePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return data, err
}
