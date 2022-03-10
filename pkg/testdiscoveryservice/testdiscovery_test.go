// Package testdiscoveryservice is used for discover tests
package testdiscoveryservice

import (
	"context"
	"testing"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/testutils"
	"github.com/LambdaTest/synapse/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func Test_testDiscoveryService_Discover(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	global.TestEnv = true
	defer func() { global.TestEnv = false }()
	execManager := new(mocks.ExecutionManager)
	execManager.On("GetEnvVariables", mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(envMap, secretData map[string]string) []string {
			return []string{"success", "ss"}
		},
		func(envMap, secretData map[string]string) error {
			return nil
		})

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
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
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
				secretData: map[string]string{},
				diff:       map[string]int{},
			},
			true, // global.RepoDir does not exist on local, TODO: check by running test in docker container
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
				secretData: map[string]string{},
				diff:       map[string]int{"../../tesutils/testdata/tas.yaml": 2},
			},
			true, // global.RepoDir does not exist on local, TODO: check by running test in docker container
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
				secretData: map[string]string{},
				diff:       map[string]int{"../../tesutils/testdata/dne.yaml": 4},
			},
			true, // global.RepoDir does not exist on local, TODO: check by running test in docker container
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tds := NewTestDiscoveryService(tt.fields.execManager, tt.fields.logger)
			if err := tds.Discover(tt.args.ctx, tt.args.tasConfig, tt.args.payload, tt.args.secretData, tt.args.diff); (err != nil) != tt.wantErr {
				t.Errorf("testDiscoveryService.Discover() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
