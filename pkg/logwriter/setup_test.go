package logwriter

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/LambdaTest/test-at-scale/mocks"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/stretchr/testify/mock"
)

func Test_azure_write_logger_strategy(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
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
		azureClient core.AzureClient
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
			fields{
				azureClient: azureClientGetSASURL,
			},
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				reader:   &strings.Reader{},
			},
			errGetSASURL,
			true,
		},
		{"Test StoreCommandLogs for CreateUsingSASURL error",
			fields{
				azureClient: azureClientCreateSASURL,
			},
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				reader:   &strings.Reader{},
			},
			errCreateUsingSASURL,
			true,
		},
		{"Test StoreCommandLogs for success",
			fields{
				azureClient: azureClientSuccess,
			},
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				reader:   &strings.Reader{},
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
			m := &AzureLogWriter{
				logger:      logger,
				purpose:     core.PurposeCache,
				azureClient: tt.fields.azureClient,
			}
			got := m.Write(tt.args.ctx, tt.args.reader)
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

func mockUtil(azureClient *mocks.AzureClient, msgGet, msgCreate, errGet, errCreate string, wantErrGet, wantErrCreate bool) {
	var x map[string]interface{}
	azureClient.On("GetSASURL", mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("core.SASURLPurpose"), x).Return(
		func(ctx context.Context, purpose core.SASURLPurpose, data map[string]interface{}) string {
			return msgGet
		},
		func(ctx context.Context, purpose core.SASURLPurpose, data map[string]interface{}) error {
			if !wantErrGet {
				return nil
			}
			return errs.New(errGet)
		})

	azureClient.On("CreateUsingSASURL", mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("string"), mock.AnythingOfType("*strings.Reader"), "text/plain").Return(
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
