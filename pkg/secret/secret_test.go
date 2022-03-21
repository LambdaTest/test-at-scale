package secret

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

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
	type data struct {
		AccessToken  string    `json:"access_token"`
		Expiry       time.Time `json:"expiry"`
		RefreshToken string    `json:"refresh_token"`
	}
	time, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2022 16:22:01 IST")
	if err != nil {
		log.Fatalf("Could not parse time, error: %v", err)
	}
	Data := data{AccessToken: "token", Expiry: time, RefreshToken: "refresh"}

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
			expected := "&{{token 2022-02-22 16:22:01 +0530 IST refresh}}"
			received := fmt.Sprintf("%v", got)
			if got != nil && !(strings.HasPrefix(received, "&{{token") && strings.HasSuffix(received, "refresh}}")) {
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

func TestExpired(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}

	secretParser := New(logger)

	tests := []struct {
		token   *core.Oauth
		expired bool
	}{
		{
			expired: false,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "12345",
					RefreshToken: ""},
			},
		},
		{
			expired: false,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "12345",
					RefreshToken: "",
					Expiry:       time.Now().Add(-time.Hour)},
			},
		},
		{
			expired: false,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "12345",
					RefreshToken: "54321"},
			},
		},
		{
			expired: false,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "12345",
					RefreshToken: "54321",
					Expiry:       time.Now().Add(time.Hour)},
			},
		},
		// missing access token
		{
			expired: true,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "",
					RefreshToken: "54321"},
			},
		},
		// token expired
		{
			expired: true,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "12345",
					RefreshToken: "54321",
					Expiry:       time.Now().Add(-time.Second)},
			},
		},
		// this token is not expired, however, it is within
		// the default 15 minute expiry window.
		{
			expired: true,
			token: &core.Oauth{
				Data: core.Token{
					AccessToken:  "12345",
					RefreshToken: "54321",
					Expiry:       time.Now().Add(time.Second * 600)},
			},
		},
	}

	for i, test := range tests {
		if got, want := secretParser.Expired(test.token), test.expired; got != want {
			t.Errorf("Want token expired %v, got %v at index %d", want, got, i)
		}
	}
}
