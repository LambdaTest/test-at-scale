package tasconfigdownloader

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/gitmanager"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/tasconfigmanager"
)

const ymlVersionMismtachRemarks = "the yml structure is invalid, please check the TAS yml documentation : %s"

type TASConfigDownloader struct {
	logger           lumber.Logger
	gitmanager       core.GitManager
	tasconfigmanager core.TASConfigManager
}

func New(logger lumber.Logger) *TASConfigDownloader {
	return &TASConfigDownloader{
		logger:           logger,
		gitmanager:       gitmanager.NewGitManager(logger, nil),
		tasconfigmanager: tasconfigmanager.NewTASConfigManager(logger),
	}
}

func (t *TASConfigDownloader) GetTASConfig(ctx context.Context, gitProvider, commitID, repoSlug,
	filePath string, oauth *core.Oauth, eventType core.EventType, licenseTier core.Tier) (*core.TASConfigDownloaderOutput, error) {
	ymlPath, err := t.gitmanager.DownloadFileByCommit(ctx, gitProvider, repoSlug, commitID, filePath, oauth)
	if err != nil {
		t.logger.Errorf("error occurred while downloading file %s from %s for commitID %s, error %v", filePath, repoSlug, commitID, err)
		return nil, err
	}

	version, err := t.tasconfigmanager.GetVersion(ymlPath)
	if err != nil {
		t.logger.Errorf("error reading version for tas config file %s, error %v", ymlPath, err)
		return nil, err
	}

	tasConfig, err := t.tasconfigmanager.LoadAndValidate(ctx, version, ymlPath, eventType, licenseTier, filePath)
	if err != nil {
		if supportedVersion := t.checkYmlValidityForOtherVersion(ctx, version, ymlPath, eventType,
			licenseTier, filePath); supportedVersion != -1 {
			errMsg := fmt.Sprintf(ymlVersionMismtachRemarks, global.TASYmlConfigurationDocLink)
			t.logger.Errorf("error while parsing yml for commitID %s, error: %s", commitID, errMsg)
			return nil, errors.New(errMsg)
		}
		t.logger.Errorf("error while parsing yml for commitID %s error %v", commitID, err)
		return nil, err
	}
	if err := os.Remove(ymlPath); err != nil {
		t.logger.Errorf("failed to delete file %s , error %v", ymlPath, err)
		return nil, err
	}
	return &core.TASConfigDownloaderOutput{Version: version, TASConfig: tasConfig}, nil
}

func (t *TASConfigDownloader) checkYmlValidityForOtherVersion(ctx context.Context,
	version int,
	ymlPath string,
	eventType core.EventType,
	licenseTier core.Tier, filePath string) int {
	for _, supportedVersion := range global.ValidYMLVersions {
		if version == supportedVersion {
			continue
		}
		if _, err := t.tasconfigmanager.LoadAndValidate(ctx, supportedVersion, ymlPath, eventType, licenseTier, filePath); err == nil {
			return supportedVersion
		}
	}
	return -1
}
