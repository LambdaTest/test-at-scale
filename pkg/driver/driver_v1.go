/*
This file implements core.Driver  with operation over TAS config (YAML) version 1
*/
package driver

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logwriter"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
	"golang.org/x/sync/errgroup"
)

const languageJs = "javascript"

type (
	driverV1 struct {
		logger               lumber.Logger
		nodeInstaller        NodeInstaller
		TestExecutionService core.TestExecutionService
		TestDiscoveryService core.TestDiscoveryService
		AzureClient          core.AzureClient
		BlockTestService     core.BlockTestService
		ExecutionManager     core.ExecutionManager
		TASConfigManager     core.TASConfigManager
		CacheStore           core.CacheStore
		DiffManager          core.DiffManager
		ListSubModuleService core.ListSubModuleService
		TASVersion           int
		TASFilePath          string
	}

	setUpResultV1 struct {
		diffExists bool
		diff       map[string]int
		cacheKey   string
	}
)

func (d *driverV1) RunDiscovery(ctx context.Context, payload *core.Payload,
	taskPayload *core.TaskPayload, oauth *core.Oauth, coverageDir string, secretMap map[string]string) error {
	tas, err := d.TASConfigManager.LoadAndValidate(ctx, d.TASVersion, d.TASFilePath, payload.EventType, payload.LicenseTier, d.TASFilePath)
	if err != nil {
		d.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	tasConfig := tas.(*core.TASConfig)
	language := global.FrameworkLanguageMap[tasConfig.Framework]
	setupResults, err := d.setUp(ctx, payload, tasConfig, oauth, language)
	if err != nil {
		d.logger.Errorf("Error while doing common opertations error %v", err)
		return err
	}

	if postErr := d.ListSubModuleService.Send(ctx, payload.BuildID, 1); postErr != nil {
		return postErr
	}

	if tasConfig.Prerun != nil {
		d.logger.Infof("Running pre-run steps for top module")
		azureLogWriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposePreRunLogs, d.logger)
		err = d.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload, tasConfig.Prerun, secretMap, azureLogWriter, global.RepoDir)
		if err != nil {
			d.logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
	}

	err = d.ExecutionManager.ExecuteInternalCommands(ctx, core.InstallRunners, global.InstallRunnerCmds, global.RepoDir, nil, nil)
	if err != nil {
		d.logger.Errorf("Unable to install custom runners %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	d.logger.Debugf("Caching workspace")

	if err = d.CacheStore.CacheWorkspace(ctx, ""); err != nil {
		d.logger.Errorf("Error caching workspace: %+v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	args := d.buildDiscoveryArgs(payload, tasConfig, secretMap, setupResults.diffExists, setupResults.diff)

	discoveryResult, err := d.TestDiscoveryService.Discover(ctx, &args)
	if err != nil {
		d.logger.Errorf("Unable to perform test discovery: %+v", err)
		err = &errs.StatusFailed{Remark: "Failed in discovering tests"}
		return err
	}

	populateDiscovery(discoveryResult, tasConfig)
	if err = d.TestDiscoveryService.SendResult(ctx, discoveryResult); err != nil {
		d.logger.Errorf("error while sending discovery API call , error %v", err)
		return err
	}
	if language == languageJs {
		if err = d.CacheStore.Upload(ctx, setupResults.cacheKey, tasConfig.Cache.Paths...); err != nil {
			d.logger.Errorf("Unable to upload cache: %v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}

	taskPayload.Status = core.Passed
	d.logger.Debugf("Cache uploaded successfully")
	return nil
	// return nil
}

func (d *driverV1) RunExecution(ctx context.Context, payload *core.Payload,
	taskPayload *core.TaskPayload, oauth *core.Oauth, coverageDir string, secretMap map[string]string) error {
	tas, err := d.TASConfigManager.LoadAndValidate(ctx, 1, d.TASFilePath, payload.EventType, payload.LicenseTier, d.TASFilePath)
	if err != nil {
		d.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	tasConfig := tas.(*core.TASConfig)
	if cachErr := d.setCache(tasConfig); cachErr != nil {
		return cachErr
	}
	if errG := d.BlockTestService.GetBlockTests(ctx, tasConfig.Blocklist, payload.BranchName); errG != nil {
		d.logger.Errorf("Unable to fetch blocklisted tests: %v", errG)
		errG = errs.New(errs.GenericErrRemark.Error())
		return errG
	}
	buildArgs := d.buildTestExecutionArgs(payload, tasConfig, secretMap, coverageDir)
	executionResults, err := d.TestExecutionService.Run(ctx, &buildArgs)
	if err != nil {
		d.logger.Infof("Unable to perform test execution: %v", err)
		err = &errs.StatusFailed{Remark: "Failed in executing tests."}
		if executionResults == nil {
			return err
		}
	}

	resp, err := d.TestExecutionService.SendResults(ctx, executionResults)
	if err != nil {
		d.logger.Errorf("error while sending test reports %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	taskPayload.Status = resp.TaskStatus
	logWriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposePostRunLogs, d.logger)

	if tasConfig.Postrun != nil {
		d.logger.Infof("Running post-run steps")
		err = d.ExecutionManager.ExecuteUserCommands(ctx, core.PostRun, payload, tasConfig.Postrun, secretMap, logWriter, global.RepoDir)
		if err != nil {
			d.logger.Errorf("Unable to run post-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running post-run steps."}
			return err
		}
	}
	return nil
}

func (d *driverV1) setUp(ctx context.Context, payload *core.Payload,
	tasConfig *core.TASConfig, oauth *core.Oauth, language string) (*setUpResultV1, error) {
	d.logger.Infof("Tas yaml: %+v", tasConfig)
	if err := d.setCache(tasConfig); err != nil {
		return nil, err
	}
	cacheKey := ""
	if language == languageJs {
		cacheKey = tasConfig.Cache.Key
	}

	os.Setenv("REPO_CACHE_DIR", global.RepoCacheDir)
	if tasConfig.NodeVersion != "" && language == languageJs {
		nodeVersion := tasConfig.NodeVersion
		if nodeErr := d.nodeInstaller.InstallNodeVersion(ctx, nodeVersion); nodeErr != nil {
			return nil, nodeErr
		}
	}
	blYml := tasConfig.Blocklist
	if errG := d.BlockTestService.GetBlockTests(ctx, blYml, payload.BranchName); errG != nil {
		d.logger.Errorf("Unable to fetch blocklisted tests: %v", errG)
		errG = errs.New(errs.GenericErrRemark.Error())
		return nil, errG
	}

	g, errCtx := errgroup.WithContext(ctx)
	if language == languageJs {
		g.Go(func() error {
			if errG := d.CacheStore.Download(errCtx, cacheKey); errG != nil {
				d.logger.Errorf("Unable to download cache: %v", errG)
				errG = errs.New(errs.GenericErrRemark.Error())
				return errG
			}
			return nil
		})
	}

	d.logger.Infof("Identifying changed files ...")
	diffExists := true
	diff := map[string]int{}
	g.Go(func() error {
		diffC, errG := d.DiffManager.GetChangedFiles(errCtx, payload, oauth)
		if errG != nil {
			if errors.Is(errG, errs.ErrGitDiffNotFound) {
				diffExists = false
			} else {
				d.logger.Errorf("Unable to identify changed files %s", errG)
				errG = errs.New("Error occurred in fetching diff from GitHub")
				return errG
			}
		}
		diff = diffC
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &setUpResultV1{
		diffExists: diffExists,
		diff:       diff,
		cacheKey:   cacheKey,
	}, nil
}

func (d *driverV1) buildDiscoveryArgs(payload *core.Payload, tasConfig *core.TASConfig,
	secretMap map[string]string,
	diffExists bool,
	diff map[string]int) core.DiscoveyArgs {
	testPattern, envMap := d.getEnvAndPattern(payload, tasConfig)
	return core.DiscoveyArgs{
		TestPattern:      testPattern,
		Payload:          payload,
		EnvMap:           envMap,
		SecretData:       secretMap,
		TestConfigFile:   tasConfig.ConfigFile,
		FrameWork:        tasConfig.Framework,
		SmartRun:         tasConfig.SmartRun,
		Diff:             diff,
		DiffExists:       diffExists,
		FrameWorkVersion: tasConfig.FrameworkVersion,
		CWD:              global.RepoDir,
	}
}

func (d *driverV1) buildTestExecutionArgs(payload *core.Payload, tasConfig *core.TASConfig,
	secretMap map[string]string,
	coverageDir string) core.TestExecutionArgs {
	testPattern, envMap := d.getEnvAndPattern(payload, tasConfig)
	logWriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposeExecutionLogs, d.logger)
	return core.TestExecutionArgs{
		Payload:           payload,
		CoverageDir:       coverageDir,
		LogWriterStrategy: logWriter,
		TestPattern:       testPattern,
		EnvMap:            envMap,
		TestConfigFile:    tasConfig.ConfigFile,
		FrameWork:         tasConfig.Framework,
		SecretData:        secretMap,
		FrameWorkVersion:  tasConfig.FrameworkVersion,
		CWD:               global.RepoDir,
	}
}

func (d *driverV1) getEnvAndPattern(payload *core.Payload, tasConfig *core.TASConfig) (target []string, envMap map[string]string) {
	if payload.EventType == core.EventPullRequest {
		return tasConfig.Premerge.Patterns, tasConfig.Premerge.EnvMap
	}
	return tasConfig.Postmerge.Patterns, tasConfig.Postmerge.EnvMap
}

func populateDiscovery(testDiscoveryResult *core.DiscoveryResult, tasConfig *core.TASConfig) {
	testDiscoveryResult.Parallelism = tasConfig.Parallelism
	testDiscoveryResult.SplitMode = tasConfig.SplitMode
}

func (d *driverV1) setCache(tasConfig *core.TASConfig) error {
	language := global.FrameworkLanguageMap[tasConfig.Framework]
	if tasConfig.Cache == nil && language == "javascript" {
		checksum, err := utils.ComputeChecksum(fmt.Sprintf("%s/%s", global.RepoDir, global.PackageJSON))
		if err != nil {
			d.logger.Errorf("Error while computing checksum, error %v", err)
			return err
		}
		tasConfig.Cache = &core.Cache{
			Key:   checksum,
			Paths: []string{},
		}
	}
	return nil
}
