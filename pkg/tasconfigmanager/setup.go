// Package tasconfigmanager is used for fetching and validating the tas config file
package tasconfigmanager

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

const packageJSON = "package.json"

var tierEnumMapping = map[core.Tier]int{
	core.XSmall: 1,
	core.Small:  2,
	core.Medium: 3,
	core.Large:  4,
	core.XLarge: 5,
}

// tasConfigManager represents an instance of TASConfigManager instance
type tasConfigManager struct {
	logger lumber.Logger
}

// NewTASConfigManager creates and returns a new TASConfigManager instance
func NewTASConfigManager(logger lumber.Logger) core.TASConfigManager {
	return &tasConfigManager{logger: logger}
}

func (tc *tasConfigManager) LoadAndValidate(ctx context.Context,
	path string,
	eventType core.EventType,
	licenseTier core.Tier) (*core.TASConfig, error) {
	path, err := utils.GetConfigFileName(path)
	if err != nil {
		return nil, err
	}

	yamlFile, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", global.RepoDir, path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.New(fmt.Sprintf("Configuration file not found at path: %s", path))
		}
		tc.logger.Errorf("Error while reading file, error %v", err)
		return nil, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", path))
	}

	tasConfig, err := utils.ValidateStruct(ctx, yamlFile)
	if err != nil {
		return nil, err
	}

	if tasConfig.Cache == nil {
		checksum, err := utils.ComputeChecksum(fmt.Sprintf("%s/%s", global.RepoDir, packageJSON))
		if err != nil {
			tc.logger.Errorf("Error while computing checksum, error %v", err)
			return nil, err
		}
		tasConfig.Cache = &core.Cache{
			Key:   checksum,
			Paths: []string{},
		}
	}

	if tasConfig.CoverageThreshold == nil {
		tasConfig.CoverageThreshold = new(core.CoverageThreshold)
	}

	switch eventType {
	case core.EventPullRequest:
		if tasConfig.Premerge == nil {
			return nil, errs.New("`preMerge` is not configured in configuration file")
		}
	case core.EventPush:
		if tasConfig.Postmerge == nil {
			return nil, errs.New("`postMerge` is not configured in configuration file")
		}
	}
	if err := isValidLicenseTier(tasConfig.Tier, licenseTier); err != nil {
		tc.logger.Errorf("LicenseTier validation failed. error: %v", err)
		return nil, err
	}
	return tasConfig, nil
}

func isValidLicenseTier(yamlLicense, currentLicense core.Tier) error {
	if tierEnumMapping[yamlLicense] > tierEnumMapping[currentLicense] {
		return fmt.Errorf("tier must not exceed max tier in license %v", currentLicense)
	}
	return nil
}
