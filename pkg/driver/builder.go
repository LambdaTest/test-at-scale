package driver

import (
	"context"
	"fmt"
	"os"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

type (
	Builder struct {
		Logger               lumber.Logger
		TestExecutionService core.TestExecutionService
		AzureClient          core.AzureClient
		BlockTestService     core.BlockTestService
		ExecutionManager     core.ExecutionManager
		TASConfigManager     core.TASConfigManager
		CacheStore           core.CacheStore
		DiffManager          core.DiffManager
		ListSubModuleService core.ListSubModuleService
	}
	NodeInstaller struct {
		logger           lumber.Logger
		ExecutionManager core.ExecutionManager
	}
)

func (b *Builder) GetDriver(version int) (core.Driver, error) {
	if version < global.NewTASVersion {
		return &driverV1{
			logger:               b.Logger,
			TestExecutionService: b.TestExecutionService,
			AzureClient:          b.AzureClient,
			BlockTestService:     b.BlockTestService,
			ExecutionManager:     b.ExecutionManager,
			TASConfigManager:     b.TASConfigManager,
			CacheStore:           b.CacheStore,
			DiffManager:          b.DiffManager,
			ListSubModuleService: b.ListSubModuleService,
			nodeInstaller: NodeInstaller{
				logger:           b.Logger,
				ExecutionManager: b.ExecutionManager,
			},
		}, nil
	}
	return &driverV2{
		logger:               b.Logger,
		TestExecutionService: b.TestExecutionService,
		AzureClient:          b.AzureClient,
		BlockTestService:     b.BlockTestService,
		ExecutionManager:     b.ExecutionManager,
		TASConfigManager:     b.TASConfigManager,
		CacheStore:           b.CacheStore,
		DiffManager:          b.DiffManager,
		ListSubModuleService: b.ListSubModuleService,
		nodeInstaller: NodeInstaller{
			logger:           b.Logger,
			ExecutionManager: b.ExecutionManager,
		},
	}, nil
}

func (n *NodeInstaller) InstallNodeVersion(ctx context.Context, nodeVersion string) error {
	// Running the `source` commands in a directory where .nvmrc is present, exits with exitCode 3
	// https://github.com/nvm-sh/nvm/issues/1985
	// TODO [good-to-have]: Auto-read and install from .nvmrc file, if present
	commands := []string{
		"source /home/nucleus/.nvm/nvm.sh",
		fmt.Sprintf("nvm install %s", nodeVersion),
	}
	n.logger.Infof("Using user-defined node version: %v", nodeVersion)
	err := n.ExecutionManager.ExecuteInternalCommands(ctx, core.InstallNodeVer, commands, "", nil, nil)
	if err != nil {
		n.logger.Errorf("Unable to install user-defined nodeversion %v", err)
		err = errs.New(errs.GenericErrRemark.Error())
		return err
	}
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("/home/nucleus/.nvm/versions/node/v%s/bin:%s", nodeVersion, origPath))
	return nil
}
