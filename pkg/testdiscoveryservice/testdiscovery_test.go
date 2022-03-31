// Package testdiscoveryservice is used for discover tests
package testdiscoveryservice

import (
	"context"
	"reflect"
	"testing"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/requestutils"
	"github.com/LambdaTest/synapse/testutils"
	"github.com/LambdaTest/synapse/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func Test_testDiscoveryService_Discover(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	requests := requestutils.New(logger)
	tdResChan := make(chan core.DiscoveryResult)
	global.TestEnv = true
	defer func() { global.TestEnv = false }()

	var PassedEnvMap map[string]string        // envMap which should pass to call execManager.GetEnvVariables
	var PassedSecretDataMap map[string]string // secretData map which should pass to call execManager.GetEnvVariables

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

	type fields struct {
		logger      lumber.Logger
		execManager core.ExecutionManager
	}
	type args struct {
		ctx        context.Context
		tasConfig  *core.TASConfig
		payload    *core.Payload
		secretData map[string]string
		diff       map[string]int
		diffExists bool
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		wantEnvMap     map[string]string
		wantSecretData map[string]string
	}{
		{"Test Discover with Premerge pattern",
			fields{
				logger:      logger,
				execManager: execManager,
			},
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					Postmerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "RepoName"},
					},
					Premerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "repo"},
					},
				},
				payload: &core.Payload{
					EventType:   "pull-request",
					TasFileName: "../../tesutils/testdata/tas.yaml",
				},
				secretData: map[string]string{"secret": "data"},
				diff:       map[string]int{},
			},
			true,
			map[string]string{"env": "repo"},
			map[string]string{"secret": "data"},
		},
		{"Test Discover with Postmerge pattern",
			fields{
				logger:      logger,
				execManager: execManager,
			},
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					Postmerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "RepoName"},
					},
					Premerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "repo"},
					},
				},
				payload: &core.Payload{
					EventType:   "push",
					TasFileName: "../../tesutils/testdata/tas.yaml",
				},
				secretData: map[string]string{"this is": "a secret"},
				diff:       map[string]int{"../../tesutils/testdata/tas.yaml": 2},
			},
			true,
			map[string]string{"env": "RepoName"},
			map[string]string{"this is": "a secret"},
		},
		{"Test Discover not to execute discoverAll",
			fields{
				logger:      logger,
				execManager: execManager,
			},
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					Postmerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "RepoName"},
					},
					Premerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "repo"},
					},
					SmartRun: true,
				},
				payload: &core.Payload{
					EventType:                  "push",
					TasFileName:                "../../tesutils/testdata/tas.yaml",
					ParentCommitCoverageExists: true,
				},
				secretData: map[string]string{"secret": "data"},
				diff:       map[string]int{"../../tesutils/testdata/dne.yaml": 4},
			},
			true,
			map[string]string{"env": "RepoName"},
			map[string]string{"secret": "data"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tds := NewTestDiscoveryService(context.TODO(), tdResChan, tt.fields.execManager, requests, tt.fields.logger)
			err := tds.Discover(tt.args.ctx, tt.args.tasConfig, tt.args.payload, tt.args.secretData, tt.args.diff, tt.args.diffExists)

			if !reflect.DeepEqual(PassedEnvMap, tt.wantEnvMap) || !reflect.DeepEqual(PassedSecretDataMap, tt.wantSecretData) {
				t.Errorf("expected Envmap: %+v, received: %+v\nexpected SecretDataMap: %+v, received: %+v\n", tt.wantEnvMap, PassedEnvMap, tt.wantSecretData, PassedSecretDataMap)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("testDiscoveryService.Discover() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
