package utils

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/bmatcuk/doublestar/v4"
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
)

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// ComputeChecksum compute the md5 hash for the given filename
func ComputeChecksum(filename string) (string, error) {
	checksum := ""

	file, err := os.Open(filename)
	if err != nil {
		return checksum, err
	}

	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return checksum, err
	}

	checksum = fmt.Sprintf("%x", hash.Sum(nil))
	return checksum, nil
}

// InterfaceToMap converts interface{} to map[string]string
func InterfaceToMap(in interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range in.(map[string]interface{}) {
		result[key] = value.(string)
	}
	return result
}

// CreateDirectory creates directory recursively if does not exists
func CreateDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, global.DirectoryPermissions); err != nil {
			return errs.ERR_DIR_CRT(err.Error())
		}
	}
	return nil
}

// WriteFileToDirectory writes `data` file to `filename`/`path`
func WriteFileToDirectory(path, filename string, data []byte) error {
	location := fmt.Sprintf("%s/%s", path, filename)
	if err := os.WriteFile(location, data, global.FilePermissions); err != nil {
		return errs.ERR_FIL_CRT(err.Error())
	}
	return nil
}

// GetOutboundIP returns preferred outbound ip of this container
func GetOutboundIP() string {
	return global.SynapseContainerURL
}

// GetConfigFileName returns the name of the configuration file
func GetConfigFileName(path string) (string, error) {
	if global.TestEnv {
		return path, nil
	}
	ext := filepath.Ext(path)
	// Add support for both yaml extensions
	if ext == ".yaml" || ext == ".yml" {
		matches, _ := doublestar.Glob(os.DirFS(global.RepoDir), strings.TrimSuffix(path, ext)+".{yml,yaml}")
		if len(matches) == 0 {
			return "", errs.New(
				fmt.Sprintf(
					"`%s` configuration file not found at the root of your project. Please make sure you have placed it correctly.",
					path))
		}
		// If there are files with the both extensions, pick the first match
		path = matches[0]
	}
	return path, nil
}

func ValidateStructTASYmlV1(ctx context.Context, ymlContent []byte) (*core.TASConfig, error) {
	validate, err := getValidator()
	if err != nil {
		return nil, err
	}
	tasConfig := &core.TASConfig{SmartRun: true, Tier: core.Small, SplitMode: core.TestSplit}
	if err := yaml.Unmarshal(ymlContent, tasConfig); err != nil {
		return nil, fmt.Errorf("Error in unmarshling tas yml file")
	}
	if err := validateStruct(validate, tasConfig); err != nil {
		return nil, err
	}
	return tasConfig, nil
}

// configureValidator configure the struct validator
func configureValidator(validate *validator.Validate, trans ut.Translator) {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// nolint: gomnd
		name := strings.SplitN(fld.Tag.Get(yamlTagName), ",", 2)[0]
		if name == emptyTagName {
			return fld.Name
		}
		return name
	})

	// nolint: errcheck
	validate.RegisterTranslation(requiredTagName, trans, func(ut ut.Translator) error {
		return ut.Add(requiredTagName, "{0} field is required!", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		i := strings.Index(fe.Namespace(), namespaceSeparator)
		t, _ := ut.T(requiredTagName, fe.Namespace()[i+1:])
		return t
	})
}

func GetVersion(ymlContent []byte) (float32, error) {
	tasVersion := &core.TasVersion{}
	if err := yaml.Unmarshal(ymlContent, tasVersion); err != nil {
		return 0.0, fmt.Errorf("Error in unmarshling tas yml file")
	}

	return tasVersion.Version, nil
}

func ValidateStructTASYmlV2(ctx context.Context, ymlContent []byte) (*core.TASConfigV2, error) {

	tasConfig := &core.TASConfigV2{SmartRun: true, Tier: core.Small, SplitMode: core.TestSplit}
	if err := yaml.Unmarshal(ymlContent, tasConfig); err != nil {
		return nil, fmt.Errorf("Error in unmarshling tas yml file")
	}
	validate, err := getValidator()
	if err != nil {
		return nil, err
	}
	if err := validateStruct(validate, tasConfig); err != nil {
		return nil, err
	}

	return tasConfig, nil
}

func getValidator() (*validator.Validate, error) {
	enObj := en.New()
	uni := ut.New(enObj, enObj)
	trans, _ := uni.GetTranslator("en")
	validate := validator.New()
	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, err
	}
	configureValidator(validate, trans)
	return validate, nil
}

func validateStruct(validate *validator.Validate, config interface{}) error {
	validateErr := validate.Struct(config)
	if validateErr != nil {
		// translate all error at once
		validationErrs := validateErr.(validator.ValidationErrors)
		err := new(errs.ErrInvalidConf)
		for _, e := range validationErrs {
			// can translate each error one at a time.
			err.Fields = append(err.Fields, e.Field())
			err.Values = append(err.Values, e.Value())
		}
		return err
	}
	return nil
}
