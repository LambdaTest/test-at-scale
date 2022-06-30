package driver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/logwriter"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
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
	}

	setUpResultV2 struct {
		diffExists bool
		diff       map[string]int
		cacheKey   string
	}
)

func (m *driverV2) RunDiscovery(ctx context.Context, payload *core.Payload,
	taskPayload *core.TaskPayload, oauth *core.Oauth, coverageDir string, secretMap map[string]string) error {
	// do something
	tas, err := m.TASConfigManager.LoadAndValidate(ctx, m.TASVersion, payload.TasFileName, payload.EventType, payload.LicenseTier)
	if err != nil {
		m.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	tasConfig := tas.(*core.TASConfigV2)
	taskPayload.Status = core.Passed
	setUpResult, err := m.setUpDiscovery(ctx, payload, tasConfig, oauth)
	if err != nil {
		return err
	}
	mainBuffer := new(bytes.Buffer)
	blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, os.Getenv("TASK_ID"), core.PreRun)
	azureLogWriter := logwriter.NewAzureLogWriter(m.AzureClient, blobPath, m.logger)

	defer func() {
		m.logger.Debugf("Writing the preRUN logs to path %s", blobPath)
		if writeErr := <-azureLogWriter.Write(ctx, mainBuffer); writeErr != nil {
			// error in writing log should not fail the build
			m.logger.Errorf("error in writing pre run log, error %v", writeErr)
		}
	}()

	if payload.EventType == core.EventPush {
		if discoveryErr := m.runDiscoveryV2Helper(ctx, tasConfig.PostMerge.PreRun,
			tasConfig.PostMerge.SubModules, payload, tasConfig,
			taskPayload, setUpResult.diff, setUpResult.diffExists, mainBuffer, secretMap); discoveryErr != nil {
			return discoveryErr
		}
	} else {
		if discoveryErr := m.runDiscoveryV2Helper(ctx, tasConfig.PreMerge.PreRun, tasConfig.PreMerge.SubModules,
			payload, tasConfig, taskPayload, setUpResult.diff, setUpResult.diffExists, mainBuffer, secretMap); discoveryErr != nil {
			return discoveryErr
		}
	}
	if err = m.CacheStore.Upload(ctx, setUpResult.cacheKey, tasConfig.Cache.Paths...); err != nil {
		// cache upload failure should not fail the task
		m.logger.Errorf("Unable to upload cache: %v", err)
	}
	m.logger.Debugf("Cache uploaded successfully")

	return nil
}

