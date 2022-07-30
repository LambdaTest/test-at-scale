// Package testdiscoveryservice is used for discover tests
package testdiscoveryservice

import (
	"context"
	"reflect"
	"testing"

	"github.com/LambdaTest/test-at-scale/mocks"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/mock"
)

type argsV1 struct {
	ctx           context.Context
	discoveryArgs core.DiscoveyArgs
}

type testV1 struct {
	name           string
	args           argsV1
	wantErr        bool
	wantEnvMap     map[string]string
	wantSecretData map[string]string
}

func Test_testDiscoveryService_Discover(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	requests := requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{})
	tdResChan := make(chan core.DiscoveryResult)
	global.TestEnv = true
	defer func() { global.TestEnv = false }()

	var PassedEnvMap map[string]string        // envMap which should be passed to call execManager.GetEnvVariables
	var PassedSecretDataMap map[string]string // secretData map which should be passed to call execManager.GetEnvVariables

	execManager := new(mocks.ExecutionManager)
	execManager.On("GetEnvVariables", mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(envMap, secretData map[string]string) []string {
			PassedEnvMap = envMap
			PassedSecretDataMap = secretData
			return []string{"success", "ss"}
		},
		func(envMap, secretData map[string]string) error {
			PassedEnvMap = envMap
			PassedSecretDataMap = secretData
			return nil
		},
	)
	tds := NewTestDiscoveryService(context.TODO(), tdResChan, execManager, requests, logger)
	tests := getTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tds.Discover(tt.args.ctx, &tt.args.discoveryArgs)
			if !reflect.DeepEqual(PassedEnvMap, tt.wantEnvMap) || !reflect.DeepEqual(PassedSecretDataMap, tt.wantSecretData) {
				t.Errorf("expected Envmap: %+v, received: %+v\nexpected SecretDataMap: %+v, received: %+v\n",
					tt.wantEnvMap, PassedEnvMap, tt.wantSecretData, PassedSecretDataMap)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getTestCases() []*testV1 {
	testCases := []*testV1{
		{"Test Discover with Premerge pattern",

			argsV1{
				ctx: context.TODO(),
				discoveryArgs: core.DiscoveyArgs{
					TestPattern: []string{"./test/**/*.spec.ts"},
					Payload: &core.Payload{
						EventType:   core.EventPullRequest,
						TasFileName: "../../tesutils/testdata/tas.yaml",
					},
					EnvMap:         map[string]string{"env": "repo"},
					SecretData:     map[string]string{"secret": "data"},
					TestConfigFile: "",
					FrameWork:      "jest",
					SmartRun:       false,
					Diff:           map[string]int{},
					DiffExists:     true,
				},
			},
			true,
			map[string]string{"env": "repo"},
			map[string]string{"secret": "data"},
		},
		{"Test Discover with Postmerge pattern",
			argsV1{
				ctx: context.TODO(),
				discoveryArgs: core.DiscoveyArgs{
					TestPattern: []string{"./test/**/*.spec.ts"},
					EnvMap:      map[string]string{"env": "RepoName"},
					Payload: &core.Payload{
						EventType:   "push",
						TasFileName: "../../tesutils/testdata/tas.yaml",
					},
					SecretData:     map[string]string{"this is": "a secret"},
					FrameWork:      "mocha",
					TestConfigFile: "",
					SmartRun:       false,
					Diff:           map[string]int{},
					DiffExists:     false,
				},
			},
			true,
			map[string]string{"env": "RepoName"},
			map[string]string{"this is": "a secret"},
		},
		{"Test Discover not to execute discoverAll",
			argsV1{
				ctx: context.TODO(),
				discoveryArgs: core.DiscoveyArgs{
					TestPattern: []string{"./test/**/*.spec.ts"},
					EnvMap:      map[string]string{"env": "RepoName"},
					Payload: &core.Payload{
						EventType:                  "push",
						TasFileName:                "../../tesutils/testdata/tas.yaml",
						ParentCommitCoverageExists: true,
					},
					SecretData: map[string]string{"secret": "data"},
					FrameWork:  "jasmine",
				},
			},
			true,
			map[string]string{"env": "RepoName"},
			map[string]string{"secret": "data"},
		},
	}
	return testCases
}
