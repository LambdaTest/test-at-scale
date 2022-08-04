package tasconfigmanager

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/stretchr/testify/assert"
)

func assertTasConfigV1(got, want *core.TASConfig) error {
	if got.SmartRun != want.SmartRun {
		return fmt.Errorf("Mismatch in smart run got %t, want %t", got.SmartRun, want.SmartRun)
	}
	if got.Framework != want.Framework {
		return fmt.Errorf("Mismatch in framework , got %s , want %s", got.Framework, want.Framework)
	}
	if got.ConfigFile != want.ConfigFile {
		return fmt.Errorf("Mismatch in configFile , got %s , want %s", got.ConfigFile, want.ConfigFile)
	}
	if got.NodeVersion != want.NodeVersion {
		return fmt.Errorf("Mismatch in nodeVersion , got %s, want %s", got.NodeVersion, want.NodeVersion)
	}
	if got.Tier != want.Tier {
		return fmt.Errorf("Mismatch in tier , got %s, want %s", got.Tier, want.Tier)
	}
	if got.SplitMode != want.SplitMode {
		return fmt.Errorf("Mismatch in split mode , got %s, want %s", got.SplitMode, want.SplitMode)
	}
	if got.Version != want.Version {
		return fmt.Errorf("Mismatch in version , got %s, want %s", got.Version, want.Version)
	}
	if !reflect.DeepEqual(*got.Premerge, *want.Premerge) {
		return fmt.Errorf("Mismmatch in pre merge pattern , got %+v, want %+v", *got.Premerge, *want.Premerge)
	}
	if !reflect.DeepEqual(*got.Postmerge, *want.Postmerge) {
		return fmt.Errorf("Mismmatch in post merge pattern , got %+v, want %+v", *got.Postmerge, *want.Postmerge)
	}
	if !reflect.DeepEqual(*got.Prerun, *want.Prerun) {
		return fmt.Errorf("Mismmatch in preRun , got %+v, want %+v", *got.Prerun, *want.Prerun)
	}
	if !reflect.DeepEqual(*got.Postrun, *want.Postrun) {
		return fmt.Errorf("Mismmatch in preRun , got %+v, want %+v", *got.Postrun, *want.Postrun)
	}
	return nil
}

func assertTasConfigV2(got, want *core.TASConfigV2) error {
	if got.SmartRun != want.SmartRun {
		return fmt.Errorf("Mismatch in smart run got %t, want %t", got.SmartRun, want.SmartRun)
	}

	if got.Tier != want.Tier {
		return fmt.Errorf("Mismatch in tier , got %s, want %s", got.Tier, want.Tier)
	}
	if got.SplitMode != want.SplitMode {
		return fmt.Errorf("Mismatch in split mode , got %s, want %s", got.SplitMode, want.SplitMode)
	}
	if got.Version != want.Version {
		return fmt.Errorf("Mismatch in version , got %s, want %s", got.Version, want.Version)
	}
	if err := assertMergeV2(got.PreMerge, want.PreMerge, "preMerge"); err != nil {
		return err
	}

	return assertMergeV2(got.PostMerge, want.PostMerge, "postMerge")
}

func assertMergeV2(got, want *core.MergeV2, mode string) error {
	if !assert.ObjectsAreEqualValues(got.PreRun, want.PreRun) {
		return fmt.Errorf("Mismatch in %s preRun , got %+v, want %+v", mode, got.PreRun, want.PreRun)
	}

	if !reflect.DeepEqual(got.EnvMap, want.EnvMap) {
		return fmt.Errorf("Mismatch in %s env , got %+v, want %+v", mode, got.EnvMap, want.EnvMap)
	}
	return nil
}

