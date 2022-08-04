/*
This file implements core.Driver with operation over TAS config (YAML) version 2
*/
package driver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logwriter"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
	"golang.org/x/sync/errgroup"
)

const preRunLog = "Running Pre Run on Top level"

type (
	driverV2 struct {
		logger               lumber.Logger
		TestExecutionService core.TestExecutionService
		AzureClient          core.AzureClient
		BlockTestService     core.BlockTestService
		ExecutionManager     core.ExecutionManager
		TASConfigManager     core.TASConfigManager
		CacheStore           core.CacheStore
		DiffManager          core.DiffManager
		ListSubModuleService core.ListSubModuleService
		nodeInstaller        NodeInstaller
		TestDiscoveryService core.TestDiscoveryService
		TASVersion           int
		TASFilePath          string
	}

	setUpResultV2 struct {
		diffExists bool
		diff       map[string]int
		cacheKey   string
	}
)

func (d *driverV2) RunDiscovery(ctx context.Context, payload *core.Payload,
	taskPayload *core.TaskPayload, oauth *core.Oauth, coverageDir string, secretMap map[string]string) error {
	// do something
	d.logger.Debugf("Running in %d version", d.TASVersion)
	tas, err := d.TASConfigManager.LoadAndValidate(ctx, d.TASVersion, d.TASFilePath, payload.EventType, payload.LicenseTier, d.TASFilePath)
	if err != nil {
		d.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	tasConfig := tas.(*core.TASConfigV2)
	taskPayload.Status = core.Passed
	setUpResult, err := d.setUpDiscovery(ctx, payload, tasConfig, oauth)
	if err != nil {
		return err
	}
	mainBuffer := new(bytes.Buffer)
	azureLogWriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposePreRunLogs, d.logger)

	defer func() {
		if writeErr := <-azureLogWriter.Write(ctx, mainBuffer); writeErr != nil {
			// error in writing log should not fail the build
			d.logger.Errorf("error in writing pre run log, error %v", writeErr)
		}
	}()

	if payload.EventType == core.EventPush {
		if discoveryErr := d.runDiscoveryHelper(ctx, tasConfig.PostMerge.PreRun,
			tasConfig.PostMerge.SubModules, payload, tasConfig,
			taskPayload, setUpResult.diff, setUpResult.diffExists, mainBuffer, secretMap); discoveryErr != nil {
			return discoveryErr
		}
	} else {
		if discoveryErr := d.runDiscoveryHelper(ctx, tasConfig.PreMerge.PreRun, tasConfig.PreMerge.SubModules,
			payload, tasConfig, taskPayload, setUpResult.diff, setUpResult.diffExists, mainBuffer, secretMap); discoveryErr != nil {
			return discoveryErr
		}
	}
	if err = d.CacheStore.Upload(ctx, setUpResult.cacheKey, tasConfig.Cache.Paths...); err != nil {
		// cache upload failure should not fail the task
		d.logger.Errorf("Unable to upload cache: %v", err)
	}
	d.logger.Debugf("Cache uploaded successfully")

	return nil
}

