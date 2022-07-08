// Package tasconfigmanager is used for fetching and validating the tas config file
package tasconfigmanager

import (
	"context"
	"errors"
	"fmt"
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
var getTasFilePathFn = getTasFilePath

// tasConfigManager represents an instance of TASConfigManager instance
type tasConfigManager struct {
	logger lumber.Logger
}

// NewTASConfigManager creates and returns a new TASConfigManager instance
func NewTASConfigManager(logger lumber.Logger) core.TASConfigManager {
	return &tasConfigManager{logger: logger}
}

func (tc *tasConfigManager) LoadAndValidate(ctx context.Context,
	version int,
	path string,
	eventType core.EventType,
	licenseTier core.Tier) (interface{}, error) {
	if version < global.NewTASVersion {
		return tc.loadAndValidateV1(ctx, path, eventType, licenseTier)
	}
	return tc.loadAndValidateV2(ctx, path, eventType, licenseTier)
}

func (tc *tasConfigManager) loadAndValidateV1(ctx context.Context,
	path string,
	eventType core.EventType,
	licenseTier core.Tier) (*core.TASConfig, error) {
	filePath, err := getTasFilePathFn(path)
	if err != nil {
		return nil, err
	}
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		tc.logger.Errorf("Error while reading file %s, error %v", filePath, err)
		return nil, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", path))
	}
	return tc.validateYMLV1(ctx, yamlFile, eventType, licenseTier, filePath)
}

func (tc *tasConfigManager) validateYMLV1(ctx context.Context,
	yamlFile []byte,
	eventType core.EventType,
	licenseTier core.Tier,
	filePath string) (*core.TASConfig, error) {
	tasConfig, err := utils.ValidateStructTASYmlV1(ctx, yamlFile, filePath)
	if err != nil {
		return nil, err
	}

	if tasConfig.CoverageThreshold == nil {
		tasConfig.CoverageThreshold = new(core.CoverageThreshold)
	}

	switch eventType {
	case core.EventPullRequest:
		if tasConfig.Premerge == nil {
			return nil, errs.New(fmt.Sprintf("`preMerge` test cases are not configured in `%s` configuration file.", filePath))
		}
	case core.EventPush:
		if tasConfig.Postmerge == nil {
			return nil, errs.New(fmt.Sprintf("`postMerge` test cases are not configured in `%s` configuration file.", filePath))
		}
	}
	if err := isValidLicenseTier(tasConfig.Tier, licenseTier); err != nil {
		tc.logger.Errorf("LicenseTier validation failed. error: %v", err)
		return nil, err
	}

	language := global.FrameworkLanguageMap[tasConfig.Framework]
	if tasConfig.Cache == nil && language == "javascript" {
		checksum, err := utils.ComputeChecksum(fmt.Sprintf("%s/%s", global.RepoDir, global.PackageJSON))
		if err != nil {
			tc.logger.Errorf("Error while computing checksum, error %v", err)
			return nil, err
		}
		tasConfig.Cache = &core.Cache{
			Key:     checksum,
			Paths:   []string{},
			Version: global.CacheVersion,
		}
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

func (tc *tasConfigManager) validateYMLV2(ctx context.Context,
	yamlFile []byte,
	eventType core.EventType,
	licenseTier core.Tier,
	yamlFilePath string) (*core.TASConfigV2, error) {
	tasConfig, err := utils.ValidateStructTASYmlV2(ctx, yamlFile, yamlFilePath)
	if err != nil {
		return nil, err
	}

	if tasConfig.CoverageThreshold == nil {
		tasConfig.CoverageThreshold = new(core.CoverageThreshold)
	}

	switch eventType {
	case core.EventPullRequest:
		if tasConfig.PreMerge == nil {
			return nil, fmt.Errorf("`preMerge` is missing in tas configuration file %s", yamlFilePath)
		}
		subModuleMap := map[string]bool{}
		for i := 0; i < len(tasConfig.PreMerge.SubModules); i++ {
			if err := utils.ValidateSubModule(&tasConfig.PreMerge.SubModules[i]); err != nil {
				return nil, err
			}
			if _, ok := subModuleMap[tasConfig.PreMerge.SubModules[i].Name]; ok {
				return nil, fmt.Errorf("duplicate subModule name found in `preMerge` in tas configuration file %s", yamlFilePath)
			}
			subModuleMap[tasConfig.PreMerge.SubModules[i].Name] = true
		}

	case core.EventPush:
		if tasConfig.PostMerge == nil {
			return nil, fmt.Errorf("`postMerge` is missing in tas configuration file %s", yamlFilePath)
		}
		subModuleMap := map[string]bool{}

		for i := 0; i < len(tasConfig.PostMerge.SubModules); i++ {
			if err := utils.ValidateSubModule(&tasConfig.PostMerge.SubModules[i]); err != nil {
				return nil, err
			}
			if _, ok := subModuleMap[tasConfig.PostMerge.SubModules[i].Name]; ok {
				return nil, fmt.Errorf("duplicate subModule name found in `postMerge` in tas configuration file %s", yamlFilePath)
			}
			subModuleMap[tasConfig.PostMerge.SubModules[i].Name] = true
		}
	}
	if err := isValidLicenseTier(tasConfig.Tier, licenseTier); err != nil {
		tc.logger.Errorf("LicenseTier validation failed. error: %v", err)
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
	return tasConfig, nil
}

func (tc *tasConfigManager) GetVersion(path string) (int, error) {
	yamlFile, err := os.ReadFile(fmt.Sprintf("%s/%s", global.RepoDir, path))
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

func (tc *tasConfigManager) loadAndValidateV2(ctx context.Context,
	path string,
	eventType core.EventType,
	licenseTier core.Tier) (*core.TASConfigV2, error) {
	yamlFilePath, err := getTasFilePathFn(path)
	if err != nil {
		return nil, err
	}
	yamlFile, err := os.ReadFile(yamlFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.New(fmt.Sprintf("Configuration file not found at path: %s", path))
		}
		tc.logger.Errorf("Error while reading file, error %v", err)
		return nil, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", path))
	}
	return tc.validateYMLV2(ctx, yamlFile, eventType, licenseTier, yamlFilePath)
}

func getTasFilePath(path string) (string, error) {
	path, err := utils.GetConfigFileName(path)
	if err != nil {
		return "", err
	}
	filePath := fmt.Sprintf("%s/%s", global.RepoDir, path)
	return filePath, nil
}
