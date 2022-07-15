package command

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/LambdaTest/test-at-scale/mocks"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/secret"
	"github.com/LambdaTest/test-at-scale/testutils"
)

func TestNewExecutionManager(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	azureClient := new(mocks.AzureClient)
	secretParser := secret.New(logger)
	type args struct {
		secretParser core.SecretParser
		azureClient  core.AzureClient
		logger       lumber.Logger
	}
	tests := []struct {
		name string
		args args
		want core.ExecutionManager
	}{
		{"Test initialisation func",
			args{secretParser: secretParser,
				azureClient: azureClient,
				logger:      logger,
			},
			&manager{
				logger:       logger,
				secretParser: secretParser,
				azureClient:  azureClient,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewExecutionManager(tt.args.secretParser, tt.args.azureClient, tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewExecutionManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_manager_GetEnvVariables(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	secretParser := secret.New(logger)
	azureClient := new(mocks.AzureClient)
	envVars := os.Environ()

	type fields struct {
		logger       lumber.Logger
		secretParser core.SecretParser
		azureClient  core.AzureClient
	}
	type args struct {
		envMap     map[string]string
		secretData map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{"Test GetEnvVariables for success",
			fields{
				logger:       logger,
				secretParser: secretParser,
				azureClient:  azureClient,
			},
			args{
				envMap:     map[string]string{"os": "linux", "arch": "amd64", "ver": "1.15"},
				secretData: map[string]string{"key1": "abc", "key2": "xyz", "key3": "123"},
			},
			append(envVars, "arch=amd64 os=linux ver=1.15"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				logger:       tt.fields.logger,
				secretParser: tt.fields.secretParser,
				azureClient:  tt.fields.azureClient,
			}
			got, err := m.GetEnvVariables(tt.args.envMap, tt.args.secretData)
			if (err != nil) != tt.wantErr {
				t.Errorf("manager.GetEnvVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Strings(got)
			sort.Strings(tt.want)
			received := fmt.Sprintf("%v", got)
			want := fmt.Sprintf("%v", tt.want)
			if len(received) != len(want) || received != want {
				t.Errorf("manager.GetEnvVariables() = \n%v, \nwant \n%v", got, tt.want)
			}
		})
	}
}
