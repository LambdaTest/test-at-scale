package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/fileutils"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"golang.org/x/sync/errgroup"
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
	// load tas yaml file
	version, err := pl.TASConfigManager.GetVersion(payload.TasFileName)
	if err != nil {
		pl.Logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	pl.Logger.Infof("TAS Version %f", version)

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
	if version >= global.NewTASVersion {
		// run new version
		err = pl.runNewVersion(ctx, payload, taskPayload, oauth, coverageDir, secretMap)
		return err
	}
	// set MODULE_PATH to empty as env variable
	os.Setenv(global.ModulePath, "")

	err = pl.runOldVersion(ctx, payload, taskPayload, oauth, coverageDir, secretMap)
	return err
}

func (pl *Pipeline) runOldVersion(ctx context.Context,
	payload *Payload,
	taskPayload *TaskPayload,
	oauth *Oauth,
	coverageDir string,
	secretMap map[string]string) error {
	tasConfig, err := pl.TASConfigManager.LoadAndValidateV1(ctx, payload.TasFileName, payload.EventType, payload.LicenseTier)
	if err != nil {
		pl.Logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}

	cacheKey := ""

	language := global.FrameworkLanguageMap[tasConfig.Framework]

	if language == languageJs {
		cacheKey = fmt.Sprintf("%s/%s/%s/%s", tasConfig.Cache.Version, payload.OrgID, payload.RepoID, tasConfig.Cache.Key)
	}

	pl.Logger.Infof("Tas yaml: %+v", tasConfig)

	os.Setenv("REPO_CACHE_DIR", global.RepoCacheDir)

	if tasConfig.NodeVersion != "" && language == languageJs {
		nodeVersion := tasConfig.NodeVersion
		if nodeErr := pl.installNodeVersion(ctx, nodeVersion); nodeErr != nil {
			return nodeErr
		}
	}

	if pl.Cfg.DiscoverMode {
		if err := pl.runDiscoveryV1(ctx, payload, tasConfig, cacheKey, secretMap, oauth, taskPayload); err != nil {
			return err
		}
	}

	if pl.Cfg.ExecuteMode || pl.Cfg.FlakyMode {
		// execute test cases
		if err := pl.runTestExecutionV1(ctx, tasConfig, coverageDir, secretMap, taskPayload, payload); err != nil {
			return err
		}
	}
	pl.Logger.Debugf("Completed pipeline")

	return nil
}