func TestLoadAndValidateV1(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	tasConfigManager := NewTASConfigManager(logger)
	ctx := context.TODO()
	tests := []struct {
		Name      string
		FilePath  string
		EventType core.EventType
		Tier      core.Tier
		want      *core.TASConfig
		wantErr   error
	}{
		{
			"Invalid yml file for tas version 1",
			path.Join("../../", "testutils/testdata/tasyml/junk.yml"),
			core.EventPush,
			core.Small,
			nil,
			fmt.Errorf("`%s` configuration file contains invalid format. Please correct the `%s` file",
				"../../testutils/testdata/tasyml/junk.yml",
				"../../testutils/testdata/tasyml/junk.yml"),
		},
		{
			"Valid Config",
			path.Join("../../", "testutils/testdata/tasyml/validwithCacheKey.yml"),
			core.EventPush,
			core.Small,
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
				Cache: &core.Cache{
					Key:   "xyz",
					Paths: []string{"abcd"},
				},
			},
			nil,
		},
		{
			"PreMerge is empty in tas yml for PR",
			path.Join("../../", "testutils/testdata/tasyml/pre_merge_emptyv1.yml"),
			core.EventPullRequest,
			core.Small,
			nil,
			fmt.Errorf("`preMerge` test cases are not configured in `%s` configuration file.",
				"../../testutils/testdata/tasyml/pre_merge_emptyv1.yml"),
		},
		{
			"post merge is empty in tas yml for push event ",
			path.Join("../../", "testutils/testdata/tasyml/postmerge_emptyv1.yml"),
			core.EventPush,
			core.Small,
			nil,
			fmt.Errorf("`postMerge` test cases are not configured in `%s` configuration file.",
				"../../testutils/testdata/tasyml/postmerge_emptyv1.yml"),
		},
	}
	for _, tt := range tests {
		tas, err := tasConfigManager.LoadAndValidate(ctx, 1, tt.FilePath, tt.EventType, core.Small, tt.FilePath)
		if err != nil {
			assert.Equal(t, err.Error(), tt.wantErr.Error(), "error mismatch")
		} else {
			tasConfig := tas.(*core.TASConfig)
			err = assertTasConfigV1(tasConfig, tt.want)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
		}
	}
}

// nolint
func TestLoadAndValidateV2(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	tasConfigManager := NewTASConfigManager(logger)
	ctx := context.TODO()
	tests := []struct {
		Name      string
		FilePath  string
		EventType core.EventType
		Tier      core.Tier
		want      *core.TASConfigV2
		wantErr   error
	}{
		{
			"Invalid yml file for tas version 1",
			path.Join("../../", "testutils/testdata/tasyml/junk.yml"),
			core.EventPush,
			core.Small,
			nil,
			fmt.Errorf("`%s` configuration file contains invalid format. Please correct the `%s` file",
				"../../testutils/testdata/tasyml/junk.yml",
				"../../testutils/testdata/tasyml/junk.yml"),
		},
		{
			"PreMerge is missing in tas yml file for pull_request event",
			path.Join("../../", "testutils/testdata/tasyml/premerge_emptyv2.yaml"),
			core.EventPullRequest,
			core.Small,
			nil,
			fmt.Errorf("`preMerge` is missing in tas configuration file %s",
				"../../testutils/testdata/tasyml/premerge_emptyv2.yaml"),
		},
		{
			"PostMerge is missing in tas yml file for push event",
			path.Join("../../", "testutils/testdata/tasyml/postmerge_emptyv2.yaml"),
			core.EventPush,
			core.Small,
			nil,
			fmt.Errorf("`postMerge` is missing in tas configuration file %s",
				"../../testutils/testdata/tasyml/postmerge_emptyv2.yaml"),
		},
		{
			"Duplicate submodule name in preMerge",
			path.Join("../../", "testutils/testdata/tasyml/duplicate_submodule_premerge.yaml"),
			core.EventPullRequest,
			core.Small,
			nil,
			fmt.Errorf("duplicate subModule name found in `preMerge` in tas configuration file %s",
				"../../testutils/testdata/tasyml/duplicate_submodule_premerge.yaml"),
		},
		{
			"Duplicate submodule name in postMerge",
			path.Join("../../", "testutils/testdata/tasyml/duplicate_submodule_postmerge.yaml"),
			core.EventPush,
			core.Small,
			nil,
			fmt.Errorf("duplicate subModule name found in `postMerge` in tas configuration file %s",
				"../../testutils/testdata/tasyml/duplicate_submodule_postmerge.yaml"),
		},
		{
			"Valid Config",
			"../../testutils/testdata/tasyml/valid_with_cachekeyV2.yml",
			core.EventPush,
			core.Small,
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
				Cache: &core.Cache{
					Key:   "xyz",
					Paths: []string{"abcd"},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		tas, err := tasConfigManager.LoadAndValidate(ctx, 2, tt.FilePath, tt.EventType, core.Small, tt.FilePath)
		if err != nil {
			assert.Equal(t, err.Error(), tt.wantErr.Error(), "error mismatch")
		} else {
			tasConfig := tas.(*core.TASConfigV2)
			err = assertTasConfigV2(tasConfig, tt.want)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
		}
	}
}
