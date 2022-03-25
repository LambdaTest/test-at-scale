package secret

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

type data struct {
	AccessToken  string         `json:"access_token"`
	Expiry       time.Time      `json:"expiry"`
	RefreshToken string         `json:"refresh_token"`
	Type         core.TokenType `json:"token_type,omitempty"`
}

func Test_secretParser_GetRepoSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	secretParser := New(logger)

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "Test for correct file",
			args:    args{path: "../../testutils/testdata/secretTestData/secretfile.json"},
			want:    map[string]string{"abc": "val", "xyz": "val2"},
			wantErr: false,
		},

		{
			name:    "Test for incorrect path",
			args:    args{path: "../../testutils/testdata/secretTestData/PathNotExist/a.json"},
			want:    map[string]string{},
			wantErr: false,
		},

		{
			name:    "Test for invalid file",
			args:    args{path: "../../testutils/testdata/secretTestData/invalidsecretfile"},
			want:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secretParser.GetRepoSecret(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("secretParser.GetRepoSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Errorf("secretParser.GetRepoSecret() = %v, want %v", got, tt.want)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("secretParser.GetRepoSecret() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func Test_secretParser_GetOauthSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	secretParser := New(logger)
	parsedtime, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2022 16:22:01 IST")
	if err != nil {
		log.Fatalf("Could not parse time, error: %v", err)
	}
	Data := data{AccessToken: "token", Expiry: parsedtime, RefreshToken: "refresh", Type: core.Bearer}

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *core.Oauth
		wantErr bool
	}{
		{
			name:    "Test for correct file",
			args:    args{path: "../../testutils/testdata/secretTestData/secretOauthFile.json"},
			want:    &core.Oauth{Data: Data},
			wantErr: false,
		},

		{
			name:    "Test for incorrect path",
			args:    args{path: "../../testutils/testdata/secretTestData/PathNotExist/a.json"},
			want:    nil,
			wantErr: true,
		},

		{
			name:    "Test for invalid file",
			args:    args{path: "../../testutils/testdata/secretTestData/invalidsecretfile"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secretParser.GetOauthSecret(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("secretParser.GetOauthSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			expected := fmt.Sprintf("%v", tt.want)
			received := fmt.Sprintf("%v", got)
			if got != nil && !(strings.HasPrefix(received, "&{{token") && strings.HasSuffix(received, "Bearer}}")) {
				t.Errorf("Expected: %v, got: %v", expected, received)
				return
			}
		})
	}
}

func TestSubstituteSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}

	secretParser := New(logger)
	var expressions = []struct {
		params    map[string]string
		input     string
		output    string
		errorType error
	}{
		// basic
		{
			params:    map[string]string{"token": "secret"},
			input:     "${{ secrets.token }}",
			output:    "secret",
			errorType: nil,
		},
		// multiple
		{
			params:    map[string]string{"NPM_TOKEN": "secret", "TAG": "nucleus"},
			input:     "docker build --build-arg NPM_TOKEN=${{ secrets.NPM_TOKEN }} --tag=${{ secrets.TAG }}",
			output:    "docker build --build-arg NPM_TOKEN=secret --tag=nucleus",
			errorType: nil,
		},
		// no match
		{
			params:    map[string]string{"clone_token": "secret"},
			input:     "${{ secrets.token }}",
			output:    "${{ secrets.token }}",
			errorType: nil,
		},
	}

	for _, expr := range expressions {
		t.Run(expr.input, func(t *testing.T) {
			t.Logf(expr.input)
			output, err := secretParser.SubstituteSecret(expr.input, expr.params)
			if err != nil {
				if expr.errorType != nil {
					if err.Error() != expr.errorType.Error() {
						t.Errorf("Want error %q expanded but got error %q", expr.errorType, err)
						return
					}
					return
				}
				t.Errorf("Want %q expanded but got error %q", expr.input, err)
				return
			}

			if output != expr.output {
				t.Errorf("Want %q expanded to %q, got %q",
					expr.input,
					expr.output,
					output)
			}
		})
	}
}

func Test_secretParser_Expired(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	secretRegex := regexp.MustCompile(global.SecretRegex)
	s := &secretParser{
		logger:      logger,
		secretRegex: secretRegex,
	}

	type args struct {
		token *core.Oauth
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Missing Refresh Token",
			args: args{
				token: &core.Oauth{
					Data: data{
						AccessToken:  "54321",
						RefreshToken: "",
						Expiry:       time.Now().Add(-time.Hour)},
				},
			},
			want: false,
		},
		{
			name: "Missing Access Token",
			args: args{
				token: &core.Oauth{
					Data: data{
						AccessToken:  "",
						RefreshToken: "54321"},
				},
			},
			want: true,
		},
		{
			name: "Missing Time",
			args: args{
				token: &core.Oauth{
					Data: data{
						AccessToken:  "12345",
						RefreshToken: "54321"},
				},
			},
			want: false,
		},
		{
			name: "Token Valid",
			args: args{
				token: &core.Oauth{
					Data: data{
						AccessToken:  "12345",
						RefreshToken: "54321",
						Expiry:       time.Now().Add(time.Hour)},
				},
			},
			want: false,
		},
		{
			name: "Token Expire",
			args: args{
				token: &core.Oauth{
					Data: data{
						AccessToken:  "12345",
						RefreshToken: "54321",
						Expiry:       time.Now().Add(-time.Second)},
				},
			},
			want: true,
		},
		{
			name: "Token not Expiredn but in expiry buffer",
			args: args{
				token: &core.Oauth{
					Data: data{
						AccessToken:  "12345",
						RefreshToken: "54321",
						Expiry:       time.Now().Add(time.Second * 600)},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Expired(tt.args.token); got != tt.want {
				t.Errorf("secretParser.Expired() = %v, want %v", got, tt.want)
			}
		})
	}
}
