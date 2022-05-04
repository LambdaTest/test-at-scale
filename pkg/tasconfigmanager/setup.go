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
const newYMLVersion = 2

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

func (tc *tasConfigManager) LoadAndValidateV1(ctx context.Context,
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
	return tc.validateYMLV1(ctx, yamlFile, eventType, licenseTier)
}

func (tc *tasConfigManager) validateYMLV1(ctx context.Context, yamlFile []byte, eventType core.EventType, licenseTier core.Tier) (*core.TASConfig, error) {
	tasConfig, err := utils.ValidateStructTASYmlV1(ctx, yamlFile)
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

func (tc *tasConfigManager) validateYMLV2(ctx context.Context, yamlFile []byte, eventType core.EventType, licenseTier core.Tier) (*core.TASConfigV2, error) {
	tasConfig, err := utils.ValidateStructTASYmlV2(ctx, yamlFile)
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

		for _, module := range tasConfig.PreMerge.SubModules {
			if err := validateModule(&module); err != nil {
				return nil, err
			}
		}

	case core.EventPush:
		for _, module := range tasConfig.PostMerge.SubModules {
			if err := validateModule(&module); err != nil {
				return nil, err
			}
		}
	}
	if err := isValidLicenseTier(tasConfig.Tier, licenseTier); err != nil {
		tc.logger.Errorf("LicenseTier validation failed. error: %v", err)
		return nil, err
	}
	return tasConfig, nil
}

func validateModule(module *core.SubModule) error {
	if module.Name == "" {
		errs.New("module name is not defined")
	}
	if module.Path == "" {
		return errs.New(fmt.Sprintf("module path is not defined for module %s ", module.Name))
	}
	if len(module.Patterns) == 0 {
		return errs.New(fmt.Sprintf("module %s pattern length is 0", module.Name))
	}

	return nil
}

func (tc *tasConfigManager) GetVersion(path string) (float32, error) {
	yamlFile, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", global.RepoDir, path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, errs.New(fmt.Sprintf("Configuration file not found at path: %s", path))
		}
		tc.logger.Errorf("Error while reading file, error %v", err)
		return 0, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", path))
	}
	versionYml, err := utils.GetVersion(yamlFile)
	if err != nil {
		tc.logger.Errorf("Error while reading tas yml version error %v", err)
		return 0, errs.New("Error while reading tas yml version")
	}
	return versionYml, nil
}

func (tc *tasConfigManager) LoadAndValidateV2(ctx context.Context,
	path string,
	eventType core.EventType,
	licenseTier core.Tier) (*core.TASConfigV2, error) {
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
	return tc.validateYMLV2(ctx, yamlFile, eventType, licenseTier)
}
