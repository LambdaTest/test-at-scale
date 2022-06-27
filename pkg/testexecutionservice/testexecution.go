// Package testexecutionservice is used for executing tests
package testexecutionservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logstream"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

const locatorFile = "locators"

type testExecutionService struct {
	logger         lumber.Logger
	azureClient    core.AzureClient
	cfg            *config.NucleusConfig
	ts             *teststats.ProcStats
	execManager    core.ExecutionManager
	requests       core.Requests
	serverEndpoint string
}

// NewTestExecutionService creates and returns a new TestExecutionService instance
func NewTestExecutionService(cfg *config.NucleusConfig,
	requests core.Requests,
	execManager core.ExecutionManager,
	azureClient core.AzureClient,
	ts *teststats.ProcStats,
	logger lumber.Logger) core.TestExecutionService {
	return &testExecutionService{cfg: cfg,
		requests:       requests,
		serverEndpoint: global.NeuronHost + "/report",
		execManager:    execManager,
		azureClient:    azureClient,
		ts:             ts,
		logger:         logger}
}

// Run executes the test files
func (tes *testExecutionService) RunV1(ctx context.Context,
	tasConfig *core.TASConfig,
	payload *core.Payload,
	coverageDir string,
	secretData map[string]string) (*core.ExecutionResults, error) {
	azureReader, azureWriter := io.Pipe()
	defer azureWriter.Close()
	blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, payload.TaskID, core.Execution)
	errChan := tes.execManager.StoreCommandLogs(ctx, blobPath, azureReader)
	defer tes.closeAndWriteLog(azureWriter, errChan)
	logWriter := lumber.NewWriter(tes.logger)
	defer logWriter.Close()
	multiWriter := io.MultiWriter(logWriter, azureWriter)
	maskWriter := logstream.NewMasker(multiWriter, secretData)

	target, envMap := getPatternAndEnvV1(payload, tasConfig)
	args, err := tes.buildCmdArgsV1(ctx, tasConfig, payload, target)
	if err != nil {
		return nil, err
	}

	collectCoverage := payload.CollectCoverage
	commandArgs := args
	envVars, err := tes.execManager.GetEnvVariables(envMap, secretData)
	if err != nil {
		tes.logger.Errorf("failed to parse env variables, error: %v", err)
		return nil, err
	}

	executionResults := &core.ExecutionResults{
		TaskID:   payload.TaskID,
		BuildID:  payload.BuildID,
		RepoID:   payload.RepoID,
		OrgID:    payload.OrgID,
		CommitID: payload.BuildTargetCommit,
		TaskType: payload.TaskType,
	}
	for i := 1; i <= tes.cfg.ConsecutiveRuns; i++ {
		var cmd *exec.Cmd
		if tasConfig.Framework == "jasmine" || tasConfig.Framework == "mocha" {
			if collectCoverage {
				cmd = exec.CommandContext(ctx, "nyc", commandArgs...)
			} else {
				cmd = exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...) //nolint:gosec
			}
		} else {
			cmd = exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...) //nolint:gosec
			if collectCoverage {
				envVars = append(envVars, "TAS_COLLECT_COVERAGE=true")
			}
		}
		cmd.Dir = global.RepoDir
		cmd.Env = envVars
		cmd.Stdout = maskWriter
		cmd.Stderr = maskWriter
		tes.logger.Debugf("Executing test execution command: %s", cmd.String())
		if err := cmd.Start(); err != nil {
			tes.logger.Errorf("failed to execute test %s %v", cmd.String(), err)
			return nil, err
		}
		pid := int32(cmd.Process.Pid)
		tes.logger.Debugf("execution command started with pid %d", pid)

		if err := tes.ts.CaptureTestStats(pid, tes.cfg.CollectStats); err != nil {
			tes.logger.Errorf("failed to find process for command %s with pid %d %v", cmd.String(), pid, err)
			return nil, err
		}
		err := cmd.Wait()
		result := <-tes.ts.ExecutionResultOutputChannel
		if err != nil {
			tes.logger.Errorf("error in test execution: %+v", err)
			// returning error when result is nil to throw execution errors like heap out of memory
			if result == nil {
				return nil, err
			}
		}
		if result != nil {
			executionResults.Results = append(executionResults.Results, result.Results...)
		}
	}
	return executionResults, nil
}

