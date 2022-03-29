package command

import (
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func Test_manager_createScript(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	var azureClient core.AzureClient
	commands := []string{"cmd1", "cmd2", "cmd3"}
	secretData := map[string]string{"secret1": "s1", "secret2": "s2", "secret3": "s3"}

	secretParser := new(mocks.SecretParser)    // secretParser is a mock interface which on calling SubstituteSecret returns a string and nil error
	secretParserErr := new(mocks.SecretParser) // secretParserErr is a mock interface which on calling SubstituteSecret returns an empty string and a dummy error
	want := `

set -e


echo + "cmd1"
fakecommand

echo + "cmd2"
fakecommand

echo + "cmd3"
fakecommand
`
	type fields struct {
		logger       lumber.Logger
		secretParser core.SecretParser
		azureClient  core.AzureClient
	}
	type args struct {
		commands   []string
		secretData map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{"Test for success", fields{logger: logger, secretParser: secretParser, azureClient: azureClient}, args{commands: commands, secretData: secretData}, want, false},
		{"This should throw an error", fields{logger: logger, secretParser: secretParserErr, azureClient: azureClient}, args{commands: commands, secretData: secretData}, "", true},
	}

	secretParser.On("SubstituteSecret", mock.AnythingOfType("string"), secretData).Return(
		func(command string, secretData map[string]string) string {
			return "fakecommand"
		},
		func(command string, secretData map[string]string) error {
			return nil
		})

	secretParserErr.On("SubstituteSecret", mock.AnythingOfType("string"), secretData).Return(
		func(command string, secretData map[string]string) string {
			return ""
		},
		func(command string, secretData map[string]string) error {
			return errs.New("error from mocked interface")
		})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				logger:       tt.fields.logger,
				secretParser: tt.fields.secretParser,
				azureClient:  tt.fields.azureClient,
			}
			got, err := m.createScript(tt.args.commands, tt.args.secretData)
			if (err != nil) != tt.wantErr {
				t.Errorf("manager.createScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("manager.createScript() = %v, want %v", got, tt.want)
			}
		})
	}
}
