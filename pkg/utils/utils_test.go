package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/stretchr/testify/assert"
)

const (
	directory = "../../testutils/testdirectory"
)

func TestMin(t *testing.T) {
	type args struct {
		x int
		y int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"x: 5, y: -1", args{5, -1}, -1},
		{"x: 0, y: 0", args{0, 0}, 0},
		{"x: -293836, y: 0", args{-293836, 0}, -293836},
		{"x: 2545, y: 374", args{2545, 374}, 374},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Min(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("Min() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	_, err := os.Create("dummy_file")
	if err != nil {
		fmt.Printf("Error in creating file, error: %v", err)
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"dummy_file_name", args{"dummy_file_name"}, "", true},
		{"dummy_file", args{"dummy_file"}, "d41d8cd98f00b204e9800998ecf8427e", false},
	}
	defer removeCreatedFile("dummy_file")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeChecksum(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ComputeChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDirectory(t *testing.T) {
	newDir := "../../testutils/nonExistingDir"
	existDir := directory
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Existing directory: ../../testutils/testdirecotry", args{existDir}, false},
		{"Non-existing directory: ../../testutils/nonExistingDir", args{newDir}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateDirectory(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("CreateDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.args.path == newDir {
				if _, err := os.Lstat(newDir); err != nil {
					t.Errorf("Directory did not exist, error: %v", err)
					return
				}
				defer removeCreatedFile(newDir)
			}
		})
	}
}

func TestWriteFileToDirectory(t *testing.T) {
	path := directory
	filename := "writeFileToDirectory"
	data := []byte("Hello world!")
	err := WriteFileToDirectory(path, filename, data)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	defer removeCreatedFile(filepath.Join(path, filename))
	checkData, err := os.ReadFile(filepath.Join(path, filename))
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	if string(checkData) != "Hello world!" {
		t.Errorf("expected file contents: Hello world!, got: %s", string(checkData))
	}
}

func TestGetOutboundIP(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test1", "http://synapse:8000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutboundIP(); got != tt.want {
				t.Errorf("GetOutboundIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateStructv1(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		name     string
		filename string
		wantErr  error
		want     *core.TASConfig
	}{
		{
			"Junk characters File",
			"testutils/testdata/tasyml/junk.yml",
			// nolint:lll
			fmt.Errorf("`testutils/testdata/tasyml/junk.yml` configuration file contains invalid format. Please correct the `testutils/testdata/tasyml/junk.yml` file"),
			nil,
		},
		{
			"Invalid Types",
			"testutils/testdata/tasyml/invalid_types.yml",
			// nolint:lll
			fmt.Errorf("`testutils/testdata/tasyml/invalid_types.yml` configuration file contains invalid format. Please correct the `testutils/testdata/tasyml/invalid_types.yml` file"),
			nil,
		},
		{
			"Invalid Field Values",
			"testutils/testdata/tasyml/invalid_fields.yml",
			errs.ErrInvalidConf{
				// nolint:lll
				Message: "Invalid values provided for the following fields in the `testutils/testdata/tasyml/invalid_fields.yml` configuration file: \n",
				Fields:  []string{"framework", "nodeVersion"},
				Values:  []interface{}{"hello", "test"}},
			nil,
		},
		{
			"Valid Config",
			"testutils/testdata/tasyml/valid.yml",
			nil,
			&core.TASConfig{
				SmartRun:  true,
				Framework: "jest",
				Postmerge: &core.Merge{
					EnvMap:   map[string]string{"NODE_ENV": "development"},
					Patterns: []string{"{packages,scripts}/**/__tests__/*{.js,.coffee,[!d].ts}"},
				},
				Premerge: &core.Merge{
					EnvMap:   map[string]string{"NODE_ENV": "development"},
					Patterns: []string{"{packages,scripts}/**/__tests__/*{.js,.coffee,[!d].ts}"},
				},
				Prerun:      &core.Run{EnvMap: map[string]string{"NODE_ENV": "development"}, Commands: []string{"yarn"}},
				Postrun:     &core.Run{Commands: []string{"node --version"}},
				ConfigFile:  "scripts/jest/config.source-www.js",
				NodeVersion: "14.17.6",
				Tier:        "small",
				SplitMode:   core.TestSplit,
				Version:     "1.0",
			},
		},
		{
			"Valid Config - Only Framework",
			"testutils/testdata/tasyml/framework_only_required.yml",
			nil,
			&core.TASConfig{
				SmartRun:  true,
				Framework: "mocha",
				Tier:      "small",
				SplitMode: core.TestSplit,
				Version:   "1.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ymlContent, err := testutils.LoadFile(tt.filename)
			if err != nil {
				t.Errorf("Error loading testfile %s", tt.filename)
				return
			}
			tasConfig, errV := ValidateStructTASYmlV1(ctx, ymlContent, tt.filename)
			if errV != nil {
				assert.Equal(t, errV.Error(), tt.wantErr.Error(), "Error mismatch")
				return
			}
			assert.Equal(t, tt.want, tasConfig, "Struct mismatch")
		})
	}
}

func removeCreatedFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}
func TestValidateStructv2(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		name     string
		filename string
		wantErr  error
		want     *core.TASConfigV2
	}{
		{
			"Junk characters File",
			"testutils/testdata/tasyml/junk.yml",
			// nolint:lll
			fmt.Errorf("`testutils/testdata/tasyml/junk.yml` configuration file contains invalid format. Please correct the `testutils/testdata/tasyml/junk.yml` file"),
			nil,
		},
		{
			"Invalid Types",
			"testutils/testdata/tasyml/invalid_typesv2.yml",
			// nolint:lll
			fmt.Errorf("`testutils/testdata/tasyml/invalid_typesv2.yml` configuration file contains invalid format. Please correct the `testutils/testdata/tasyml/invalid_typesv2.yml` file"),
			nil,
		},

		{
			"Valid Config",
			"testutils/testdata/tasyml/validV2.yml",
			nil,
			&core.TASConfigV2{
				SmartRun:  true,
				Tier:      "small",
				SplitMode: core.TestSplit,
				PostMerge: &core.MergeV2{
					SubModules: []core.SubModule{
						{
							Name: "some-module-1",
							Path: "./somepath",
							Patterns: []string{
								"./x/y/z",
							},
							Framework:  "mocha",
							ConfigFile: "x/y/z",
						},
					},
				},
				PreMerge: &core.MergeV2{
					SubModules: []core.SubModule{
						{
							Name: "some-module-1",
							Path: "./somepath",
							Patterns: []string{
								"./x/y/z",
							},
							Framework:  "jasmine",
							ConfigFile: "/x/y/z",
						},
					},
				},
				Parallelism: 1,
				Version:     "2.0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ymlContent, err := testutils.LoadFile(tt.filename)
			if err != nil {
				t.Errorf("Error loading testfile %s", tt.filename)
				return
			}
			tasConfig, errV := ValidateStructTASYmlV2(ctx, ymlContent, tt.filename)
			if errV != nil {
				assert.Equal(t, errV.Error(), tt.wantErr.Error(), "Error mismatch")
				return
			}

			assert.Equal(t, tt.want, tasConfig, "Struct mismatch")
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  error
		want     int
	}{
		{
			"Test with invalid version type",
			"testutils/testdata/tasyml/invalidVersion.yml",
			fmt.Errorf("strconv.Atoi: parsing \"a\": invalid syntax"),
			0,
		},
		{
			"Test valid yml type for tas version 1",
			"testutils/testdata/tasyml/valid.yml",
			nil,
			1,
		},
		{
			"Test valid yml type for tas version 2",
			"testutils/testdata/tasyml/validV2.yml",
			nil,
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ymlContent, err := testutils.LoadFile(tt.filename)
			if err != nil {
				t.Errorf("Error loading testfile %s", tt.filename)
				return
			}
			version, errV := GetVersion(ymlContent)
			if errV != nil {
				assert.Equal(t, errV.Error(), tt.wantErr.Error(), "Error mismatch")
				return
			}
			assert.Equal(t, tt.want, version, "value mismatch")
		})
	}
}

func TestValidateSubModule(t *testing.T) {
	tests := []struct {
		name      string
		subModule core.SubModule
		wantErr   error
	}{
		{
			"Test submodule if name is empty",
			core.SubModule{
				Path:     "/x/y",
				Patterns: []string{"/a/c"},
			},

			errs.New("module name is not defined"),
		},
		{
			"Test submodule if path is empty",
			core.SubModule{
				Name:     "some name",
				Patterns: []string{"/a/c"},
			},

			errs.New("module path is not defined for module some name "),
		},
		{
			"Test submodule if pattern length is empty",
			core.SubModule{
				Name: "some-name",
				Path: "/x/y",
			},

			errs.New("module some-name pattern length is 0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateSubModule(&tt.subModule)
			assert.Equal(t, tt.wantErr, gotErr, "Error mismatch")
		})
	}
}

func Test_ShuffleLocators(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	locatorArrValue := []core.LocatorConfig{{
		Locator: "Locator_A"},
		{
			Locator: "Locator_B"},
		{
			Locator: "Locator_C"}}

	type args struct {
		locatorArr      []core.LocatorConfig
		locatorFilePath string
	}

	tests := []struct {
		name string
		args args
	}{
		{"Test_shuffleLocators",
			args{locatorArrValue, "/tmp/locators"}}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ShuffleLocators(tt.args.locatorArr, tt.args.locatorFilePath, logger); err != nil {
				t.Errorf("shuffleLocators() throws error %v", err)
			}

			VerifyLocators(tt.args.locatorFilePath, t)
		})
	}
}

