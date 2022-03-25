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

//nolint unused
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
		{"Test for correct file", args{path: "../../testutils/testdata/secretTestData/secretfile.json"}, map[string]string{"abc": "val", "xyz": "val2"}, false},

		{"Test for incorrect path", args{path: "../../testutils/testdata/secretTestData/PathNotExist/a.json"}, map[string]string{}, false},

		{"Test for invalid file", args{path: "../../testutils/testdata/secretTestData/invalidsecretfile"}, map[string]string{}, true},
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
	time, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2022 16:22:01 IST")
	if err != nil {
		log.Fatalf("Could not parse time, error: %v", err)
	}
	Data := data{AccessToken: "token", Expiry: time, RefreshToken: "refresh", Type: core.Bearer}

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *core.Oauth
		wantErr bool
	}{
		{"Test for correct file", args{path: "../../testutils/testdata/secretTestData/secretOauthFile.json"}, &core.Oauth{Data: Data}, false},

		{"Test for incorrect path", args{path: "../../testutils/testdata/secretTestData/PathNotExist/a.json"}, nil, true},

		{"Test for invalid file", args{path: "../../testutils/testdata/secretTestData/invalidsecretfile"}, nil, true},
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

	type fields struct {
		logger      lumber.Logger
		secretRegex *regexp.Regexp
	}
	type args struct {
		token *core.Oauth
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Missing Refresh Token",
			fields: fields{
				logger:      logger,
				secretRegex: regexp.MustCompile(global.SecretRegex),
			},
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
			fields: fields{
				logger:      logger,
				secretRegex: regexp.MustCompile(global.SecretRegex),
			},
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
			fields: fields{
				logger:      logger,
				secretRegex: regexp.MustCompile(global.SecretRegex),
			},
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
			fields: fields{
				logger:      logger,
				secretRegex: regexp.MustCompile(global.SecretRegex),
			},
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
			fields: fields{
				logger:      logger,
				secretRegex: regexp.MustCompile(global.SecretRegex),
			},
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
			fields: fields{
				logger:      logger,
				secretRegex: regexp.MustCompile(global.SecretRegex),
			},
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
			s := &secretParser{
				logger:      tt.fields.logger,
				secretRegex: tt.fields.secretRegex,
			}
			if got := s.Expired(tt.args.token); got != tt.want {
				t.Errorf("secretParser.Expired() = %v, want %v", got, tt.want)
			}
		})
	}
}
