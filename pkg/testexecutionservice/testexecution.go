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

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/logstream"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/service/teststats"
)

const locatorFile = "locators"

type testExecutionService struct {
	logger      lumber.Logger
	azureClient core.AzureClient
	ts          *teststats.ProcStats
	execManager core.ExecutionManager
}

// NewTestExecutionService creates and returns a new TestExecutionService instance
func NewTestExecutionService(execManager core.ExecutionManager,
	azureClient core.AzureClient,
	ts *teststats.ProcStats,
	logger lumber.Logger) core.TestExecutionService {
	return &testExecutionService{execManager: execManager,
		azureClient: azureClient,
		ts:          ts,
		logger:      logger}
}

// Run executes the test files
func (tes *testExecutionService) Run(ctx context.Context,
	tasConfig *core.TASConfig,
	payload *core.Payload,
	coverageDir string,
	secretData map[string]string) (*core.ExecutionResult, error) {

	azureReader, azureWriter := io.Pipe()
	defer azureWriter.Close()
	blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, payload.TaskID, core.Execution)
	errChan := tes.execManager.StoreCommandLogs(ctx, blobPath, azureReader)
	logWriter := lumber.NewWriter(tes.logger)
	defer logWriter.Close()
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
		if err != nil {
			tes.logger.Errorf("failed to get locator file, error: %v", err)
			return nil, err
		}
		args = append(args, "--locator-file", locatorFile)
	}
	// use locators only if there is no locator address
	if payload.Locators != "" && payload.LocatorAddress == "" {
		locators := strings.Split(payload.Locators, global.TestLocatorsDelimiter)
		for _, locator := range locators {
			if locator != "" {
				args = append(args, "--locator", locator)
			}
		}
	}
	collectCoverage := payload.CollectCoverage
	testResults := make([]core.TestPayload, 0)
	testSuiteResults := make([]core.TestSuitePayload, 0)

	commandArgs := args
	envVars, err := tes.execManager.GetEnvVariables(envMap, secretData)
	if err != nil {
		tes.logger.Errorf("failed to parsed env variables, error: %v", err)
		return nil, err
	}
	var cmd *exec.Cmd
	if tasConfig.Framework == "jasmine" || tasConfig.Framework == "mocha" {
		if collectCoverage {
			cmd = exec.CommandContext(ctx, "nyc", commandArgs...)
		} else {
			cmd = exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...)
		}
	} else {
		cmd = exec.CommandContext(ctx, commandArgs[0], commandArgs[1:]...)
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

	if err := tes.ts.CaptureTestStats(pid); err != nil {
		tes.logger.Errorf("failed to find process for command %s with pid %d %v", cmd.String(), pid, err)
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		tes.logger.Errorf("Error in executing []: %+v\n", err)
		return nil, err
	}
	execResultsWithStats := <-tes.ts.ExecutionResultOutputChannel
	testResults = append(testResults, execResultsWithStats.TestPayload...)
	testSuiteResults = append(testSuiteResults, execResultsWithStats.TestSuitePayload...)

	// FIXME:  commenting this out as we will need to rework on coverage logic after test parallelization
	// if collectCoverage {
	// 	if err := tes.createCoverageManifest(tasConfig, coverageDir, removedfiles, executeAll); err != nil {
	// 		tes.logger.Errorf("failed to create manifest file %v", err)
	// 		return nil, err
	// 	}
	// }
	azureWriter.Close()
	if uploadErr := <-errChan; uploadErr != nil {
		tes.logger.Errorf("failed to upload logs for test execution, error: %v", uploadErr)
		return nil, uploadErr
	}
	return &core.ExecutionResult{
		OrgID:            payload.OrgID,
		RepoID:           payload.RepoID,
		BuildID:          payload.BuildID,
		TaskID:           payload.TaskID,
		CommitID:         payload.BuildTargetCommit,
		TestPayload:      testResults,
		TestSuitePayload: testSuiteResults,
	}, nil
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