func Test_ExtractLocators(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	locatorArrValue := []core.LocatorConfig{{
		Locator: "Locator_A"},
		{
			Locator: "Locator_B"},
		{
			Locator: "Locator_C"}}
	type args struct {
		locatorFilePath string
		flakyTestAlgo   string
		logger          lumber.Logger
	}
	tests := []struct {
		name string
		args args
		want []core.LocatorConfig
	}{
		{"Test_extractLocators",
			args{"/tmp/locators", core.RunningXTimesShuffle, logger},
			locatorArrValue}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload core.InputLocatorConfig
			payload.Locators = locatorArrValue
			file, _ := json.Marshal(payload)
			_ = os.WriteFile(tt.args.locatorFilePath, file, global.FilePermissionWrite)
			if err != nil {
				t.Errorf("In test_extractLocators error in writing to file = %v", err)
				return
			}
			locatorArr, err := ExtractLocators(tt.args.locatorFilePath, tt.args.flakyTestAlgo, tt.args.logger)
			if err != nil {
				t.Errorf("extractLocators() throws error %v", err)
			}

			if !reflect.DeepEqual(locatorArrValue, locatorArr) {
				t.Errorf("extractLocators(), array got %s, want %s", locatorArr, locatorArrValue)
			}
		})
	}
}

