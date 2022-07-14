// Package testdiscoveryservice is used for discover tests
package testdiscoveryservice

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logstream"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

type testDiscoveryService struct {
	logger            lumber.Logger
	execManager       core.ExecutionManager
	tdResChan         chan core.DiscoveryResult
	requests          core.Requests
	discoveryEndpoint string
}

// NewTestDiscoveryService creates and returns a new testDiscoveryService instance
func NewTestDiscoveryService(ctx context.Context,
	tdResChan chan core.DiscoveryResult,
	execManager core.ExecutionManager,
	requests core.Requests,
	logger lumber.Logger) core.TestDiscoveryService {
	return &testDiscoveryService{
		logger:            logger,
		execManager:       execManager,
		tdResChan:         tdResChan,
		requests:          requests,
		discoveryEndpoint: global.NeuronHost + "/test-list",
	}
}

func (tds *testDiscoveryService) Discover(ctx context.Context, discoveryArgs *core.DiscoveyArgs) (*core.DiscoveryResult, error) {
	configFilePath, err := utils.GetConfigFileName(discoveryArgs.Payload.TasFileName)
	if err != nil {
		return nil, err
	}
	impactAll := tds.shouldImpactAll(discoveryArgs.SmartRun, configFilePath, discoveryArgs.Diff)

	args := utils.GetArgs("discover", discoveryArgs.FrameWork, discoveryArgs.FrameWorkVersion,
		discoveryArgs.TestConfigFile, discoveryArgs.TestPattern)

	if !impactAll {
		if len(discoveryArgs.Diff) == 0 && discoveryArgs.DiffExists {
			// empty diff; in PR, a commit added and then reverted to cause an overall empty PR diff
			args = append(args, global.ArgDiff)
		} else {
			for k, v := range discoveryArgs.Diff {
				// in changed files we only have added or modified files.
				if v != core.FileRemoved {
					args = append(args, global.ArgDiff, k)
				}
			}
		}
	}
	tds.logger.Debugf("Discovering tests at paths %+v", discoveryArgs.TestPattern)

	cmd := exec.CommandContext(ctx, global.FrameworkRunnerMap[discoveryArgs.FrameWork], args...) //nolint:gosec
	cmd.Dir = discoveryArgs.CWD
	envVars, err := tds.execManager.GetEnvVariables(discoveryArgs.EnvMap, discoveryArgs.SecretData)
	if err != nil {
		tds.logger.Errorf("failed to parse env variables, error: %v", err)
		return nil, err
	}
	cmd.Env = envVars
	logWriter := lumber.NewWriter(tds.logger)
	defer logWriter.Close()
	maskWriter := logstream.NewMasker(logWriter, discoveryArgs.SecretData)
	cmd.Stdout = maskWriter
	cmd.Stderr = maskWriter

	tds.logger.Debugf("Executing test discovery command: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		tds.logger.Errorf("command %s of type %s failed with error: %v", cmd.String(), core.Discovery, err)
		return nil, err
	}

	testDiscoveryResult := <-tds.tdResChan
	return &testDiscoveryResult, nil
}

func (tds *testDiscoveryService) shouldImpactAll(smartRun bool, configFilePath string, diff map[string]int) bool {
	impactAll := !smartRun
	if _, ok := diff[configFilePath]; ok {
		impactAll = true
	}
	for diffFile := range diff {
		if strings.HasSuffix(diffFile, global.PackageJSON) {
			impactAll = true
			break
		}
	}
	return impactAll
}

func (tds *testDiscoveryService) SendResult(ctx context.Context, testDiscoveryResult *core.DiscoveryResult) error {
	reqBody, err := json.Marshal(testDiscoveryResult)
	if err != nil {
		tds.logger.Errorf("error while json marshal %v", err)
		return err
	}
	query, headers := utils.GetDefaultQueryAndHeaders()
	if _, _, err := tds.requests.MakeAPIRequest(ctx, http.MethodPost, tds.discoveryEndpoint, reqBody, query, headers); err != nil {
		return err
	}

	return nil
}