func getPatternAndEnvV1(payload *core.Payload, tasConfig *core.TASConfig) (target []string, envMap map[string]string) {
	if payload.EventType == core.EventPullRequest {
		target = tasConfig.Premerge.Patterns
		envMap = tasConfig.Premerge.EnvMap
	} else {
		target = tasConfig.Postmerge.Patterns
		envMap = tasConfig.Postmerge.EnvMap
	}
	return target, envMap
}

// Run executes the test files
func (tes *testExecutionService) RunV2(ctx context.Context,
	tasConfig *core.TASConfigV2,
	subModule *core.SubModule,
	payload *core.Payload,
	coverageDir string,
	envMap map[string]string,
	target []string,
	secretData map[string]string) (*core.ExecutionResults, error) {
	azureReader, azureWriter := io.Pipe()
	defer azureWriter.Close()
	blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, payload.TaskID, core.Execution)
	errChan := tes.execManager.StoreCommandLogs(ctx, blobPath, azureReader)
	defer tes.closeAndWriteLog(azureWriter, errChan)
	logWriter := lumber.NewWriter(tes.logger)
	multiWriter := io.MultiWriter(logWriter, azureWriter)
	maskWriter := logstream.NewMasker(multiWriter, secretData)

	setModulePath(subModule, envMap)

	args, err := tes.buildCmdArgsV2(ctx, subModule, payload, target)
	if err != nil {
		return nil, err
	}
	collectCoverage := payload.CollectCoverage
	commandArgs := args
	envVars, err := tes.execManager.GetEnvVariables(envMap, secretData)
	if err != nil {
		tes.logger.Errorf("failed to parse env variables, error: %v", err)
		return nil, err
	}

	executionResults := &core.ExecutionResults{
		TaskID:   payload.TaskID,
		BuildID:  payload.BuildID,
		RepoID:   payload.RepoID,
		OrgID:    payload.OrgID,
		CommitID: payload.BuildTargetCommit,
		TaskType: payload.TaskType,
	}
	for i := 1; i <= tes.cfg.ConsecutiveRuns; i++ {
		var cmd *exec.Cmd
		if subModule.Framework == "jasmine" || subModule.Framework == "mocha" {
			if collectCoverage {
				cmd = exec.CommandContext(ctx, "nyc", commandArgs...)
			} else {
				cmd = exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...) //nolint:gosec
			}
		} else {
			cmd = exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...) //nolint:gosec
			if collectCoverage {
				envVars = append(envVars, "TAS_COLLECT_COVERAGE=true")
			}
		}
		cmd.Dir = path.Join(global.RepoDir, subModule.Path)
		cmd.Env = envVars
		cmd.Stdout = maskWriter
		cmd.Stderr = maskWriter
		tes.logger.Debugf("Executing test execution command: %s", cmd.String())
		if err := cmd.Start(); err != nil {
			tes.logger.Errorf("failed to execute test %s %v", cmd.String(), err)
			return nil, err
		}
		pid := int32(cmd.Process.Pid)
		tes.logger.Debugf("execution command started with pid %d", pid)

		if err := tes.ts.CaptureTestStats(pid, tes.cfg.CollectStats); err != nil {
			tes.logger.Errorf("failed to find process for command %s with pid %d %v", cmd.String(), pid, err)
			return nil, err
		}
		// not returning error because runner like jest will return error in case of test failure
		// and we want to run test multiple times
		if err := cmd.Wait(); err != nil {
			tes.logger.Errorf("error in test execution: %+v", err)
		}
		result := <-tes.ts.ExecutionResultOutputChannel
		if result != nil {
			executionResults.Results = append(executionResults.Results, result.Results...)
		}
	}
	return executionResults, nil
}

