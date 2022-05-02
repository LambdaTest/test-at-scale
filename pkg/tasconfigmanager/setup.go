// Package tasconfigmanager is used for fetching and validating the tas config file
package tasconfigmanager

import (
	"context"
	"fmt"
	"io/ioutil"

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
		tc.logger.Errorf("Error while reading file, error %v", err)
		return nil, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", path))
	}

	tasConfig, err := utils.ValidateStruct(ctx, yamlFile, path)
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
			return nil, errs.New(fmt.Sprintf("`preMerge` test cases are not configured in `%s` configuration file.", path))
		}
	case core.EventPush:
		if tasConfig.Postmerge == nil {
			return nil, errs.New(fmt.Sprintf("`postMerge` test cases are not configured in `%s` configuration file.", path))
		}
	}
	if err := isValidLicenseTier(tasConfig.Tier, licenseTier); err != nil {
		tc.logger.Errorf("LicenseTier validation failed. error: %v", err)
		return nil, err
	}
	return tasConfig, nil
}

func isValidLicenseTier(yamlTier, licenseTier core.Tier) error {
	if tierEnumMapping[yamlTier] > tierEnumMapping[licenseTier] {
		return errs.New(
			fmt.Sprintf(
				"Sorry, the requested tier `%s` is not supported under the current plan. Please upgrade your plan.",
				yamlTier))
	}
	return nil
}
