// Package testexecutionservice is used for executing tests
package testexecutionservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logstream"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

const locatorFile = "locators"
const locatorSizeEdgeCase int = 10

const locatorSizeEdgeCase int = 10

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
func (tes *testExecutionService) Run(ctx context.Context,
	testExecutionArgs *core.TestExecutionArgs) (*core.ExecutionResults, error) {
	azureReader, azureWriter := io.Pipe()
	defer azureWriter.Close()

	errChan := testExecutionArgs.LogWriterStrategy.Write(ctx, azureReader)
	defer tes.closeAndWriteLog(azureWriter, errChan)
	logWriter := lumber.NewWriter(tes.logger)
	defer logWriter.Close()
	multiWriter := io.MultiWriter(logWriter, azureWriter)
	maskWriter := logstream.NewMasker(multiWriter, testExecutionArgs.SecretData)

	args, locatorFilePath, err := tes.buildCmdArgs(ctx, testExecutionArgs.TestConfigFile,
		testExecutionArgs.FrameWork, testExecutionArgs.FrameWorkVersion, testExecutionArgs.Payload, testExecutionArgs.TestPattern)
	if err != nil {
		return nil, err
	}

	payload := testExecutionArgs.Payload
	collectCoverage := payload.CollectCoverage
	commandArgs := args
	envVars, err := tes.execManager.GetEnvVariables(testExecutionArgs.EnvMap, testExecutionArgs.SecretData)
	if err != nil {
		tes.logger.Errorf("failed to parse env variables, error: %v", err)
		return nil, err
	}

	locatorArr, err := extractLocators(locatorFilePath, tes.cfg.FlakyTestAlgo, tes.logger)
	if err != nil {
		tes.logger.Errorf("Error in extracting locators from file: %v", err)
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
		if tes.cfg.FlakyTestAlgo == core.RunningXTimesShuffle {
			err := shuffleLocators(locatorArr, locatorFilePath, tes.logger)
			if err != nil {
				tes.logger.Errorf("Error in shuffling locator file %v", err)
			}
		}

		var cmd *exec.Cmd
		if testExecutionArgs.FrameWork == "jasmine" || testExecutionArgs.FrameWork == "mocha" {
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
		cmd.Dir = testExecutionArgs.CWD
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

func (tes *testExecutionService) buildCmdArgs(ctx context.Context,
	testConfigFile string,
	frameWork string,
	frameworkVersion int,
	payload *core.Payload,
	target []string) ([]string, string, error) {
	args := []string{global.FrameworkRunnerMap[frameWork]}

	args = append(args, utils.GetArgs("execute", frameWork, frameworkVersion, testConfigFile, target)...)

	locatorFilePath := ""
	if payload.LocatorAddress != "" {
		locatorFile, err := tes.getLocatorsFile(ctx, payload.LocatorAddress)
		tes.logger.Debugf("locators : %v\n", locatorFile)
		if err != nil {
			tes.logger.Errorf("failed to get locator file, error: %v", err)
			return nil, "", err
		}
		locatorFilePath = locatorFile
		args = append(args, global.ArgLocator, locatorFile)
	}

	return args, locatorFilePath, nil
}

// Read locators from the file and convert it into array of locator config
func extractLocators(locatorFilePath, flakyTestAlgo string, logger lumber.Logger) ([]core.LocatorConfig, error) {
	locatorArrTemp := []core.LocatorConfig{}
	inputLocatorConfigTemp := &core.InputLocatorConfig{}

	if flakyTestAlgo == core.RunningXTimesShuffle {
		content, err := os.ReadFile(locatorFilePath)
		if err != nil {
			logger.Errorf("Error when opening file: ", err)
			return nil, err
		}

		err = json.Unmarshal(content, &inputLocatorConfigTemp)
		if err != nil {
			logger.Errorf("Error during Unmarshal(): ", err)
			return nil, err
		}
		locatorArrTemp = inputLocatorConfigTemp.Locators
	}

	return locatorArrTemp, nil
}

// shuffling order of elements in locator array
func shuffleLocators(locatorArr []core.LocatorConfig, locatorFilePath string, logger lumber.Logger) error {
	locatorArrOrig := make([]core.LocatorConfig, len(locatorArr))
	locatorArrSize := len(locatorArr)

	if locatorArrSize < locatorSizeEdgeCase {
		copy(locatorArrOrig, locatorArr)
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(locatorArrSize, func(i, j int) { locatorArr[i], locatorArr[j] = locatorArr[j], locatorArr[i] })

	// For smaller number probability that random order becomes the original order is high, to handle those edge case
	// we reverse the array if the shuffled order is same as original, For larger size this probability is negligible.
	if locatorArrSize < locatorSizeEdgeCase {
		if reflect.DeepEqual(locatorArrOrig, locatorArr) {
			for i, j := 0, len(locatorArr)-1; i < j; i, j = i+1, j-1 {
				locatorArr[i], locatorArr[j] = locatorArr[j], locatorArr[i]
			}
		}
	}
	inputLocatorConfigTemp := &core.InputLocatorConfig{}
	inputLocatorConfigTemp.Locators = locatorArr
	file, _ := json.Marshal(inputLocatorConfigTemp)
	err := os.WriteFile(locatorFilePath, file, global.FilePermissionWrite)
	if err != nil {
		logger.Errorf("Error While Writing Locators To File ", err)
		return err
	}
	return nil
}
