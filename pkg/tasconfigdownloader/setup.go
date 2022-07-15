package tasconfigdownloader

import (
	"context"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/gitmanager"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/tasconfigmanager"
)

type TASConfigDownloaderOutput struct {
	Version   int
	TasConfig interface{}
}

func GetTasConfig(ctx context.Context, logger lumber.Logger, gitProvider, commitID, repoSlug,
	filePath string, oauth *core.Oauth, eventType core.EventType, licenseTier core.Tier) (*TASConfigDownloaderOutput, error) {
	gm := gitmanager.NewGitManager(logger, nil)
	ymlPath, err := gm.DownloadFileByCommit(ctx, gitProvider, repoSlug, commitID, filePath, oauth)
	if err != nil {
		logger.Errorf("error occurred while downloading file %s from %s for commitID %s, error %v", filePath, repoSlug, commitID, err)
		return nil, err
	}
	tcm := tasconfigmanager.NewTASConfigManager(logger)

	version, err := tcm.GetVersion(ymlPath)
	if err != nil {
		logger.Errorf("error reading version for tas config file %s, error %v", ymlPath, err)
		return nil, err
	}
	tasConfig, err := tcm.LoadAndValidate(ctx, version, ymlPath, eventType, licenseTier)
	if err != nil {
		logger.Errorf("error while parsing yml , error %v", err)
		return nil, err
	}
	return &TASConfigDownloaderOutput{Version: version, TasConfig: tasConfig}, nil

}