func setModulePath(subModule *core.SubModule, envMap map[string]string) {
	if path.Join(global.RepoDir, subModule.Path) == global.RepoDir {
		envMap[global.ModulePath] = ""
	} else {
		envMap[global.ModulePath] = subModule.Path
	}
}

func (tes *testExecutionService) SendResults(ctx context.Context,
	payload *core.ExecutionResults) (resp *core.TestReportResponsePayload, err error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		tes.logger.Errorf("failed to marshal request body %v", err)
		return nil, err
	}
	params := utils.FetchQueryParams()
	headers := map[string]string{
		"Authorization": fmt.Sprintf("%s %s", "Bearer", os.Getenv("TOKEN")),
	}
	respBody, _, err := tes.requests.MakeAPIRequest(ctx, http.MethodPost, tes.serverEndpoint, reqBody, params, headers)
	if err != nil {
		tes.logger.Errorf("error while sending reports %v", err)
		return nil, err
	}
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		tes.logger.Errorf("failed to unmarshal response body %v", err)
		return nil, err
	}
	if resp.TaskStatus == "" {
		return nil, errors.New("empty task status")
	}
	return resp, nil
}

func (tes *testExecutionService) getLocatorsFile(ctx context.Context, locatorAddress string) (string, error) {
	u, err := url.Parse(locatorAddress)
	if err != nil {
		return "", err
	}
	// string the container name to get blob path
	blobPath := strings.Replace(u.Path, fmt.Sprintf("/%s/", core.PayloadContainer), "", -1)

	sasURL, err := tes.azureClient.GetSASURL(ctx, blobPath, core.PayloadContainer)
	if err != nil {
		return "", err
	}
	resp, err := tes.azureClient.FindUsingSASUrl(ctx, sasURL)
	if err != nil {
		tes.logger.Errorf("Error while downloading cache for key: %s, error %v", u.Path, err)
		return "", err
	}
	defer resp.Close()

	locatorFilePath := filepath.Join(os.TempDir(), locatorFile)
	out, err := os.Create(locatorFilePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp); err != nil {
		return "", err
	}
	return locatorFilePath, err
}

func (tes *testExecutionService) closeAndWriteLog(azureWriter *io.PipeWriter, errChan <-chan error) {
	azureWriter.Close()
	if err := <-errChan; err != nil {
		tes.logger.Errorf("failed to upload logs for test execution, error: %v", err)
	}
}

func (tes *testExecutionService) buildCmdArgsV1(ctx context.Context,
	tasConfig *core.TASConfig,
	payload *core.Payload,
	target []string) ([]string, error) {
	args := []string{global.FrameworkRunnerMap[tasConfig.Framework]}

	args = append(args, utils.GetArgs("execute", tasConfig, target)...)

	if payload.LocatorAddress != "" {
		locatorFile, err := tes.getLocatorsFile(ctx, payload.LocatorAddress)
		tes.logger.Debugf("locators : %v\n", locatorFile)
		if err != nil {
			tes.logger.Errorf("failed to get locator file, error: %v", err)
			return nil, err
		}

		args = append(args, global.ArgLocator, locatorFile)
	}

	return args, nil
}

func (tes *testExecutionService) buildCmdArgsV2(ctx context.Context,
	subModule *core.SubModule,
	payload *core.Payload,
	target []string) ([]string, error) {
	args := []string{global.FrameworkRunnerMap[subModule.Framework], global.ArgCommand, "execute"}
	if subModule.ConfigFile != "" {
		args = append(args, global.ArgConfig, subModule.ConfigFile)
	}
	for _, pattern := range target {
		args = append(args, global.ArgPattern, pattern)
	}

	if payload.LocatorAddress != "" {
		locatorFile, err := tes.getLocatorsFile(ctx, payload.LocatorAddress)
		tes.logger.Debugf("locators : %v\n", locatorFile)
		if err != nil {
			tes.logger.Errorf("failed to get locator file, error: %v", err)
			return nil, err
		}
		args = append(args, global.ArgLocator, locatorFile)
	}
	return args, nil
}
