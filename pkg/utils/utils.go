package utils

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const (
	namespaceSeparator = "."
	emptyTagName       = "-"
	yamlTagName        = "yaml"
	requiredTagName    = "required"
	v1                 = 1
	v2                 = 2
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
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, global.DirectoryPermissions); err != nil {
			return errs.ERR_DIR_CRT(err.Error())
		}
	}
	return nil
}

// DeleteDirectory deletes directory and all its children
func DeleteDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return errs.ErrDirDel(err.Error())
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

func ValidateStructTASYmlV1(ctx context.Context, ymlContent []byte, ymlFilename string) (*core.TASConfig, error) {
	validate, err := getValidator()
	if err != nil {
		return nil, err
	}
	tasConfig := &core.TASConfig{SmartRun: true, Tier: core.Small, SplitMode: core.TestSplit, Version: global.DefaultTASVersion}
	if err := yaml.Unmarshal(ymlContent, tasConfig); err != nil {
		return nil, fmt.Errorf("`%s` configuration file contains invalid format. Please correct the `%s` file", ymlFilename, ymlFilename)
	}
	if err := validateStruct(validate, tasConfig, ymlFilename); err != nil {
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

// GetVersion returns version of tas yml file
func GetVersion(ymlContent []byte) (int, error) {
	tasVersion := &core.TasVersion{Version: global.DefaultTASVersion}
	if err := yaml.Unmarshal(ymlContent, tasVersion); err != nil {
		return 0, fmt.Errorf("error in unmarshling tas yml file")
	}
	majorVersion := strings.Split(tasVersion.Version, ".")[0]

	version, err := strconv.Atoi(majorVersion)
	if err != nil {
		return version, errs.New("error while parsing version for tas yml")
	}
	return version, err
}

// ValidateStructTASYmlV2 validates tas configuration file
func ValidateStructTASYmlV2(ctx context.Context, ymlContent []byte, ymlFileName string) (*core.TASConfigV2, error) {
	tasConfig := &core.TASConfigV2{SmartRun: true, Tier: core.Small, SplitMode: core.TestSplit}
	if err := yaml.Unmarshal(ymlContent, tasConfig); err != nil {
		return nil, fmt.Errorf("`%s` configuration file contains invalid format. Please correct the `%s` file", ymlFileName, ymlFileName)
	}
	validate, err := getValidator()
	if err != nil {
		return nil, err
	}
	if err := validateStruct(validate, tasConfig, ymlFileName); err != nil {
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

func validateStruct(validate *validator.Validate, config interface{}, ymlFilename string) error {
	validateErr := validate.Struct(config)
	if validateErr != nil {
		// translate all error at once
		validationErrs := validateErr.(validator.ValidationErrors)
		err := new(errs.ErrInvalidConf)
		err.Message = errs.New(
			fmt.Sprintf(
				"Invalid values provided for the following fields in the `%s` configuration file: \n",
				ymlFilename),
		).Error()
		for _, e := range validationErrs {
			// can translate each error one at a time.
			err.Fields = append(err.Fields, e.Field())
			err.Values = append(err.Values, e.Value())
		}
		return err
	}
	return nil
}

// ValidateSubModule validates submodule
func ValidateSubModule(module *core.SubModule) error {
	if module.Name == "" {
		return errs.New("module name is not defined")
	}
	if module.Path == "" {
		return errs.New(fmt.Sprintf("module path is not defined for module %s ", module.Name))
	}
	if len(module.Patterns) == 0 {
		return errs.New(fmt.Sprintf("module %s pattern length is 0", module.Name))
	}

	return nil
}

// GetDefaultQueryAndHeaders returns the query and headers that should be supplied with each request made to TAS Server
func GetDefaultQueryAndHeaders() (query map[string]interface{}, headers map[string]string) {
	query = map[string]interface{}{
		"repoID":  os.Getenv("REPO_ID"),
		"buildID": os.Getenv("BUILD_ID"),
		"orgID":   os.Getenv("ORG_ID"),
		"taskID":  os.Getenv("TASK_ID"),
	}
	headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", os.Getenv("TOKEN")),
	}
	return query, headers
}

func GetArgs(command string, frameWork string, frameworkVersion int,
	configFile string,
	target []string) []string {
	language := global.FrameworkLanguageMap[frameWork]

	args := []string{}
	if language == "java" {
		args = append(args, "-jar", "/test-at-scale-java.jar",
			global.ArgCommand, command, global.ArgFrameworVersion,
			strconv.Itoa(frameworkVersion))
	} else {
		args = append(args, global.ArgCommand, command)
	}

	if configFile != "" {
		args = append(args, global.ArgConfig, configFile)
	}

	for _, pattern := range target {
		args = append(args, global.ArgPattern, pattern)
	}

	return args
}

// GetTASFilePath returns tas file path
func GetTASFilePath(path string) (string, error) {
	path, err := GetConfigFileName(path)
	if err != nil {
		return "", err
	}
	filePath := fmt.Sprintf("%s/%s", global.RepoDir, path)
	return filePath, nil
}

// GenerateUUID generates uuid v4
func GenerateUUID() string {
	uuidV4 := uuid.New() // panics on error
	return strings.Map(func(r rune) rune {
		if r == '-' {
			return -1
		}
		return r
	}, uuidV4.String())
}

// ValidateStructTASYml validates the TAS config for all supported version
func ValidateStructTASYml(ctx context.Context, ymlContent []byte, ymlFilename string) (interface{}, error) {
	version, err := GetVersion(ymlContent)
	if err != nil {
		return nil, err
	}
	switch version {
	case v1:
		return ValidateStructTASYmlV1(ctx, ymlContent, ymlFilename)
	case v2:
		return ValidateStructTASYmlV2(ctx, ymlContent, ymlFilename)
	default:
		return nil, fmt.Errorf("version %d is not supported ", version)
	}
}