func Test_UpdateLocatorBasedOnAlgo(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	locatorArrValue := []core.LocatorConfig{{
		Locator: "Locator_A"},
		{
			Locator: "Locator_B"},
		{
			Locator: "Locator_C"}}

	type args struct {
		locatorArr      []core.LocatorConfig
		locatorFilePath string
		flakyAlgo       string
	}

	tests := []struct {
		name string
		args args
	}{
		{"Test_shuffleLocators",
			args{locatorArrValue, "/tmp/locators", core.RunningXTimesShuffle}}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateLocatorBasedOnAlgo(tt.args.flakyAlgo, tt.args.locatorFilePath, tt.args.locatorArr, logger); err != nil {
				t.Errorf("shuffleLocators() throws error %v", err)
			}

			VerifyLocators(tt.args.locatorFilePath, t)
		})
	}
}

func VerifyLocators(locatorFilePath string, t *testing.T) {
	content, err := os.ReadFile(locatorFilePath)
	if err != nil {
		t.Errorf("In test_shuffleLocators error in opening file = %v", err)
		return
	}
	// Now let's unmarshall the data into `payload`
	var payload core.InputLocatorConfig
	err = json.Unmarshal(content, &payload)
	if err != nil {
		t.Errorf("Error in unmarshlling = %v", err)
		return
	}
	if payload.Locators[0].Locator == "Locator_A" &&
		payload.Locators[1].Locator == "Locator_B" &&
		payload.Locators[2].Locator == "Locator_C" {
		t.Errorf("Shuffling could not be done, order is same as original")
	}
}
