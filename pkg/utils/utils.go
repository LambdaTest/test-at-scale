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
			return "", errs.New(fmt.Sprintf("Configuration file not found at path: %s", path))
		}
		// If there are files with the both extensions, pick the first match
		path = matches[0]
	}
	return path, nil
}

func ValidateStruct(ctx context.Context, ymlContent []byte) (*core.TASConfig, error) {
	enObj := en.New()
	uni := ut.New(enObj, enObj)
	trans, _ := uni.GetTranslator("en")
	validate := validator.New()
	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, err
	}
	configureValidator(validate, trans)

	tasConfig := &core.TASConfig{SmartRun: true, Tier: core.Small, SplitMode: core.TestSplit}
	if err := yaml.Unmarshal(ymlContent, tasConfig); err != nil {
		return nil, errs.ErrInvalidConfFileFormat
	}

	validateErr := validate.Struct(tasConfig)
	if validateErr != nil {
		// translate all error at once
		validationErrs := validateErr.(validator.ValidationErrors)
		err := new(errs.ErrInvalidConf)
		for _, e := range validationErrs {
			// can translate each error one at a time.
			err.Fields = append(err.Fields, e.Field())
			err.Values = append(err.Values, e.Value())
		}

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
