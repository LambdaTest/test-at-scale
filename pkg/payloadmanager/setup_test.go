// Package payloadmanager is used for fetching and validating the nucleus execution payload
package payloadmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func getPayloadManagerArgs() (core.AzureClient, lumber.Logger, *config.NucleusConfig, error) {
	logger, err := testutils.GetLogger()
	if err != nil {
		return nil, nil, nil, err
	}

	cfg, err := testutils.GetConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	var azureClient core.AzureClient
	return azureClient, logger, cfg, nil
}

func Test_payloadManager_FetchPayload(t *testing.T) {
	server := httptest.NewServer( // mock server
		http.FileServer(http.Dir("../../testutils/testdata")), // mock data stored at testutils/testdata/index.txt
	)
	defer server.Close()

	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't get logger, received: %s", err)
	}

	cfg, err := testutils.GetConfig()
	if err != nil {
		t.Errorf("Couldn't get config, received: %s", err)
	}

	ct := core.PayloadContainer
	azureClient := new(mocks.AzureClient)
	azureClient.On("GetSASURL", mock.AnythingOfType("*context.emptyCtx"), "/index.txt", ct).Return(
		func(ctc context.Context, blobPath string, containerType core.ContainerType) string {
			return server.URL + "/index.txt"
		},
		func(ctc context.Context, blobPath string, containerType core.ContainerType) error {
			return nil
		},
	)

	wantResp, err := ioutil.ReadFile("../../testutils/testdata/index.json")
	if err != nil {
		fmt.Printf("error in reading file: %+v\n", err)
	}

	pm := NewPayloadManger(azureClient, logger, cfg)

	type args struct {
		ctx            context.Context
		payloadAddress string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"Test Payload fetch for success", args{ctx: context.TODO(), payloadAddress: server.URL + "/index.txt"}, string(wantResp), false},

		{"Test Payload fetch for empty url", args{ctx: context.TODO(), payloadAddress: ""}, "<nil>", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pm.FetchPayload(tt.args.ctx, tt.args.payloadAddress)
			if tt.wantErr {
				if err == nil {
					t.Errorf("payloadManager.FetchPayload() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			received, _ := json.Marshal(got)
			receivedPayload := fmt.Sprintf("%+v\n", string(received))
			if receivedPayload != tt.want {
				t.Errorf("payloadManager.FetchPayload() = \n%v, \nwant: %v\n", receivedPayload, tt.want)
			}
		})
	}
}

func Test_payloadManager_ValidatePayload(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}

	type args struct {
		ctx            context.Context
		payload        *core.Payload
		coverageMode   bool
		locators       string
		locatorAddress string
		taskID         string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test validate payload for empty repolink", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: ""}}, true},

		{"Test validate payload for empty reposlug", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: ""}}, true},

		{"Test validate payload for empty gitprovider", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: ""}}, true},

		{"Test validate payload for empty buildID", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: ""}}, true},

		{"Test validate payload for empty repoID", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: ""}}, true},

		{"Test validate payload for empty branchName", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: ""}}, true},

		{"Test validate payload for empty orgID", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: ""}}, true},

		{"Test validate payload for empty TASFileName", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: ""}}, true},

		{"Test validate payload for expected payload.Locator Address & payloadLocator", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "a"}, locators: "/locator", locatorAddress: "/locatorAddr"}, true},

		{"Test validate payload for empty build target commit", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: ""}}, true},

		{"Test validate payload for empty target commit in config", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: "btg"}, coverageMode: false, locators: "../dummy"}, true},

		{"Test validate payload for target & base commit in config", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: "btg"}, coverageMode: false, locators: "../dummy"}, true},

		{"Test validate payload for target, base commit & taskID in config", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: "btg"}, coverageMode: false, locators: "../dummy", taskID: "tid"}, true},

		{"Test validate payload for non push and pull event", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: "btg", EventType: "invalid"}, coverageMode: true}, true},

		{"Test validate payload for push event with nil commit", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: "btg", EventType: "push", Commits: []core.CommitChangeList{}}, coverageMode: true}, true},

		{"Test validate payload for success", args{ctx: context.TODO(), payload: &core.Payload{RepoLink: "github.com/abc/", RepoSlug: "/slug", GitProvider: "fake", BuildID: "build", RepoID: "repo", BranchName: "branch", OrgID: "org", TasFileName: "tas", BuildTargetCommit: "btg", EventType: "push", Commits: []core.CommitChangeList{{Sha: "sha", Message: "msg"}}}, coverageMode: true}, false},
	}
	for _, tt := range tests {
		cfg.CoverageMode = tt.args.coverageMode
		cfg.LocatorAddress = tt.args.locatorAddress
		// cfg.BuildTargetCommit = tt.args.buildTargetCommit
		cfg.Locators = tt.args.locators
		// cfg.BuildBaseCommit = tt.args.buildBaseCommit
		cfg.TaskID = tt.args.taskID

		pm := NewPayloadManger(azureClient, logger, cfg)
		t.Run(tt.name, func(t *testing.T) {
			if err := pm.ValidatePayload(tt.args.ctx, tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("payloadManager.ValidatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if cfg.Locators != "" {
				if tt.args.payload.Locators != tt.args.locators {
					t.Errorf("payloadManager.ValidatePayload() payload.locatorAdress = %v, want: %v", tt.args.payload.LocatorAddress, tt.args.locators)
					return
				}
			}

			if cfg.LocatorAddress != "" {
				if tt.args.payload.LocatorAddress != tt.args.locatorAddress {
					t.Errorf("payloadManager.ValidatePayload() payload.locatorAdress = %v, want: %v", tt.args.payload.LocatorAddress, tt.args.locatorAddress)
					return
				}
			}
			if !(cfg.CoverageMode) {
				if cfg.TaskID != "" {
					if tt.args.payload.TaskID != tt.args.taskID {
						t.Errorf("got payload.TaskID: %v, want: %v", tt.args.payload.TaskID, tt.args.taskID)
					}
				}
			}

		})
	}
}