func (d *driverV2) RunExecution(ctx context.Context, payload *core.Payload,
	taskPayload *core.TaskPayload, oauth *core.Oauth, coverageDir string, secretMap map[string]string) error {
	tas, err := d.TASConfigManager.LoadAndValidate(ctx, d.TASVersion, d.TASFilePath, payload.EventType, payload.LicenseTier, d.TASFilePath)
	if err != nil {
		d.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}

	subModuleName := os.Getenv(global.SubModuleName)
	tasConfig := tas.(*core.TASConfigV2)
	if cachErr := d.setCache(tasConfig); cachErr != nil {
		return cachErr
	}
	subModule, err := d.findSubmodule(tasConfig, payload, subModuleName)
	if err != nil {
		d.logger.Errorf("Error finding sub module %s in tas config file", subModuleName)
		return err
	}
	// Get blocklist data before execution
	blYML := subModule.Blocklist
	if err = d.BlockTestService.GetBlockTests(ctx, blYML, payload.BranchName); err != nil {
		d.logger.Errorf("Unable to fetch blocklisted tests: %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	modulePath := path.Join(global.RepoDir, subModule.Path)
	// PRE RUN steps should be run only if RunPrerunEveryTime is set to true
	if subModule.Prerun != nil && subModule.RunPrerunEveryTime {
		if preErr := d.runPreRunBeforeTestExecution(ctx, tasConfig, subModule, payload, secretMap, modulePath); preErr != nil {
			return preErr
		}
	}
	args := d.buildTestExecutionArgs(payload, tasConfig, subModule, secretMap, coverageDir)
	testResult, err := d.TestExecutionService.Run(ctx, &args)
	if err != nil {
		return err
	}
	resp, err := d.TestExecutionService.SendResults(ctx, testResult)
	if err != nil {
		d.logger.Errorf("error while sending test reports %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	taskPayload.Status = resp.TaskStatus

	if subModule.Postrun != nil {
		d.logger.Infof("Running post-run steps")
		azureLogwriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposePostRunLogs, d.logger)

		err = d.ExecutionManager.ExecuteUserCommands(ctx, core.PostRun, payload, subModule.Postrun, secretMap, azureLogwriter, modulePath)
		if err != nil {
			d.logger.Errorf("Unable to run post-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running post-run steps."}
			return err
		}
	}
	return nil
}

func (d *driverV2) runPreRunBeforeTestExecution(ctx context.Context,
	tasConfig *core.TASConfigV2,
	subModule *core.SubModule,
	payload *core.Payload,
	secretMap map[string]string,
	modulePath string) error {
	if tasConfig.NodeVersion != "" {
		// install node version before preRuns
		if err := d.nodeInstaller.InstallNodeVersion(ctx, tasConfig.NodeVersion); err != nil {
			d.logger.Debugf("error while installing node of version %s, error %v ", tasConfig.NodeVersion, err)
			return err
		}
	}

	d.logger.Infof("Running pre-run steps for submodule %s", subModule.Name)
	azureLogwriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposePreRunLogs, d.logger)
	err := d.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload, subModule.Prerun, secretMap, azureLogwriter, modulePath)
	if err != nil {
		d.logger.Errorf("Unable to run pre-run steps %v", err)
		err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
		return err
	}
	d.logger.Debugf("installing runners at path %s", modulePath)
	if err = d.ExecutionManager.ExecuteInternalCommands(ctx, core.InstallRunners, global.InstallRunnerCmds,
		modulePath, nil, nil); err != nil {
		d.logger.Errorf("Unable to install custom runners %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	return nil
}

func (d *driverV2) runDiscoveryHelper(ctx context.Context,
	topPreRun *core.Run,
	subModuleList []core.SubModule,
	payload *core.Payload,
	tasConfig *core.TASConfigV2,
	taskPayload *core.TaskPayload,
	diff map[string]int,
	diffExists bool,
	mainBuffer *bytes.Buffer,
	secretMap map[string]string) error {
	totalSubmoduleCount := len(subModuleList)
	if apiErr := d.ListSubModuleService.Send(ctx, payload.BuildID, totalSubmoduleCount); apiErr != nil {
		return apiErr
	}

	if tasConfig.NodeVersion != "" {
		if err := d.nodeInstaller.InstallNodeVersion(ctx, tasConfig.NodeVersion); err != nil {
			return err
		}
	}

	if err := d.runPreRunCommand(ctx, topPreRun, mainBuffer, payload, secretMap, taskPayload, subModuleList); err != nil {
		return err
	}
	d.logger.Debugf("Caching workspace")
	// TODO: this will be change after we move to parallel pod executuon
	if err := d.CacheStore.CacheWorkspace(ctx, ""); err != nil {
		d.logger.Errorf("Error caching workspace: %+v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	errChannelDiscovery := make(chan error, totalSubmoduleCount)
	discoveryWaitGroup := sync.WaitGroup{}
	for i := 0; i < totalSubmoduleCount; i++ {
		discoveryWaitGroup.Add(1)
		go func(subModule *core.SubModule) {
			defer discoveryWaitGroup.Done()
			err := d.runDiscoveryForEachSubModule(ctx, payload, subModule, tasConfig, diff, diffExists, secretMap)
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

func (d *driverV2) runPreRunCommand(ctx context.Context,
	topPreRun *core.Run,
	mainBuffer *bytes.Buffer, payload *core.Payload,
	secretMap map[string]string, taskPayload *core.TaskPayload,
	subModuleList []core.SubModule) error {
	totalSubmoduleCount := len(subModuleList)

	errChannelPreRun := make(chan error, totalSubmoduleCount)

	preRunWaitGroup := sync.WaitGroup{}

	if topPreRun != nil {
		d.logger.Debugf("Running Pre Run on top level")
		if _, err := mainBuffer.WriteString(preRunLog); err != nil {
			return err
		}
		bufferWirter := logwriter.NewBufferLogWriter("TOP-LEVEL", mainBuffer, d.logger)
		if err := d.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload,
			topPreRun, secretMap, bufferWirter, global.RepoDir); err != nil {
			d.logger.Errorf("Error occurred running top level PreRun , err %v", err)
			return err
		}
	}

	bufferList := []*bytes.Buffer{}

	d.logger.Debugf("pre run on top level ended")
	for i := 0; i < totalSubmoduleCount; i++ {
		preRunWaitGroup.Add(1)

		newBuffer := new(bytes.Buffer)

		bufferList = append(bufferList, newBuffer)

		go func(subModule *core.SubModule) {
			defer preRunWaitGroup.Done()
			bufferWirterSubmodule := logwriter.NewBufferLogWriter(subModule.Name, newBuffer, d.logger)
			dicoveryErr := d.runPreRunForEachSubModule(ctx, payload, subModule, secretMap, bufferWirterSubmodule)
			if dicoveryErr != nil {
				taskPayload.Status = core.Error
				d.logger.Errorf("error while running discovery for sub module %s, error %v", subModule.Name, dicoveryErr)
			}
			errChannelPreRun <- dicoveryErr
		}(&subModuleList[i])
	}

	preRunWaitGroup.Wait()

	for i := 0; i < totalSubmoduleCount; i++ {
		mainBuffer.WriteString(bufferList[i].String())
	}
	for i := 0; i < totalSubmoduleCount; i++ {
		e := <-errChannelPreRun
		if e != nil {
			d.logger.Debugf("pre run failed with error %v", e)
			return e
		}
	}
	return nil
}

func (d *driverV2) runDiscoveryForEachSubModule(ctx context.Context,
	payload *core.Payload,
	subModule *core.SubModule,
	tasConfig *core.TASConfigV2,
	diff map[string]int,
	diffExists bool,
	secretMap map[string]string) error {
	args := d.buildDiscoveryArgs(payload, tasConfig, subModule, secretMap, diffExists, diff)

	discoveryResult, err := d.TestDiscoveryService.Discover(ctx, &args)
	if err != nil {
		d.logger.Errorf("Unable to perform test discovery: %+v", err)
		err = &errs.StatusFailed{Remark: "Failed in discovering tests"}
		return err
	}
	populateTestDiscoveryV2(discoveryResult, subModule, tasConfig)
	if err := d.TestDiscoveryService.SendResult(ctx, discoveryResult); err != nil {
		return err
	}
	return nil
}

func (d *driverV2) runPreRunForEachSubModule(ctx context.Context,
	payload *core.Payload,
	subModule *core.SubModule,
	secretMap map[string]string,
	bufferWirterSubmodule core.LogWriterStrategy) error {
	d.logger.Debugf("Running discovery for sub module %s", subModule.Name)
	blYML := subModule.Blocklist
	if err := d.BlockTestService.GetBlockTests(ctx, blYML, payload.BranchName); err != nil {
		d.logger.Errorf("Unable to fetch blocklisted tests: %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	modulePath := path.Join(global.RepoDir, subModule.Path)
	// PRE RUN steps
	if subModule.Prerun != nil {
		d.logger.Infof("Running pre-run steps for submodule %s", subModule.Name)
		err := d.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload, subModule.Prerun,
			secretMap, bufferWirterSubmodule, modulePath)
		if err != nil {
			d.logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
		d.logger.Debugf("error checks end")
	}
	err := d.ExecutionManager.ExecuteInternalCommands(ctx, core.InstallRunners, global.InstallRunnerCmds, modulePath, nil, nil)
	if err != nil {
		d.logger.Errorf("Unable to install custom runners %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	return nil
}

func (d *driverV2) setUpDiscovery(ctx context.Context,
	payload *core.Payload,
	tasConfig *core.TASConfigV2,
	oauth *core.Oauth) (*setUpResultV2, error) {
	if err := d.setCache(tasConfig); err != nil {
		return nil, err
	}
	cacheKey := tasConfig.Cache.Key

	g, errCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if errG := d.CacheStore.Download(errCtx, cacheKey); errG != nil {
			d.logger.Errorf("Unable to download cache: %v", errG)
			errG = errs.New(errs.GenericErrRemark.Error())
			return errG
		}
		return nil
	})
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
	err := g.Wait()
	if err != nil {
		return nil, err
	}
	return &setUpResultV2{
		cacheKey:   cacheKey,
		diffExists: diffExists,
		diff:       diff,
	}, nil
}

func (d *driverV2) buildDiscoveryArgs(payload *core.Payload, tasConfig *core.TASConfigV2,
	subModule *core.SubModule,
	secretMap map[string]string,
	diffExists bool,
	diff map[string]int) core.DiscoveyArgs {
	testPattern := subModule.Patterns
	envMap := getEnv(payload, tasConfig, subModule)
	modulePath := path.Join(global.RepoDir, subModule.Path)

	return core.DiscoveyArgs{
		TestPattern:    testPattern,
		Payload:        payload,
		EnvMap:         envMap,
		SecretData:     secretMap,
		TestConfigFile: subModule.ConfigFile,
		FrameWork:      subModule.Framework,
		SmartRun:       tasConfig.SmartRun,
		Diff:           GetSubmoduleBasedDiff(diff, subModule.Path),
		DiffExists:     diffExists,
		CWD:            modulePath,
	}
}

func getEnv(payload *core.Payload, tasConfig *core.TASConfigV2, subModule *core.SubModule) map[string]string {
	var envMap map[string]string
	if payload.EventType == core.EventPullRequest {
		envMap = tasConfig.PreMerge.EnvMap
	} else {
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
	if path.Join(global.RepoDir, subModule.Path) == global.RepoDir {
		envMap[global.ModulePath] = ""
	} else {
		envMap[global.ModulePath] = subModule.Path
	}
	return envMap
}

func populateTestDiscoveryV2(testDiscoveryResult *core.DiscoveryResult, subModule *core.SubModule, tasConfig *core.TASConfigV2) {
	testDiscoveryResult.Parallelism = subModule.Parallelism
	testDiscoveryResult.SplitMode = tasConfig.SplitMode
	testDiscoveryResult.SubModule = subModule.Name
}

func (d *driverV2) findSubmodule(tasConfig *core.TASConfigV2, payload *core.Payload, subModuleName string) (*core.SubModule, error) {
	if payload.EventType == core.EventPullRequest {
		for i := 0; i < len(tasConfig.PreMerge.SubModules); i++ {
			if tasConfig.PreMerge.SubModules[i].Name == subModuleName {
				return &tasConfig.PreMerge.SubModules[i], nil
			}
		}
	} else {
		for i := 0; i < len(tasConfig.PostMerge.SubModules); i++ {
			if tasConfig.PostMerge.SubModules[i].Name == subModuleName {
				return &tasConfig.PostMerge.SubModules[i], nil
			}
		}
	}
	return nil, errs.ErrSubModuleNotFound
}

func (d *driverV2) buildTestExecutionArgs(payload *core.Payload,
	tasConfig *core.TASConfigV2,
	subModule *core.SubModule,
	secretMap map[string]string,
	coverageDir string) core.TestExecutionArgs {
	target := subModule.Patterns
	envMap := getEnv(payload, tasConfig, subModule)
	modulePath := path.Join(global.RepoDir, subModule.Path)

	azureLogWriter := logwriter.NewAzureLogWriter(d.AzureClient, core.PurposeExecutionLogs, d.logger)
	return core.TestExecutionArgs{
		Payload:           payload,
		CoverageDir:       coverageDir,
		LogWriterStrategy: azureLogWriter,
		TestPattern:       target,
		EnvMap:            envMap,
		TestConfigFile:    subModule.ConfigFile,
		FrameWork:         subModule.Framework,
		SecretData:        secretMap,
		CWD:               modulePath,
	}
}

func GetSubmoduleBasedDiff(diff map[string]int, subModulePath string) map[string]int {
	newDiff := map[string]int{}
	subModulePath = strings.TrimPrefix(subModulePath, "./")
	if !strings.HasSuffix(subModulePath, "/") {
		subModulePath += "/"
	}

	for file, value := range diff {
		filePath := strings.TrimPrefix(file, subModulePath)

		newDiff[filePath] = value
	}
	return newDiff
}

func (d *driverV2) setCache(tasConfig *core.TASConfigV2) error {
	if tasConfig.Cache == nil {
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
