// Package testexecutionservice is used for executing tests
package testexecutionservice

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logstream"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
)

const locatorFile = "locators"

type testExecutionService struct {
	logger      lumber.Logger
	azureClient core.AzureClient
	cfg         *config.NucleusConfig
	ts          *teststats.ProcStats
	execManager core.ExecutionManager
}

// NewTestExecutionService creates and returns a new TestExecutionService instance
func NewTestExecutionService(cfg *config.NucleusConfig,
	execManager core.ExecutionManager,
	azureClient core.AzureClient,
	ts *teststats.ProcStats,
	logger lumber.Logger) core.TestExecutionService {
	return &testExecutionService{cfg: cfg,
		execManager: execManager,
		azureClient: azureClient,
		ts:          ts,
		logger:      logger}
}

// Run executes the test files
func (tes *testExecutionService) Run(ctx context.Context,
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
	multiWriter := io.MultiWriter(logWriter, azureWriter)
	maskWriter := logstream.NewMasker(multiWriter, secretData)

	var target []string
	var envMap map[string]string
	if payload.EventType == core.EventPullRequest {
		target = tasConfig.Premerge.Patterns
		envMap = tasConfig.Premerge.EnvMap
	} else {
		target = tasConfig.Postmerge.Patterns
		envMap = tasConfig.Postmerge.EnvMap
	}
	var args []string
	args = []string{global.FrameworkRunnerMap[tasConfig.Framework], "--command", "execute"}
	if tasConfig.ConfigFile != "" {
		args = append(args, "--config", tasConfig.ConfigFile)
	}
	for _, pattern := range target {
		args = append(args, "--pattern", pattern)
	}

	if payload.LocatorAddress != "" {
		locatorFile, err := tes.GetLocatorsFile(ctx, payload.LocatorAddress)
		tes.logger.Debugf("locators : %v\n", locatorFile)
		if err != nil {
			tes.logger.Errorf("failed to get locator file, error: %v", err)
			return nil, err
		}
		args = append(args, "--locator-file", locatorFile)
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
		// not returning error because runner like jest will return error in case of test failure
		// and we want to run test multiple times
		if err := cmd.Wait(); err != nil {
			tes.logger.Errorf("error in test execution: %+v", err)
		}
		result := <-tes.ts.ExecutionResultOutputChannel
		executionResults.Results = append(executionResults.Results, result.Results...)
	}
	// FIXME:  commenting this out as we will need to rework on coverage logic after test parallelization
	// if collectCoverage {
	// 	if err := tes.createCoverageManifest(tasConfig, coverageDir, removedfiles, executeAll); err != nil {
	// 		tes.logger.Errorf("failed to create manifest file %v", err)
	// 		return nil, err
	// 	}
	// }
	return executionResults, nil
}

// func (tes *testExecutionService) createCoverageManifest(tasConfig *core.TASConfig, coverageDirectory string, removedFiles []string, executeAll bool) error {
// 	manifestFile := core.CoverageManifest{
// 		Removedfiles:     removedFiles,
// 		AllFilesExecuted: executeAll,
// 	}

// 	coverageThreshold := core.CoverageThreshold{
// 		Branches:   tasConfig.CoverageThreshold.Branches,
// 		Lines:      tasConfig.CoverageThreshold.Lines,
// 		Functions:  tasConfig.CoverageThreshold.Functions,
// 		Statements: tasConfig.CoverageThreshold.Statements,
// 		PerFile:    tasConfig.CoverageThreshold.PerFile,
// 	}

// 	if coverageThreshold != (core.CoverageThreshold{}) {
// 		manifestFile.CoverageThreshold = &coverageThreshold
// 	}

// 	manifestPath := filepath.Join(coverageDirectory, global.CoverageManifestFileName)

// 	rawBytes, err := json.Marshal(manifestFile)
// 	if err != nil {
// 		return err
// 	}
// 	return ioutil.WriteFile(manifestPath, rawBytes, 0644)
// }

func (tes *testExecutionService) GetLocatorsFile(ctx context.Context, locatorAddress string) (string, error) {
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
