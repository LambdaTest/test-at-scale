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

const (
	v1 = 1
	v2 = 2
)

// tasConfigManager represents an instance of TASConfigManager instance
type tasConfigManager struct {
	secretParser core.SecretParser
	logger       lumber.Logger
}

// NewTASConfigManager creates and returns a new TASConfigManager instance
func NewTASConfigManager(logger lumber.Logger,
	secretParser core.SecretParser) core.TASConfigManager {
	return &tasConfigManager{logger: logger, secretParser: secretParser}
}

func (tc *tasConfigManager) LoadAndValidate(ctx context.Context,
	version int,
	path string,
	eventType core.EventType,
	licenseTier core.Tier,
	secretMap map[string]string,
	tasFilePathInRepo string) (interface{}, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.New(fmt.Sprintf("Configuration file not found at path: %s", tasFilePathInRepo))
		}
		tc.logger.Errorf("Error while reading file, error %v", err)
		return nil, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", tasFilePathInRepo))
	}
	if version < global.NewTASVersion {
		return tc.validateYMLV1(ctx, yamlFile, eventType, licenseTier, secretMap, tasFilePathInRepo)
	}
	return tc.validateYMLV2(ctx, yamlFile, eventType, licenseTier, secretMap, tasFilePathInRepo)
}

func (tc *tasConfigManager) validateYMLV1(ctx context.Context,
	yamlFile []byte,
	eventType core.EventType,
	licenseTier core.Tier,
	secretMap map[string]string,
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
	if err := tc.validateTASWithRepoSecretV1(secretMap, tasConfig); err != nil {
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

func (tc *tasConfigManager) validateYMLV2(ctx context.Context,
	yamlFile []byte,
	eventType core.EventType,
	licenseTier core.Tier,
	secretMap map[string]string,
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
	if err := tc.validateTASWithRepoSecretV2(eventType, secretMap, tasConfig); err != nil {
		return nil, err
	}

	return tasConfig, nil
}

func (tc *tasConfigManager) GetVersion(path string) (int, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, errs.New(fmt.Sprintf("Configuration file not found at path: %s", path))
		}
		tc.logger.Errorf("Error while reading file, error %v", err)
		return 0, errs.New(fmt.Sprintf("Error while reading configuration file at path: %s", path))
	}
	versionYml, err := utils.GetVersion(yamlFile)
	if err != nil {
		tc.logger.Errorf("error while reading tas yml version : %v", err)
		return 0, err
	}
	return versionYml, nil
}

func (tc *tasConfigManager) GetTasConfigFilePath(payload *core.Payload) (string, error) {
	// load tas yaml file
	filePath, err := utils.GetTASFilePath(payload.TasFileName)
	if err != nil {
		tc.logger.Errorf("Unable to load tas yaml file, error: %v", err)
		return "", err
	}
	return filePath, nil
}

func (tc *tasConfigManager) validateTASWithRepoSecretV1(secretMap map[string]string,
	tasConfig *core.TASConfig) error {
	if tasConfig.Prerun == nil {
		return nil
	}
	commands := tasConfig.Prerun.Commands
	for _, cmd := range commands {
		if err := tc.secretParser.ValidateRepoSecret(cmd, secretMap); err != nil {
			return err
		}
	}
	for _, val := range tasConfig.Prerun.EnvMap {
		if err := tc.secretParser.ValidateRepoSecret(val, secretMap); err != nil {
			return err
		}
	}
	return nil
}

func (tc *tasConfigManager) validateTASWithRepoSecretV2(eventType core.EventType,
	secretMap map[string]string,
	tasConfig *core.TASConfigV2) error {
	switch eventType {
	case core.EventPullRequest:
		if tasConfig.PreMerge != nil {
			for i := 0; i < len(tasConfig.PreMerge.SubModules); i++ {
				subModule := tasConfig.PreMerge.SubModules[i]
				if err := tc.validateSubModuleWithRepoSecret(secretMap, &subModule); err != nil {
					return err
				}
			}
		}
	case core.EventPush:
		for i := 0; i < len(tasConfig.PostMerge.SubModules); i++ {
			subModule := tasConfig.PostMerge.SubModules[i]
			if err := tc.validateSubModuleWithRepoSecret(secretMap, &subModule); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("event type %s is not supported", eventType)
	}
	return nil
}

func (tc *tasConfigManager) validateSubModuleWithRepoSecret(secretMap map[string]string,
	subModule *core.SubModule) error {
	if subModule.Prerun == nil {
		return nil
	}
	commands := subModule.Prerun.Commands
	for _, cmd := range commands {
		if err := tc.secretParser.ValidateRepoSecret(cmd, secretMap); err != nil {
			return err
		}
	}
	for _, val := range subModule.Prerun.EnvMap {
		if err := tc.secretParser.ValidateRepoSecret(val, secretMap); err != nil {
			return err
		}
	}
	return nil
}
