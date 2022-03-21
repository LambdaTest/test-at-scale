package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/fileutils"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

const (
	endpointPostTestResults = "http://localhost:9876/results"
)

var endpointPostTestList string
var endpointNeuronReport string

// NewPipeline creates and returns a new Pipeline instance
func NewPipeline(cfg *config.NucleusConfig, logger lumber.Logger) (*Pipeline, error) {
	return &Pipeline{
		Cfg:    cfg,
		Logger: logger,
		HttpClient: http.Client{
			Timeout: 45 * time.Second,
		},
	}, nil
}

//Start starts pipeline lifecycle
func (pl *Pipeline) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var errRemark string
	startTime := time.Now()

	pl.Logger.Debugf("Starting pipeline.....")
	pl.Logger.Debugf("Fetching config")

	endpointPostTestList = global.NeuronHost + "/test-list"
	endpointNeuronReport = global.NeuronHost + "/report"
	// fetch configuration
	payload, err := pl.PayloadManager.FetchPayload(ctx, pl.Cfg.PayloadAddress)
	if err != nil {
		pl.Logger.Fatalf("error while fetching payload: %v", err)
	}

	err = pl.PayloadManager.ValidatePayload(ctx, payload)
	if err != nil {
		pl.Logger.Fatalf("error while validating payload %v", err)
	}

	pl.Logger.Debugf("Payload for current task: %+v \n", *payload)

	if pl.Cfg.CoverageMode {
		if err := pl.CoverageService.MergeAndUpload(ctx, payload); err != nil {
			pl.Logger.Fatalf("error while merge and upload coverage files %v", err)
		}
		os.Exit(0)
	}

	oauth, err := pl.getOauthSecret(payload.RepoID)
	if err != nil {
		pl.Logger.Fatalf("failed to get oauth secret %v", err)
	}

	// set payload on pipeline object
	pl.Payload = payload
	if pl.Cfg.ParseMode {
		err = pl.GitManager.Clone(ctx, payload, oauth.Data.AccessToken)
		if err != nil {
			pl.Logger.Fatalf("failed to clone YML for build ID: %s, error: %v", payload.BuildID, err)
		}
		if err = pl.ParserService.ParseAndValidate(ctx, payload); err != nil {
			pl.Logger.Fatalf("error while parsing YML for build ID: %s, error: %v", payload.BuildID, err)
		}
		os.Exit(0)
	}

	taskPayload := &TaskPayload{
		TaskID:      payload.TaskID,
		BuildID:     payload.BuildID,
		RepoSlug:    payload.RepoSlug,
		RepoLink:    payload.RepoLink,
		OrgID:       payload.OrgID,
		RepoID:      payload.RepoID,
		GitProvider: payload.GitProvider,
		StartTime:   startTime,
		Status:      Running,
	}
	if pl.Cfg.DiscoverMode {
		taskPayload.Type = DiscoveryTask
	} else {
		taskPayload.Type = ExecutionTask
	}

	// marking task to running state
	if err := pl.Task.UpdateStatus(taskPayload); err != nil {
		pl.Logger.Fatalf("failed to update task status %v", err)
	}

	// update task status when pipeline exits
	defer func() {
		taskPayload.EndTime = time.Now()
		if p := recover(); p != nil {
			pl.Logger.Errorf("panic stack trace: %v", p)
			taskPayload.Status = Error
			taskPayload.Remark = errs.GenericErrRemark.Error()
		} else if err != nil {
			if err == context.Canceled {
				taskPayload.Status = Aborted
				taskPayload.Remark = "Task aborted"
			} else {
				taskPayload.Status = Error
				taskPayload.Remark = errRemark
			}
		}
		if err := pl.Task.UpdateStatus(taskPayload); err != nil {
			pl.Logger.Fatalf("failed to update task status %v", err)
		}
	}()

	if pl.Cfg.DiscoverMode {
		pl.Logger.Infof("Cloning repo ...")
		err = pl.GitManager.Clone(ctx, pl.Payload, oauth.Data.AccessToken)
		if err != nil {
			pl.Logger.Errorf("Unable to clone repo '%s': %s", payload.RepoLink, err)
			errRemark = fmt.Sprintf("Unable to clone repo: %s", payload.RepoLink)
			return err
		}
	} else {
		pl.Logger.Debugf("Extracting workspace")
		// Replicate workspace
		if err = pl.CacheStore.ExtractWorkspace(ctx); err != nil {
			pl.Logger.Errorf("Error replicating workspace: %+v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}
	}

	// load tas yaml file
	tasConfig, err := pl.TASConfigManager.LoadConfig(ctx, payload.TasFileName, payload.EventType, false)
	if err != nil {
		pl.Logger.Errorf("Unable to load tas yaml file, error: %v", err)
		errRemark = err.Error()
		return err
	}

	coverageDir := filepath.Join(global.CodeCoverageDir, payload.OrgID, payload.RepoID, payload.BuildTargetCommit)
	cacheKey := fmt.Sprintf("%s/%s/%s", payload.OrgID, payload.RepoID, tasConfig.Cache.Key)

	pl.Logger.Infof("Tas yaml: %+v", tasConfig)

	// set testing taskID, orgID and buildID as environment variable
	os.Setenv("TASK_ID", payload.TaskID)
	os.Setenv("ORG_ID", payload.OrgID)
	os.Setenv("BUILD_ID", payload.BuildID)
	//set target commit_id as environment variable
	os.Setenv("COMMIT_ID", payload.BuildTargetCommit)
	//set repo_id as environment variable
	os.Setenv("REPO_ID", payload.RepoID)
	//set coverage_dir as environment variable
	os.Setenv("CODE_COVERAGE_DIR", coverageDir)
	os.Setenv("BRANCH_NAME", payload.BranchName)
	os.Setenv("ENV", pl.Cfg.Env)
	os.Setenv("TAS_PARALLELISM", strconv.Itoa(tasConfig.Parallelism))
	os.Setenv("ENDPOINT_POST_TEST_LIST", endpointPostTestList)
	os.Setenv("ENDPOINT_POST_TEST_RESULTS", endpointPostTestResults)
	os.Setenv("REPO_ROOT", global.RepoDir)
	os.Setenv("BLOCK_TESTS_FILE", global.BlockTestFileLocation)

	if tasConfig.NodeVersion != nil {
		nodeVersion := tasConfig.NodeVersion.String()
		// Running the `source` commands in a directory where .nvmrc is present, exits with exitCode 3
		// https://github.com/nvm-sh/nvm/issues/1985
		// TODO [good-to-have]: Auto-read and install from .nvmrc file, if present
		commands := []string{
			"source /home/nucleus/.nvm/nvm.sh",
			fmt.Sprintf("nvm install %s", nodeVersion),
		}
		pl.Logger.Infof("Using user-defined node version: %v", nodeVersion)
		err = pl.ExecutionManager.ExecuteInternalCommands(ctx, InstallNodeVer, commands, "", nil, nil)
		if err != nil {
			pl.Logger.Errorf("Unable to install user-defined nodeversion %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", fmt.Sprintf("/home/nucleus/.nvm/versions/node/v%s/bin:%s", nodeVersion, origPath))
	}

	if payload.CollectCoverage {
		if err = fileutils.CreateIfNotExists(coverageDir, true); err != nil {
			pl.Logger.Errorf("failed to create coverage directory %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}
	}

	// read secrets
	secretMap, err := pl.SecretParser.GetRepoSecret(global.RepoSecretPath)
	if err != nil {
		pl.Logger.Errorf("Error in fetching Repo secrets %v", err)
		errRemark = errs.GenericErrRemark.Error()
		return err
	}

	if pl.Cfg.DiscoverMode {
		err = pl.BlockTestService.GetBlockTests(ctx, tasConfig, payload.RepoID, payload.BranchName)
		if err != nil {
			pl.Logger.Errorf("Unable to fetch blocklisted tests: %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}

		if err = pl.CacheStore.Download(ctx, cacheKey); err != nil {
			pl.Logger.Errorf("Unable to download cache: %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}

		if tasConfig.Prerun != nil {
			pl.Logger.Infof("Running pre-run steps")
			err = pl.ExecutionManager.ExecuteUserCommands(ctx, PreRun, payload, tasConfig.Prerun, secretMap)
			if err != nil {
				pl.Logger.Errorf("Unable to run pre-run steps %v", err)
				errRemark = "Error occurred in pre-run steps"
				return err
			}
		}
		err = pl.ExecutionManager.ExecuteInternalCommands(ctx, InstallRunners, global.InstallRunnerCmds, global.RepoDir, nil, nil)
		if err != nil {
			pl.Logger.Errorf("Unable to install custom runners %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}

		pl.Logger.Infof("Identifying changed files ...")
		diffExists := true
		diff, err := pl.DiffManager.GetChangedFiles(ctx, payload, oauth.Data.AccessToken)
		if err != nil {
			if errors.Is(err, errs.ErrGitDiffNotFound) {
				diffExists = false
			} else {
				pl.Logger.Errorf("Unable to identify changed files %s", err)
				errRemark = "Error occurred in fetching diff from GitHub"
				return err
			}
		}

		// discover test cases
		err = pl.TestDiscoveryService.Discover(ctx, tasConfig, pl.Payload, secretMap, diff, diffExists)
		if err != nil {
			pl.Logger.Errorf("Unable to perform test discovery: %+v", err)
			errRemark = "Error occurred in discovering tests"
			return err
		}
		// mark status as passed
		taskPayload.Status = Passed

		pl.Logger.Debugf("Caching workspace")
		// Persist workspace
		if err = pl.CacheStore.CacheWorkspace(ctx); err != nil {
			pl.Logger.Errorf("Error caching workspace: %+v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}

		// Upload cache once for other builds
		if err = pl.CacheStore.Upload(ctx, cacheKey, tasConfig.Cache.Paths...); err != nil {
			pl.Logger.Errorf("Unable to upload cache: %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}
		pl.Logger.Debugf("Cache uploaded successfully")
	}

	if pl.Cfg.ExecuteMode {

		pl.Logger.Debugf("execute Mode")
		// execute test cases
		executionResults, err := pl.TestExecutionService.Run(ctx, tasConfig, pl.Payload, coverageDir, secretMap)
		if err != nil {
			pl.Logger.Infof("Unable to perform test execution: %v", err)
			errRemark = "Error occurred in executing tests"
			return err
		}

		if err = pl.sendStats(*executionResults); err != nil {
			pl.Logger.Errorf("error while sending test reports %v", err)
			errRemark = errs.GenericErrRemark.Error()
			return err
		}

		taskPayload.Status = findTaskPayloadStatus(executionResults)

		if tasConfig.Postrun != nil {
			pl.Logger.Infof("Running post-run steps")
			err = pl.ExecutionManager.ExecuteUserCommands(ctx, PostRun, payload, tasConfig.Postrun, secretMap)
			if err != nil {
				pl.Logger.Errorf("Unable to run post-run steps %v", err)
				errRemark = "Error occurred in post-run steps"
				return err
			}
		}
	}
	pl.Logger.Debugf("Completed pipeline")

	return nil
}

func findTaskPayloadStatus(executionResults *ExecutionResults) Status {
	for _, result := range executionResults.Results {
		for i := 0; i < len(result.TestPayload); i++ {
			testResult := &result.TestPayload[i]
			if testResult.Status == "failed" {
				return Failed
			}
		}
	}
	return Passed
}

func (pl *Pipeline) sendStats(payload ExecutionResults) error {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		pl.Logger.Errorf("failed to marshal request body %v", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, endpointNeuronReport, bytes.NewBuffer(reqBody))
	if err != nil {
		pl.Logger.Errorf("failed to create new request %v", err)
		return err
	}

	resp, err := pl.HttpClient.Do(req)

	if err != nil {
		pl.Logger.Errorf("error while sending reports %v", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pl.Logger.Errorf("error while sending reports, non 200 status")
		return errors.New("non 200 status")
	}
	return nil
}

// getOauthSecret returns a valid oauth token
func (pl *Pipeline) getOauthSecret(repoID string) (*Oauth, error) {
	oauth, err := pl.SecretParser.GetOauthSecret(global.OauthSecretPath)
	if err != nil {
		return nil, err
	}

	if !pl.SecretParser.Expired(oauth) {
		return oauth, nil
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", global.NeuronHost+global.RefreshTokenEndpoint, repoID), nil)
	if err != nil {
		pl.Logger.Errorf("failed to create new request %v", err)
	}

	res, err := pl.HttpClient.Do(req)
	if err != nil {
		pl.Logger.Errorf("error while refreshing token for RepoID %s : %s", repoID, err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		pl.Logger.Errorf("error while reading token response body for RepoID %s : %s", repoID, err)
	}

	refreshedOauth := new(Oauth)
	err = json.Unmarshal(body, &refreshedOauth)
	if err != nil {
		pl.Logger.Errorf("error while unmarshaling json to oauth for RepoID %s : %s", repoID, err)
	}

	return refreshedOauth, nil
}