func (m *driverV2) RunExecution(ctx context.Context, payload *core.Payload,
	taskPayload *core.TaskPayload, oauth *core.Oauth, coverageDir string, secretMap map[string]string) error {
	tas, err := m.TASConfigManager.LoadAndValidate(ctx, m.TASVersion, payload.TasFileName, payload.EventType, payload.LicenseTier)
	if err != nil {
		m.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		err = &errs.StatusFailed{Remark: err.Error()}
		return err
	}
	subModuleName := os.Getenv(global.SubModuleName)
	tasConfig := tas.(*core.TASConfigV2)
	subModule, err := m.findSubmodule(tasConfig, payload, subModuleName)
	if err != nil {
		m.logger.Errorf("Error finding sub module %s in tas config file", subModuleName)
		return err
	}
	// Get blocklist data before execution
	blYML := subModule.Blocklist
	if err = m.BlockTestService.GetBlockTests(ctx, blYML, payload.BranchName); err != nil {
		m.logger.Errorf("Unable to fetch blocklisted tests: %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	/*
		1. run PRE run steps
		2. run Test execution
		3. run POST run steps
	*/

	modulePath := path.Join(global.RepoDir, subModule.Path)
	// PRE RUN steps should be run only if RunPrerunEveryTime is set to true
	if subModule.Prerun != nil && subModule.RunPrerunEveryTime {
		m.logger.Infof("Running pre-run steps for submodule %s", subModule.Name)
		blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, os.Getenv("TASK_ID"), core.PreRun)

		azureLogwriter := logwriter.NewAzureLogWriter(m.AzureClient, blobPath, m.logger)
		err = m.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload, subModule.Prerun, secretMap, azureLogwriter, modulePath)
		if err != nil {
			m.logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
		if err = m.ExecutionManager.ExecuteInternalCommands(ctx, core.InstallRunners, global.InstallRunnerCmds,
			global.RepoDir, nil, nil); err != nil {
			m.logger.Errorf("Unable to install custom runners %v", err)
			err = errs.New(errs.GenericErrRemark.Error())
			return err
		}
	}
	args := m.buildTestExecutionArgs(payload, tasConfig, subModule, secretMap, coverageDir)
	testResult, err := m.TestExecutionService.Run(ctx, args)
	if err != nil {
		return err
	}
	resp, err := m.TestExecutionService.SendResults(ctx, testResult)
	if err != nil {
		m.logger.Errorf("error while sending test reports %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	taskPayload.Status = resp.TaskStatus

	if subModule.Postrun != nil {
		m.logger.Infof("Running post-run steps")
		blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, os.Getenv("TASK_ID"), core.PostRun)

		azureLogwriter := logwriter.NewAzureLogWriter(m.AzureClient, blobPath, m.logger)

		err = m.ExecutionManager.ExecuteUserCommands(ctx, core.PostRun, payload, subModule.Postrun, secretMap, azureLogwriter, modulePath)
		if err != nil {
			m.logger.Errorf("Unable to run post-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running post-run steps."}
			return err
		}
	}
	return nil
}

func (m *driverV2) runDiscoveryV2Helper(ctx context.Context,
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
	if apiErr := m.ListSubModuleService.Send(ctx, payload.BuildID, totalSubmoduleCount); apiErr != nil {
		return apiErr
	}
	errChannelPreRun := make(chan error, totalSubmoduleCount)
	preRunWaitGroup := sync.WaitGroup{}
	if tasConfig.NodeVersion != "" {
		if err := m.nodeInstaller.InstallNodeVersion(ctx, tasConfig.NodeVersion); err != nil {
			return err
		}
	}
	if topPreRun != nil {
		m.logger.Debugf("Running Pre Run on top level")
		if _, err := mainBuffer.WriteString(preRunLog); err != nil {
			return err
		}
		bufferWirter := logwriter.NewABufferLogWriter("TOP-LEVEL", mainBuffer, m.logger)
		if err := m.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload,
			topPreRun, secretMap, bufferWirter, global.RepoDir); err != nil {
			m.logger.Errorf("Error occurred running top level PreRun , err %v", err)
			return err
		}
	}
	bufferList := []*bytes.Buffer{}
	m.logger.Debugf("pre run on top level ended")
	for i := 0; i < totalSubmoduleCount; i++ {
		preRunWaitGroup.Add(1)
		newBuffer := new(bytes.Buffer)
		bufferList = append(bufferList, newBuffer)
		go func(subModule *core.SubModule) {
			defer preRunWaitGroup.Done()
			bufferWirterSubmodule := logwriter.NewABufferLogWriter(subModule.Name, newBuffer, m.logger)
			dicoveryErr := m.runPreRunForEachSubModule(ctx, payload, subModule, secretMap, bufferWirterSubmodule)
			if dicoveryErr != nil {
				taskPayload.Status = core.Error
				m.logger.Errorf("error while running discovery for sub module %s, error %v", subModule.Name, dicoveryErr)
			}
			errChannelPreRun <- dicoveryErr
		}(&subModuleList[i])
	}
	preRunWaitGroup.Wait()
	for i := 0; i < totalSubmoduleCount; i++ {
		mainBuffer.WriteString(bufferList[i].String())
	}
	m.logger.Debugf("checking the pre runs errors ")
	for i := 0; i < totalSubmoduleCount; i++ {
		e := <-errChannelPreRun
		if e != nil {
			return e
		}
	}
	m.logger.Debugf("checked the pre run errors")
	m.logger.Debugf("Caching workspace")
	// TODO: this will be change after we move to parallel pod executuon
	if err := m.CacheStore.CacheWorkspace(ctx, ""); err != nil {
		m.logger.Errorf("Error caching workspace: %+v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	errChannelDiscovery := make(chan error, totalSubmoduleCount)
	discoveryWaitGroup := sync.WaitGroup{}
	for i := 0; i < totalSubmoduleCount; i++ {
		discoveryWaitGroup.Add(1)
		go func(subModule *core.SubModule) {
			defer discoveryWaitGroup.Done()
			err := m.runDiscoveryForEachSubModule(ctx, payload, subModule, tasConfig, diff, diffExists, secretMap)
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

func (m *driverV2) runDiscoveryForEachSubModule(ctx context.Context,
	payload *core.Payload,
	subModule *core.SubModule,
	tasConfig *core.TASConfigV2,
	diff map[string]int,
	diffExists bool,
	secretMap map[string]string) error {
	args := m.buildDiscoveryArgs(payload, tasConfig, subModule, secretMap, diffExists, diff)

	discoveryResult, err := m.TestDiscoveryService.Discover(ctx, args)
	if err != nil {
		m.logger.Errorf("Unable to perform test discovery: %+v", err)
		err = &errs.StatusFailed{Remark: "Failed in discovering tests"}
		return err
	}
	populateTestDiscoveryV2(discoveryResult, subModule, tasConfig)
	if err := m.TestDiscoveryService.SendResult(ctx, discoveryResult); err != nil {
		return err
	}
	return nil
}

func (m *driverV2) runPreRunForEachSubModule(ctx context.Context,
	payload *core.Payload,
	subModule *core.SubModule,
	secretMap map[string]string,
	bufferWirterSubmodule core.LogWriterStartegy) error {
	m.logger.Debugf("Running discovery for sub module %s", subModule.Name)
	blYML := subModule.Blocklist
	if err := m.BlockTestService.GetBlockTests(ctx, blYML, payload.BranchName); err != nil {
		m.logger.Errorf("Unable to fetch blocklisted tests: %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	modulePath := path.Join(global.RepoDir, subModule.Path)
	// PRE RUN steps
	if subModule.Prerun != nil {
		m.logger.Infof("Running pre-run steps for submodule %s", subModule.Name)
		err := m.ExecutionManager.ExecuteUserCommands(ctx, core.PreRun, payload, subModule.Prerun,
			secretMap, bufferWirterSubmodule, modulePath)
		if err != nil {
			m.logger.Errorf("Unable to run pre-run steps %v", err)
			err = &errs.StatusFailed{Remark: "Failed in running pre-run steps"}
			return err
		}
		m.logger.Debugf("error checks end")
	}
	err := m.ExecutionManager.ExecuteInternalCommands(ctx, core.InstallRunners, global.InstallRunnerCmds, modulePath, nil, nil)
	if err != nil {
		m.logger.Errorf("Unable to install custom runners %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}

	return nil
}

func (m *driverV2) setUpDiscovery(ctx context.Context,
	payload *core.Payload,
	tasConfig *core.TASConfigV2,
	oauth *core.Oauth) (*setUpResultV2, error) {
	cacheKey := fmt.Sprintf("%s/%s/%s/%s", tasConfig.Cache.Version, payload.OrgID, payload.RepoID, tasConfig.Cache.Key)

	g, errCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if errG := m.CacheStore.Download(errCtx, cacheKey); errG != nil {
			m.logger.Errorf("Unable to download cache: %v", errG)
			errG = errs.New(errs.GenericErrRemark.Error())
			return errG
		}
		return nil
	})
	diffExists := true
	diff := map[string]int{}
	g.Go(func() error {
		diffC, errG := m.DiffManager.GetChangedFiles(errCtx, payload, oauth)
		if errG != nil {
			if errors.Is(errG, errs.ErrGitDiffNotFound) {
				diffExists = false
			} else {
				m.logger.Errorf("Unable to identify changed files %s", errG)
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

func (m *driverV2) buildDiscoveryArgs(payload *core.Payload, tasConfig *core.TASConfigV2,
	subModule *core.SubModule,
	secretMap map[string]string,
	diffExists bool,
	diff map[string]int) core.DiscoveyArgs {
	testPattern := subModule.Patterns
	envMap := getEnv(payload, tasConfig, subModule)
	return core.DiscoveyArgs{
		TestPattern:    testPattern,
		Payload:        payload,
		EnvMap:         envMap,
		SecretData:     secretMap,
		TestConfigFile: subModule.ConfigFile,
		FrameWork:      subModule.Framework,
		SmartRun:       tasConfig.SmartRun,
		Diff:           diff,
		DiffExists:     diffExists,
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
	testDiscoveryResult.Tier = tasConfig.Tier
	testDiscoveryResult.ContainerImage = tasConfig.ContainerImage
}

func (m *driverV2) findSubmodule(tasConfig *core.TASConfigV2, payload *core.Payload, subModuleName string) (*core.SubModule, error) {
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

func (m *driverV2) buildTestExecutionArgs(payload *core.Payload,
	tasConfig *core.TASConfigV2,
	subModule *core.SubModule,
	secretMap map[string]string,
	coverageDir string) core.TestExecutionArgs {
	target := subModule.Patterns
	blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, os.Getenv("TASK_ID"), core.Execution)
	envMap := getEnv(payload, tasConfig, subModule)

	azureLogWriter := logwriter.NewAzureLogWriter(m.AzureClient, blobPath, m.logger)
	return core.TestExecutionArgs{
		Payload:           payload,
		CoverageDir:       coverageDir,
		LogWriterStartegy: azureLogWriter,
		TestPattern:       target,
		EnvMap:            envMap,
		TestConfigFile:    subModule.ConfigFile,
		FrameWork:         subModule.Framework,
		SecretData:        secretMap,
	}
}
