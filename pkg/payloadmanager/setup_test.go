package payloadmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/testutils"
	"github.com/LambdaTest/synapse/testutils/mocks"
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

func TestFetchPayload(t *testing.T) {
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
		})
	pm := NewPayloadManger(azureClient, logger, cfg)

	checkFetch := func(t *testing.T, url string) {
		t.Helper()
		receivedPayload, err := pm.FetchPayload(context.TODO(), url)
		expectedError := "invalid payload address"
		errGot := fmt.Sprintf("%v", err)

		if url == "" && errGot != expectedError {
			t.Errorf("Unexpected error for empty url, error: %v", err)
		}

		if err != nil {
			t.Errorf("Error in fetching payload for URL, received: %v", err)
		}

		expectedPayload := testutils.PayloadCheck
		receivedPayloadStr := fmt.Sprintf("%v", receivedPayload) // converting received payload to string for easy comparison

		if receivedPayloadStr != expectedPayload {
			t.Errorf("\nReceived payload: %v\nexpected payload: %v", receivedPayloadStr, expectedPayload)
		}
	}

	checkFetchEmptyURL := func(t *testing.T, url string) {
		t.Helper()
		_, err := pm.FetchPayload(context.TODO(), url)
		expectedError := "invalid payload address"
		errGot := fmt.Sprintf("%v", err)

		if url == "" && errGot != expectedError {
			t.Errorf("Unexpected error for empty url, error: %v", err)
		}
	}

	checkValidation := func(t *testing.T) {
		t.Helper()
		_, err := testutils.GetPayload()
		if err != nil {
			t.Errorf("error in validating payload, error %v", err.Error())
		}
	}

	t.Run("TestPayloadFetch", func(t *testing.T) {
		url := server.URL
		checkFetch(t, url+"/index.txt")
	})
	t.Run("TestPayloadFetch using empty URL", func(t *testing.T) {
		checkFetchEmptyURL(t, "")
	})
	t.Run("TestPayloadValidation", func(t *testing.T) {
		checkValidation(t)
	})
}

func TestValidatePayloadForEmptyRepoLink(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}
	payload.RepoLink = ""
	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing repo link")

	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyRepoSlug(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}
	payload.RepoSlug = ""
	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errors.New("Missing repo slug")

	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyGitProvider(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}

	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
	}

	payload.GitProvider = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing git provider")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyBuildID(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)

	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.BuildID = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing BuildID")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyRepoID(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.RepoID = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing RepoID")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyBranchName(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.BranchName = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing Branch Name")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyOrgID(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.OrgID = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing OrgID")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyTASFileName(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.TasFileName = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing tas yml filename")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyTaskID(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = false
	cfg.ParseMode = false
	cfg.Locators = "c/d"
	cfg.LocatorAddress = "test/s/"
	cfg.TargetCommit = "reusfffuv"
	cfg.TaskID = ""
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.TaskID = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing taskID in config")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyBuildTargetCommit(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = false
	cfg.ParseMode = false
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.BuildTargetCommit = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing build target commit")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForEmptyTargetCommitInCfg(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = false
	cfg.ParseMode = false
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.TargetCommit = ""

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing targetCommit in config")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForTaskIDInCfg(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = false
	cfg.ParseMode = false
	cfg.TargetCommit = "reusfffuv"
	cfg.TaskID = "gtuh"
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	err2 := pm.ValidatePayload(context.TODO(), payload)
	if payload.TaskID != cfg.TaskID {
		t.Errorf("Payload TaskID is not same as config TaskId, Received error: %v", err2)
	}
}

func TestValidatePayloadForInvalidEvent(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = true
	cfg.ParseMode = false
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.Diff = ""
	payload.EventType = "invalid"

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Invalid event type")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForPushEvent(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = true
	cfg.ParseMode = false
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.Diff = ""
	payload.EventType = "push"

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("Missing commits error")
	if err2 == nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}

func TestValidatePayloadForSucess(t *testing.T) {
	azureClient, logger, cfg, err := getPayloadManagerArgs()
	if err != nil {
		t.Errorf("Couldn't establish required arguments, error: %v", err)
		return
	}
	cfg.CoverageMode = true
	cfg.ParseMode = false
	pm := NewPayloadManger(azureClient, logger, cfg)
	payload, err := testutils.GetPayload()
	if err != nil {
		t.Errorf("Couldn't get payload, error: %v", err)
		return
	}

	payload.Diff = ""
	payload.EventType = "push"
	payload.Commits = make([]core.CommitChangeList, 1)
	payload.Commits = append(payload.Commits, core.CommitChangeList{Sha: "yetr3", Link: "heuf"})

	err2 := pm.ValidatePayload(context.TODO(), payload)
	expectedErr := errs.ErrInvalidPayload("")
	if err2 != nil {
		t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
	}
}
