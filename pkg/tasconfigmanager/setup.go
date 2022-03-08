// Package tasconfigmanager is used for fetching and validating the tas config file
package tasconfigmanager

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/global"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/utils"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"

	"gopkg.in/yaml.v2"
)

const (
	namespaceSeparator = "."
	emptyTagName       = "-"
	yamlTagName        = "yaml"
	requiredTagName    = "required"
	packageJSON        = "package.json"
)

// tasConfigManager represents an instance of TASConfigManager instance
type tasConfigManager struct {
	logger     lumber.Logger
	uni        *ut.UniversalTranslator
	validate   *validator.Validate
	translator ut.Translator
}

// NewTASConfigManager creates and returns a new TASConfigManager instance
func NewTASConfigManager(logger lumber.Logger) core.TASConfigManager {
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")
	validate := validator.New()
	en_translations.RegisterDefaultTranslations(validate, trans)
	configureValidator(validate, trans)

	return &tasConfigManager{logger: logger, uni: uni, validate: validate, translator: trans}
}

// LoadConfig used for loading and validating the  tas configuration values provided by user
func (tc *tasConfigManager) LoadConfig(ctx context.Context,
	path string,
	eventType core.EventType,
	parseMode bool) (*core.TASConfig, error) {

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

	tasConfig := &core.TASConfig{SmartRun: true, Tier: core.Small}
	err = yaml.Unmarshal(yamlFile, tasConfig)
	if err != nil {
		tc.logger.Errorf("Error while unmarshalling yaml file, path %s, error %v", path, err)
		return nil, errs.New("Invalid format of configuration file")
	}

	validateErr := tc.validate.Struct(tasConfig)
	if validateErr != nil {
		// translate all error at once
		errs := validateErr.(validator.ValidationErrors)

		errMsg := "Invalid values provided for the following fields in configuration file: \n"
		for _, e := range errs {
			// can translate each error one at a time.
			errMsg += fmt.Sprintf("%s: %s\n", e.Field(), e.Value())
		}

		tc.logger.Errorf("Error while validating yaml file, error %v", validateErr)
		return nil, errors.New(errMsg)

	}

	if !parseMode && tasConfig.Cache == nil {
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
	return tasConfig, nil
}

// configureValidator configure the struct validator
func configureValidator(validate *validator.Validate, trans ut.Translator) {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get(yamlTagName), ",", 2)[0]
		if name == emptyTagName {
			return fld.Name
		}
		return name
	})

	validate.RegisterTranslation(requiredTagName, trans, func(ut ut.Translator) error {
		return ut.Add(requiredTagName, "{0} field is required!", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		i := strings.Index(fe.Namespace(), namespaceSeparator)
		t, _ := ut.T(requiredTagName, fe.Namespace()[i+1:])
		return t
	})
}
