package testUtils

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"

	// "fmt"
	"io/ioutil"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

func getCurrentWorkingDir() string {
	_, filename, _, _ := runtime.Caller(1)
	filepath := path.Join(path.Dir(filename), "../")
	fmt.Println(filepath)
	return filepath
}

func GetConfig() (*config.NucleusConfig, error) {
	cwd := getCurrentWorkingDir()
	configJSON, err := ioutil.ReadFile(cwd + ApplicationConfigPath)
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

func GetTaskPayload() (*core.TaskPayload, error) {
	cwd := getCurrentWorkingDir()
	payloadJSON, err := ioutil.ReadFile(cwd + TaskPayloadPath)
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

func GetLogger() (lumber.Logger, error) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func GetPayload() (*core.Payload, error) {
	cwd := getCurrentWorkingDir()
	payloadJSON, err := ioutil.ReadFile(cwd + PayloadPath)
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

func GetGitDiff() map[string]int {
	m := make(map[string]int)
	m["src/steps/resource.ts"] = 3
	return m
}

func GetGitlabCommitDiff() ([]byte, error) {
	cwd := getCurrentWorkingDir()
	data, err := ioutil.ReadFile(cwd + GitlabCommitDiff)
	if err != nil {
		return nil, err
	}
	return data, err
}
