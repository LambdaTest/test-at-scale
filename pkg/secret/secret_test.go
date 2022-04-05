package secret

import (
	"errors"
	"log"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

func TestGetRepoSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("could not instantiate logger %s", err.Error())
	}
	secretParser := New(logger)

	tests := []struct {
		name      string
		path      string
		want      map[string]string
		errorType error
	}{
		{"Test for correct file", "../../testutils/testdata/secretTestData/secretfile.json", map[string]string{"abc": "val", "xyz": "val2"}, nil},
		{"Test for invalid file", "../../testutils/testdata/secretTestData/invalidsecretfile.json", map[string]string{}, errs.ErrUnMarshalJSON},
		{"Test for incorrect path", "", nil, os.ErrNotExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secretParser.GetRepoSecret(tt.path)
			if err != nil {
				if !errors.Is(err, tt.errorType) {
					t.Error(err)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected: %v, got: %v", tt.want, got)
				return
			}
		})
	}
}

func TestGetOauthSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("could not instantiate logger %s", err.Error())
	}
	secretParser := New(logger)

	oauthToken := core.Oauth{AccessToken: "token", Expiry: time.Unix(1645527121, 0), RefreshToken: "refresh", Type: core.Bearer}

	tests := []struct {
		name      string
		path      string
		want      *core.Oauth
		errorType error
	}{
		{"Test for correct file", "../../testutils/testdata/secretTestData/secretOauthFile.json", &oauthToken, nil},
		{"Test for invalid file", "../../testutils/testdata/secretTestData/invalidsecretfile.json", nil, errs.ErrMissingAccessToken},
		{"Test for incorrect path", "", nil, os.ErrNotExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := secretParser.GetOauthSecret(tt.path)
			if err != nil {
				if !errors.Is(err, tt.errorType) {
					t.Error(err)
				}
				return
			}
			if got, want := got.AccessToken, tt.want.AccessToken; got != want {
				t.Errorf("Want access_token %s, got %s", want, got)
			}
			if got, want := got.Type, tt.want.Type; got != want {
				t.Errorf("Want type %s, got %s", want, got)
			}
			if got, want := got.Expiry.Unix(), tt.want.Expiry.Unix(); got != want {
				t.Errorf("Want expiry %d, got %d", want, got)
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

//nolint:funlen
func TestExpired(t *testing.T) {
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
					AccessToken:  "54321",
					RefreshToken: "",
					Expiry:       time.Now().Add(-time.Hour)},
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
					AccessToken:  "",
					RefreshToken: "54321"},
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
					AccessToken:  "12345",
					RefreshToken: "54321"},
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
					AccessToken:  "12345",
					RefreshToken: "54321",
					Expiry:       time.Now().Add(time.Hour)},
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
					AccessToken:  "12345",
					RefreshToken: "54321",
					Expiry:       time.Now().Add(-time.Second)},
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
					AccessToken:  "12345",
					RefreshToken: "54321",
					Expiry:       time.Now().Add(time.Second * 600)},
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
