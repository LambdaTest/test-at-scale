package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/secret"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func TestNewExecutionManager(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
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
		t.Errorf("Couldn't initialise logger, error: %v", err)
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

func mockUtil(azureClient *mocks.AzureClient, msgGet, msgCreate, errGet, errCreate string, wantErrGet, wantErrCreate bool) {
	azureClient.On("GetSASURL", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("core.ContainerType")).Return(
		func(ctx context.Context, containerPath string, containerType core.ContainerType) string {
			return msgGet
		},
		func(ctx context.Context, containerPath string, containerType core.ContainerType) error {
			if !wantErrGet {
				return nil
			}
			return errs.New(errGet)
		})

	azureClient.On("CreateUsingSASURL", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("*mocks.Reader"), "text/plain").Return(
		func(ctx context.Context, sasURL string, reader io.Reader, mimeType string) string {
			return msgCreate
		},
		func(ctx context.Context, sasURL string, reader io.Reader, mimeType string) error {
			if !wantErrCreate {
				return nil
			}
			return errs.New(errCreate)
		})
}

func Test_manager_StoreCommandLogs(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	secretParser := new(mocks.SecretParser)
	azureClientGetSASURL := new(mocks.AzureClient)
	mockUtil(azureClientGetSASURL, "getSASURL", "createUsingSASURL", "error in GetSASURL", "error in CreateUsingSASURL", true, false)

	azureClientCreateSASURL := new(mocks.AzureClient)
	mockUtil(azureClientCreateSASURL, "getSASURL", "createUsingSASURL", "error in GetSASURL", "error in CreateUsingSASURL", false, true)

	azureClientSuccess := new(mocks.AzureClient)
	mockUtil(azureClientSuccess, "getSASURL", "createUsingSASURL", "error in GetSASURL", "error in CreateUsingSASURL", false, false)

	errGetSASURL := make(chan error, 1)
	defer func() { close(errGetSASURL) }()

	errCreateUsingSASURL := make(chan error, 1)
	defer func() { close(errCreateUsingSASURL) }()

	errSuccess := make(chan error, 1)
	defer func() { close(errSuccess) }()

	type fields struct {
		logger       lumber.Logger
		secretParser core.SecretParser
		azureClient  core.AzureClient
	}
	type args struct {
		ctx      context.Context
		blobPath string
		reader   io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    <-chan error
		wantErr bool
	}{
		{"Test StoreCommandLogs for getSASURL error",
			fields{logger: logger,
				secretParser: secretParser,
				azureClient:  azureClientGetSASURL,
			},
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				reader:   &mocks.Reader{},
			},
			errGetSASURL,
			true,
		},
		{"Test StoreCommandLogs for CreateUsingSASURL error",
			fields{logger: logger,
				secretParser: secretParser,
				azureClient:  azureClientCreateSASURL,
			},
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				reader:   &mocks.Reader{},
			},
			errCreateUsingSASURL,
			true,
		},
		{"Test StoreCommandLogs for success",
			fields{logger: logger,
				secretParser: secretParser,
				azureClient:  azureClientSuccess,
			},
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				reader:   &mocks.Reader{},
			},
			errSuccess,
			false,
		},
	}
	errGetSASURL <- errs.New("error in GetSASURL")
	errCreateUsingSASURL <- errs.New("error in CreateUsingSASURL")
	errSuccess <- errs.New("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				logger:       tt.fields.logger,
				secretParser: tt.fields.secretParser,
				azureClient:  tt.fields.azureClient,
			}
			got := m.StoreCommandLogs(tt.args.ctx, tt.args.blobPath, tt.args.reader)

			if !tt.wantErr {
				if len(got) != 0 {
					t.Errorf("Expected channel to be empty, received: %v", <-got)
				}
				return
			}

			received := <-got
			want := <-tt.want
			if received.Error() != want.Error() {
				t.Errorf("manager.StoreCommandLogs() = %+v, want %+v", received, want)
			}
		})
	}
}
