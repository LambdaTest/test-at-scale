package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/testutils"
	"github.com/LambdaTest/synapse/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func getParserPayload() core.ParserResponse {
	return core.ParserResponse{
		BuildID:     "buildID",
		RepoID:      "gittest.com/user/testRepoID",
		OrgID:       "orgID",
		GitProvider: "gittest",
		RepoSlug:    "user/testRepoID",
		Status: &core.ParserStatus{
			TargetCommitID: "targetCommitID",
			BaseCommitID:   "baseCommitID",
			Status:         "status",
			Message:        "msg",
			Tier:           "tier",
		},
	}
}

func getParserService(logger lumber.Logger, tasCfgManager *mocks.TASConfigManager, endpoint string) *parserService {
	return &parserService{
		logger:           logger,
		tasConfigManager: tasCfgManager,
		httpClient: http.Client{
			Timeout: global.DefaultHTTPTimeout,
		},
		endpoint: endpoint,
	}
}

func getLoggerAndPayload() (lumber.Logger, core.Payload) {
	logger, err := testutils.GetLogger()
	if err != nil {
		fmt.Printf("Couldn't get logger, error: %v", err)
	}
	payload, err := testutils.GetPayload()
	if err != nil {
		fmt.Printf("Couldn't get payload, error: %v", err)
	}
	payload.LicenseTier = "small"
	// tasConfigManager := new(mocks.TASConfigManager)

	return logger, *payload
}

func mockUtil(tasCfgManager *mocks.TASConfigManager, wantErr bool, tier core.Tier) {
	tasCfgManager.On("LoadConfig", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("core.EventType"), mock.AnythingOfType("bool")).Return(
		func(ctx context.Context, path string, eventType core.EventType, parseMode bool) *core.TASConfig {
			return &core.TASConfig{
				Tier: tier,
			}
		},
		func(ctx context.Context, path string, eventType core.EventType, parseMode bool) error {
			if wantErr {
				return errs.New("Failed to load config")
			}
			return nil
		},
	)
}

func Test_parserService_ParseAndValidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/perform-parsing" {
			t.Errorf("Expected to request '/perform-parsing', got: %v", r.URL)

			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger, payload := getLoggerAndPayload()
	type args struct {
		ctx     context.Context
		payload *core.Payload
	}
	tests := []struct {
		name        string
		args        args
		endpoint    string
		tier        core.Tier
		wantMockErr bool
		wantErr     bool
	}{
		{"Test ParseAndValidate for LoadConfig error", args{ctx: context.TODO(), payload: &payload}, server.URL + "/perform-parsing", "small", true, false},

		{"Test ParseAndValidate for success", args{ctx: context.TODO(), payload: &payload}, server.URL + "/perform-parsing", "small", false, false},

		{"Test ParseAndValidate for invalid license", args{ctx: context.TODO(), payload: &payload}, server.URL + "/perform-parsing", "large", false, false},
	}
	for _, tt := range tests {
		tasCfgManager := new(mocks.TASConfigManager)
		mockUtil(tasCfgManager, tt.wantMockErr, tt.tier)
		p := getParserService(logger, tasCfgManager, tt.endpoint)
		t.Run(tt.name, func(t *testing.T) {
			if err := p.ParseAndValidate(tt.args.ctx, tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("parserService.ParseAndValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parserService_sendParserResponse(t *testing.T) {
	logger, _ := getLoggerAndPayload()
	parserResp := getParserPayload()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parse" {
			t.Errorf("Expected to request '/parse', got: %v", r.URL)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		expBody, _ := json.Marshal(parserResp)
		if err != nil {
			http.Error(w, "can't read body", http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		} else if string(expBody) != string(body) {
			http.Error(w, "expected request body did not match with received", http.StatusBadRequest)
			fmt.Printf("expected: %v\ngot: %v\n", string(expBody), string(body))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverToRespNotFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parse" {
			t.Errorf("Expected to request '/parse', got: %v", r.URL)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(`{"value":"fixed"}`))
		if err != nil {
			fmt.Printf("Could not write data in httptest server, error: %v", err)
		}
	}))
	defer serverToRespNotFound.Close()

	type args struct {
		payload *core.ParserResponse
	}
	tests := []struct {
		name     string
		args     args
		endpoint string
		wantErr  bool
	}{
		{"Test with dummy data, responding http.StatusOK", args{payload: &parserResp}, server.URL + "/parse", false},

		{"Test with dummy data, responding http.StatusNotFound", args{payload: &parserResp}, serverToRespNotFound.URL + "/parse", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasconfigmanager := new(mocks.TASConfigManager)
			p := getParserService(logger, tasconfigmanager, tt.endpoint)
			if err := p.sendParserResponse(tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("parserService.sendParserResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isValidLicenseTier(t *testing.T) {
	var yl1, yl2, yl3, yl4, yl5, cl1, cl2, cl3, cl4, cl5 core.Tier
	yl1 = "xsmall"
	yl2 = "small"
	yl3 = "medium"
	yl4 = "large"
	yl5 = "xlarge"
	cl1 = "xsmall"
	cl2 = "small"
	cl3 = "medium"
	cl4 = "large"
	cl5 = "xlarge"
	type args struct {
		yamlLicense    core.Tier
		currentLicense core.Tier
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"Test: yamlLicense: xsmall, currentLicense: xsmall", args{yl1, cl1}, true, false},
		{"Test: yamlLicense: xsmall, currentLicense: small", args{yl1, cl2}, true, false},
		{"Test: yamlLicense: small, currentLicense: medium", args{yl2, cl3}, true, false},
		{"Test: yamlLicense: medium, currentLicense: medium", args{yl3, cl3}, true, false},
		{"Test: yamlLicense: large, currentLicense: large", args{yl4, cl4}, true, false},
		{"Test: yamlLicense: xlarge, currentLicense: xlarge", args{yl5, cl5}, true, false},
		{"Test: yamlLicense: xlarge, currentLicense: xsmall", args{yl5, cl1}, false, true},
		{"Test: yamlLicense: medium, currentLicense: xsmall", args{yl3, cl1}, false, true},
		{"Test: yamlLicense: xlarge, currentLicense: medium", args{yl5, cl3}, false, true},

		// TODO: Make necessary changes in source code for the following tests to pass
		// {"Test: yamlLicense: abc, currentLicense: small", args{"abc", "small"}, false, true},
		// {"Test: yamlLicense: yl, currentLicense: small", args{"yl", "small"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isValidLicenseTier(tt.args.yamlLicense, tt.args.currentLicense)
			if (err != nil) != tt.wantErr {
				t.Errorf("isValidLicenseTier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isValidLicenseTier() = %v, want %v", got, tt.want)
			}
		})
	}
}
