package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/fileutils"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

const (
	endpointPostTestResults = "http://localhost:9876/results"
	endpointPostTestList    = "http://localhost:9876/test-list"
	languageJs              = "javascript"
)

// NewPipeline creates and returns a new Pipeline instance
func NewPipeline(cfg *config.NucleusConfig, logger lumber.Logger) (*Pipeline, error) {
	return &Pipeline{
		Cfg:    cfg,
		Logger: logger,
	}, nil
}

// Start starts pipeline lifecycle
func (pl *Pipeline) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	startTime := time.Now()

	pl.Logger.Debugf("Starting pipeline.....")
	pl.Logger.Debugf("Fetching config")

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
		if err = pl.CoverageService.MergeAndUpload(ctx, payload); err != nil {
			pl.Logger.Fatalf("error while merge and upload coverage files %v", err)
		}
		os.Exit(0)
	}

	// set payload on pipeline object
	pl.Payload = payload

	taskPayload := pl.getTaskPayload(payload, startTime)
	payload.TaskType = taskPayload.Type
	pl.Logger.Infof("Running nucleus in %s mode", taskPayload.Type)

	go func() {
		// marking task to running state
		if err = pl.Task.UpdateStatus(context.Background(), taskPayload); err != nil {
			pl.Logger.Fatalf("failed to update task status %v", err)
		}
	}()

	// update task status when pipeline exits
	defer func() {
		taskPayload.EndTime = time.Now()
		if p := recover(); p != nil {
			pl.Logger.Errorf("panic stack trace: %v\n%s", p, string(debug.Stack()))
			taskPayload.Status = Error
			taskPayload.Remark = errs.GenericErrRemark.Error()
		} else if err != nil {
			if errors.Is(err, context.Canceled) {
				taskPayload.Status = Aborted
				taskPayload.Remark = "Task aborted"
			} else {
				if _, ok := err.(*errs.StatusFailed); ok {
					taskPayload.Status = Failed
				} else {
					taskPayload.Status = Error
				}
				taskPayload.Remark = err.Error()
			}
		}
		if err = pl.Task.UpdateStatus(context.Background(), taskPayload); err != nil {
			pl.Logger.Fatalf("failed to update task status %v", err)
		}
	}()

	oauth, err := pl.SecretParser.GetOauthSecret(global.OauthSecretPath)
	if err != nil {
		pl.Logger.Errorf("failed to get oauth secret %v", err)
		return err
	}
	// read secrets
	secretMap, err := pl.SecretParser.GetRepoSecret(global.RepoSecretPath)
	if err != nil {
		pl.Logger.Errorf("Error in fetching Repo secrets %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	if pl.Cfg.DiscoverMode {
		pl.Logger.Infof("Cloning repo ...")
		err = pl.GitManager.Clone(ctx, pl.Payload, oauth)
		if err != nil {
			pl.Logger.Errorf("Unable to clone repo '%s': %s", payload.RepoLink, err)
			err = &errs.StatusFailed{Remark: fmt.Sprintf("Unable to clone repo: %s", payload.RepoLink)}
			return err
		}
	} else {
		pl.Logger.Debugf("Extracting workspace")
		// Replicate workspace
		// TODO this will be changed after parallel discovery support
		if err = pl.CacheStore.ExtractWorkspace(ctx, ""); err != nil {
			pl.Logger.Errorf("Error replicating workspace: %+v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}
	coverageDir := filepath.Join(global.CodeCoverageDir, payload.OrgID, payload.RepoID, payload.BuildTargetCommit)
	if payload.CollectCoverage {
		if err = fileutils.CreateIfNotExists(coverageDir, true); err != nil {
			pl.Logger.Errorf("failed to create coverage directory %v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}
	filePath, err := pl.TASConfigManager.GetTasConfigFilePath(pl.Payload)
	if err != nil {
		return err
	}
	version, err := pl.TASConfigManager.GetVersion(filePath)
	if err != nil {
		pl.Logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	pl.Logger.Infof("TAS Version %f", version)
	pl.setEnv(payload, coverageDir)
	newDriver, err := pl.Builder.GetDriver(version, filePath)
	if err != nil {
		pl.Logger.Errorf("error crearing driver, error %v", err)
		return err
	}
	if pl.Cfg.DiscoverMode {
		err = newDriver.RunDiscovery(ctx, payload, taskPayload, oauth, coverageDir, secretMap)
	} else {
		err = newDriver.RunExecution(ctx, payload, taskPayload, oauth, coverageDir, secretMap)
	}

	return err
}

func (pl *Pipeline) getTaskPayload(payload *Payload, startTime time.Time) *TaskPayload {
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
	} else if pl.Cfg.FlakyMode {
		taskPayload.Type = FlakyTask
	} else {
		taskPayload.Type = ExecutionTask
	}
	return taskPayload
}

func (pl *Pipeline) setEnv(payload *Payload, coverageDir string) {
	// set testing taskID, orgID and buildID as environment variable
	os.Setenv("TASK_ID", payload.TaskID)
	os.Setenv("ORG_ID", payload.OrgID)
	os.Setenv("BUILD_ID", payload.BuildID)
	// set target commit_id as environment variable
	os.Setenv("COMMIT_ID", payload.BuildTargetCommit)
	// set repo_id as environment variable
	os.Setenv("REPO_ID", payload.RepoID)
	// set coverage_dir as environment variable
	os.Setenv("CODE_COVERAGE_DIR", coverageDir)
	os.Setenv("BRANCH_NAME", payload.BranchName)
	os.Setenv("ENV", pl.Cfg.Env)
	os.Setenv("ENDPOINT_POST_TEST_LIST", endpointPostTestList)
	os.Setenv("ENDPOINT_POST_TEST_RESULTS", endpointPostTestResults)
	os.Setenv("REPO_ROOT", global.RepoDir)
	os.Setenv("BLOCK_TESTS_FILE", global.BlockTestFileLocation)
	os.Setenv(global.SubModuleName, pl.Cfg.SubModule)
	// set MODULE_PATH to empty as env variable
	os.Setenv(global.ModulePath, "")
}