func (pl *Pipeline) runTestExecutionV1(ctx context.Context,
	tasConfig *TASConfig,
	coverageDir string,
	secretMap map[string]string,
	taskPayload *TaskPayload, payload *Payload) error {
	blYml := pl.BlockTestService.GetBlocklistYMLV1(tasConfig)
	if errG := pl.BlockTestService.GetBlockTests(ctx, blYml, payload.BranchName); errG != nil {
		pl.Logger.Errorf("Unable to fetch blocklisted tests: %v", errG)
		errG = errs.New(errs.GenericErrRemark.Error())
		return errG
	}

	executionResults, err := pl.TestExecutionService.RunV1(ctx, tasConfig, pl.Payload, coverageDir, secretMap)
	if err != nil {
		pl.Logger.Infof("Unable to perform test execution: %v", err)
		err = &errs.StatusFailed{Remark: "Failed in executing tests."}
		if executionResults == nil {
			return err
		}
	}

	resp, err := pl.TestExecutionService.SendResults(ctx, executionResults)
	if err != nil {
		pl.Logger.Errorf("error while sending test reports %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	taskPayload.Status = resp.TaskStatus

	if tasConfig.Postrun != nil {
		pl.Logger.Infof("Running post-run steps")
		err = pl.ExecutionManager.ExecuteUserCommands(ctx, PostRun, payload, tasConfig.Postrun, secretMap, global.RepoDir)
		if err != nil {
			pl.Logger.Errorf("Unable to run post-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running post-run steps."}
			return err
		}
	}
	return nil
}

func (pl *Pipeline) runDiscoveryV1(ctx context.Context,
	payload *Payload,
	tasConfig *TASConfig,
	cacheKey string,
	secretMap map[string]string,
	oauth *Oauth,
	taskPayload *TaskPayload) error {
	// Persist workspace
	// discover test cases
	// mark status as passed
	// Upload cache once for other builds
	language := global.FrameworkLanguageMap[tasConfig.Framework]

	if postErr := pl.TestDiscoveryService.UpdateSubmoduleList(ctx, payload.BuildID, 1); postErr != nil {
		return postErr
	}
	blYml := pl.BlockTestService.GetBlocklistYMLV1(tasConfig)
	if errG := pl.BlockTestService.GetBlockTests(ctx, blYml, payload.BranchName); errG != nil {
		pl.Logger.Errorf("Unable to fetch blocklisted tests: %v", errG)
		errG = errs.New(errs.GenericErrRemark.Error())
		return errG
	}
	g, errCtx := errgroup.WithContext(ctx)

	if language == languageJs {
		g.Go(func() error {
			if errG := pl.CacheStore.Download(errCtx, cacheKey); errG != nil {
				pl.Logger.Errorf("Unable to download cache: %v", errG)
				errG = errs.New(errs.GenericErrRemark.Error())
				return errG
			}
			return nil
		})

		err := pl.ExecutionManager.ExecuteInternalCommands(ctx, InstallRunners, global.InstallRunnerCmds, global.RepoDir, nil, nil)
		if err != nil {
			pl.Logger.Errorf("Unable to install custom runners %v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}

	pl.Logger.Infof("Identifying changed files ...")
	diffExists := true
	diff := map[string]int{}
	g.Go(func() error {
		diffC, errG := pl.DiffManager.GetChangedFiles(errCtx, payload, oauth)
		if errG != nil {
			if errors.Is(errG, errs.ErrGitDiffNotFound) {
				diffExists = false
			} else {
				pl.Logger.Errorf("Unable to identify changed files %s", errG)
				errG = errs.New("Error occurred in fetching diff from GitHub")
				return errG
			}
		}
		diff = diffC
		return nil
	})

	err := g.Wait()
	if err != nil {
		return err
	}

	if tasConfig.Prerun != nil {
		pl.Logger.Infof("Running pre-run steps for top module")
		err = pl.ExecutionManager.ExecuteUserCommands(ctx, PreRun, payload, tasConfig.Prerun, secretMap, global.RepoDir)
		if err != nil {
			pl.Logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
	}

	pl.Logger.Debugf("Caching workspace")

	if err = pl.CacheStore.CacheWorkspace(ctx, ""); err != nil {
		pl.Logger.Errorf("Error caching workspace: %+v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	err = pl.TestDiscoveryService.Discover(ctx, tasConfig, pl.Payload, secretMap, diff, diffExists)
	if err != nil {
		pl.Logger.Errorf("Unable to perform test discovery: %+v", err)
		err = &errs.StatusFailed{Remark: "Failed in discovering tests"}
		return err
	}
	if language == languageJs {
		if err = pl.CacheStore.Upload(ctx, cacheKey, tasConfig.Cache.Paths...); err != nil {
			pl.Logger.Errorf("Unable to upload cache: %v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}
	taskPayload.Status = Passed
	pl.Logger.Debugf("Cache uploaded successfully")
	return nil
}

func (pl *Pipeline) installNodeVersion(ctx context.Context, nodeVersion string) error {
	// Running the `source` commands in a directory where .nvmrc is present, exits with exitCode 3
	// https://github.com/nvm-sh/nvm/issues/1985
	// TODO [good-to-have]: Auto-read and install from .nvmrc file, if present
	commands := []string{
		"source /home/nucleus/.nvm/nvm.sh",
		fmt.Sprintf("nvm install %s", nodeVersion),
	}
	pl.Logger.Infof("Using user-defined node version: %v", nodeVersion)
	err := pl.ExecutionManager.ExecuteInternalCommands(ctx, InstallNodeVer, commands, "", nil, nil)
	if err != nil {
		pl.Logger.Errorf("Unable to install user-defined nodeversion %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("/home/nucleus/.nvm/versions/node/v%s/bin:%s", nodeVersion, origPath))
	return nil
}

func (pl *Pipeline) runNewVersion(ctx context.Context,
	payload *Payload,
	taskPayload *TaskPayload,
	oauth *Oauth,
	coverageDir string,
	secretMap map[string]string) error {
	tasConfig, err := pl.TASConfigManager.LoadAndValidateV2(ctx, payload.TasFileName, payload.EventType, payload.LicenseTier)

	if err != nil {
		pl.Logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	if pl.Cfg.DiscoverMode {
		if err := pl.runDiscoveryV2(payload, tasConfig, taskPayload, ctx, oauth, secretMap); err != nil {
			taskPayload.Status = Error
			return err
		}
	} else if pl.Cfg.ExecuteMode || pl.Cfg.FlakyMode {
		if err := pl.runTestExecutionV2(ctx, payload, tasConfig, taskPayload, coverageDir, secretMap); err != nil {
			return err
		}
	}

	return nil
}

func (pl *Pipeline) runDiscoveryV2(payload *Payload,
	tasConfig *TASConfigV2,
	taskPayload *TaskPayload,
	ctx context.Context,
	oauth *Oauth,
	secretMap map[string]string) error {
	/*
		Discovery steps
	*/
	// iterate through all sub modules
	// Persist workspace
	// Upload cache once for other builds
	cacheKey := fmt.Sprintf("%s/%s/%s/%s", tasConfig.Cache.Version, payload.OrgID, payload.RepoID, tasConfig.Cache.Key)

	taskPayload.Status = Passed
	g, errCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if errG := pl.CacheStore.Download(errCtx, cacheKey); errG != nil {
			pl.Logger.Errorf("Unable to download cache: %v", errG)
			errG = errs.New(errs.GenericErrRemark.Error())
			return errG
		}
		return nil
	})
	diffExists := true
	diff := map[string]int{}
	g.Go(func() error {
		diffC, errG := pl.DiffManager.GetChangedFiles(errCtx, payload, oauth)
		if errG != nil {
			if errors.Is(errG, errs.ErrGitDiffNotFound) {
				diffExists = false
			} else {
				pl.Logger.Errorf("Unable to identify changed files %s", errG)
				errG = errs.New("Error occurred in fetching diff from GitHub")
				return errG
			}
		}
		diff = diffC
		return nil
	})
	err := g.Wait()
	if err != nil {
		return err
	}

	readerBuffer := new(bytes.Buffer)
	defer func() {
		blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, os.Getenv("TASK_ID"), PreRun)
		pl.Logger.Debugf("Writing the preRUN logs to path %s", blobPath)
		if writeErr := <-pl.ExecutionManager.StoreCommandLogs(ctx, blobPath, readerBuffer); writeErr != nil {
			// error in writing log should not fail the build
			pl.Logger.Errorf("error in writing pre run log, error %v", writeErr)
		}
	}()

	if payload.EventType == EventPush {
		if discoveryErr := pl.runDiscoveryV2Helper(ctx, tasConfig.PostMerge.PreRun,
			tasConfig.PostMerge.SubModules, payload, tasConfig,
			taskPayload, diff, diffExists, readerBuffer, secretMap); discoveryErr != nil {
			return discoveryErr
		}
	} else {
		if discoveryErr := pl.runDiscoveryV2Helper(ctx, tasConfig.PreMerge.PreRun, tasConfig.PreMerge.SubModules,
			payload, tasConfig, taskPayload, diff, diffExists, readerBuffer, secretMap); discoveryErr != nil {
			return discoveryErr
		}
	}
	if err = pl.CacheStore.Upload(ctx, cacheKey, tasConfig.Cache.Paths...); err != nil {
		pl.Logger.Errorf("Unable to upload cache: %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	pl.Logger.Debugf("Cache uploaded successfully")
	return nil
}

func (pl *Pipeline) runPreRunForEachSubModule(ctx context.Context,
	payload *Payload,
	subModule *SubModule,
	secretMap map[string]string,
	readerBuffer *bytes.Buffer) error {
	pl.Logger.Debugf("Running discovery for sub module %s", subModule.Name)
	blYML := pl.BlockTestService.GetBlocklistYMLV2(subModule)
	if err := pl.BlockTestService.GetBlockTests(ctx, blYML, payload.BranchName); err != nil {
		pl.Logger.Errorf("Unable to fetch blocklisted tests: %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	modulePath := path.Join(global.RepoDir, subModule.Path)
	// PRE RUN steps
	if subModule.Prerun != nil {
		pl.Logger.Infof("Running pre-run steps for submodule %s", subModule.Name)
		err := pl.ExecutionManager.ExecuteUserCommandsV2(ctx, PreRun, payload, subModule.Prerun,
			secretMap, modulePath, subModule.Name, readerBuffer)
		if err != nil {
			pl.Logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
		pl.Logger.Debugf("error checks end")
	}
	err := pl.ExecutionManager.ExecuteInternalCommands(ctx, InstallRunners, global.InstallRunnerCmds, modulePath, nil, nil)
	if err != nil {
		pl.Logger.Errorf("Unable to install custom runners %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	return nil
}

func (pl *Pipeline) runTestExecutionV2(ctx context.Context,
	payload *Payload,
	tasConfig *TASConfigV2,
	taskPayload *TaskPayload,
	coverageDir string,
	secretMap map[string]string) error {
	subModule, err := pl.findSubmodule(tasConfig, payload)
	if err != nil {
		pl.Logger.Errorf("Error finding sub module %s in tas config file", pl.Cfg.SubModule)
		return err
	}
	var envMap map[string]string
	var target []string

	if payload.EventType == EventPullRequest {
		target = subModule.Patterns
		envMap = tasConfig.PreMerge.EnvMap
	} else {
		target = subModule.Patterns
		envMap = tasConfig.PostMerge.EnvMap
	}
	if envMap == nil {
		envMap = map[string]string{}
	}

	// overwrite the existing env with more specific one

	if subModule.Prerun != nil && subModule.Prerun.EnvMap != nil {
		for k, v := range subModule.Prerun.EnvMap {
			envMap[k] = v
		}
	}

	/*
		1. run PRE run steps
		2. run Test execution
		3. run POST run steps
	*/

	modulePath := path.Join(global.RepoDir, subModule.Path)
	// PRE RUN steps should be run only if RunPrerunEveryTime is set to true
	if subModule.Prerun != nil && subModule.RunPrerunEveryTime {
		pl.Logger.Infof("Running pre-run steps for submodule %s", subModule.Name)
		err = pl.ExecutionManager.ExecuteUserCommands(ctx, PreRun, payload, subModule.Prerun, secretMap, modulePath)
		if err != nil {
			pl.Logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
		err = pl.ExecutionManager.ExecuteInternalCommands(ctx, InstallRunners, global.InstallRunnerCmds, global.RepoDir, nil, nil)
		if err != nil {
			pl.Logger.Errorf("Unable to install custom runners %v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}
	// TO BE removed
	// pl.Logger.Debugf("Sleep time for debug")
	// time.Sleep(time.Minute * 10)
	testResult, err := pl.TestExecutionService.RunV2(ctx, tasConfig, subModule, payload, coverageDir, envMap, target, secretMap)
	if err != nil {
		return err
	}
	resp, err := pl.TestExecutionService.SendResults(ctx, testResult)
	if err != nil {
		pl.Logger.Errorf("error while sending test reports %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	taskPayload.Status = resp.TaskStatus

	if subModule.Postrun != nil {
		pl.Logger.Infof("Running post-run steps")
		err = pl.ExecutionManager.ExecuteUserCommands(ctx, PostRun, payload, subModule.Postrun, secretMap, modulePath)
		if err != nil {
			pl.Logger.Errorf("Unable to run post-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running post-run steps."}
			return err
		}
	}
	return nil
}

func (pl *Pipeline) findSubmodule(tasConfig *TASConfigV2, payload *Payload) (*SubModule, error) {
	if payload.EventType == EventPullRequest {
		for i := 0; i < len(tasConfig.PreMerge.SubModules); i++ {
			if tasConfig.PreMerge.SubModules[i].Name == pl.Cfg.SubModule {
				return &tasConfig.PreMerge.SubModules[i], nil
			}
		}
	} else {
		for i := 0; i < len(tasConfig.PostMerge.SubModules); i++ {
			if tasConfig.PostMerge.SubModules[i].Name == pl.Cfg.SubModule {
				return &tasConfig.PostMerge.SubModules[i], nil
			}
		}
	}
	return nil, errs.ErrSubModuleNotFound
}

func (pl *Pipeline) runDiscoveryV2Helper(ctx context.Context,
	topPreRun *Run,
	subModuleList []SubModule,
	payload *Payload,
	tasConfig *TASConfigV2,
	taskPayload *TaskPayload,
	diff map[string]int,
	diffExists bool,
	readerBuffer *bytes.Buffer,
	secretMap map[string]string) error {
	totalSubmoduleCount := len(subModuleList)
	if apiErr := pl.TestDiscoveryService.UpdateSubmoduleList(ctx, payload.BuildID, totalSubmoduleCount); apiErr != nil {
		return apiErr
	}
	errChannelPreRun := make(chan error, totalSubmoduleCount)
	preRunWaitGroup := sync.WaitGroup{}
	if tasConfig.NodeVersion != "" {
		if err := pl.installNodeVersion(ctx, tasConfig.NodeVersion); err != nil {
			return err
		}
	}
	if topPreRun != nil {
		pl.Logger.Debugf("Running Pre Run on top level")
		if err := pl.ExecutionManager.ExecuteUserCommandsV2(ctx, PreRun, payload,
			topPreRun, secretMap, global.RepoDir, "TOP-LEVEL", readerBuffer); err != nil {
			pl.Logger.Errorf("Error occurred running top level PreRun , err %v", err)
			return err
		}
	}
	pl.Logger.Debugf("pre run on top level ended")
	for i := 0; i < totalSubmoduleCount; i++ {
		preRunWaitGroup.Add(1)
		go func(subModule *SubModule) {
			defer preRunWaitGroup.Done()

			dicoveryErr := pl.runPreRunForEachSubModule(ctx, payload, subModule, secretMap, readerBuffer)
			if dicoveryErr != nil {
				taskPayload.Status = Error
				pl.Logger.Errorf("error while running discovery for sub module %s, error %v", subModule.Name, dicoveryErr)
			}
			errChannelPreRun <- dicoveryErr
		}(&subModuleList[i])
	}
	preRunWaitGroup.Wait()
	pl.Logger.Debugf("checking the pre runs errors ")
	for i := 0; i < totalSubmoduleCount; i++ {
		e := <-errChannelPreRun
		if e != nil {
			return e
		}
	}
	pl.Logger.Debugf("checked the pre run errors")
	pl.Logger.Debugf("Caching workspace")
	// TODO: this will be change after we move to parallel pod executuon
	if err := pl.CacheStore.CacheWorkspace(ctx, ""); err != nil {
		pl.Logger.Errorf("Error caching workspace: %+v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	errChannelDiscovery := make(chan error, totalSubmoduleCount)
	discoveryWaitGroup := sync.WaitGroup{}
	for i := 0; i < totalSubmoduleCount; i++ {
		discoveryWaitGroup.Add(1)
		go func(subModule *SubModule) {
			defer discoveryWaitGroup.Done()
			err := pl.runDiscoveryForEachSubModule(ctx, subModule, tasConfig, diff, diffExists, secretMap)
			errChannelDiscovery <- err
		}(&subModuleList[i])
	}
	discoveryWaitGroup.Wait()
	for i := 0; i < totalSubmoduleCount; i++ {
		e := <-errChannelDiscovery
		if e != nil {
			return e
		}
	}
	return nil
}

func (pl *Pipeline) runDiscoveryForEachSubModule(ctx context.Context, subModule *SubModule,
	tasConfig *TASConfigV2,
	diff map[string]int,
	diffExists bool,
	secretMap map[string]string) error {
	if err := pl.TestDiscoveryService.DiscoverV2(ctx, subModule, pl.Payload, secretMap,
		tasConfig, diff, diffExists); err != nil {
		pl.Logger.Errorf("Unable to perform test discovery: %+v", err)
		err = &errs.StatusFailed{Remark: "Failed in discovering tests"}
		return err
	}
	return nil
}
