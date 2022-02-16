package payloadmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/testUtils"
	"github.com/LambdaTest/synapse/testUtils/mocks"
	"github.com/stretchr/testify/mock"
)

func TestFetchPayload(t *testing.T) {
	server := httptest.NewServer( // mock server
		http.FileServer(http.Dir("../../testUtils/testdata")), // mock data stored at testUtils/testdata/index.txt
	)
	defer server.Close()

	logger, err := testUtils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't get logger, received: %s", err)
	}

	cfg, err := testUtils.GetConfig()
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

		expectedPayload := testUtils.Payload_old
		receivedPayloadStr := fmt.Sprintf("%v", receivedPayload) // converting recieved payload to string for easy comparision

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
		_, err := testUtils.GetPayload()
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

func TestValidatePayload(t *testing.T) {
	logger, err := testUtils.GetLogger()
	if err != nil {
		t.Errorf("Unable to get logger, error %v", err.Error())
	}

	cfg, err := testUtils.GetConfig()
	if err != nil {
		t.Errorf("Couldn't get config, received: %s", err)
	}

	checkEmptyRepoLink := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}
		payload.RepoLink = ""
		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing repo link")

		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyRepoSlug := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}
		payload.RepoSlug = ""
		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errors.New("Missing repo slug")

		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyGitProvider := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
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

	checkEmptyBuildId := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.BuildID = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing BuildID")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyRepoId := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.RepoID = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing RepoID")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyBranchName := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.BranchName = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing Branch Name")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyOrgID := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.OrgID = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing OrgID")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyTASFileName := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.TasFileName = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing tas yml filename")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkMissingTaskID := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		cfg.Locators = "c/d"
		cfg.LocatorAddress = "test/s/"
		cfg.TargetCommit = "reusfffuv"
		cfg.TaskID = ""
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.TaskID = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing taskID in config")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyBuildTargetCommit := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.BuildTargetCommit = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing build target commit")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkEmptyTargetCommit := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.TargetCommit = ""

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing targetCommit in config")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkNonEmptyCfgTaskID := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		cfg.TargetCommit = "reusfffuv"
		cfg.TaskID = "gtuh"
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		err2 := pm.ValidatePayload(context.TODO(), payload)
		if payload.TaskID != cfg.TaskID {
			t.Errorf("Payload TaskID is not same as config TaskId, Received error: %v", err2)
		}
	}

	checkInvalidEvent := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.Diff = ""
		payload.EventType = "invalid"

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Invalid event type")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkCommitError := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
		}

		payload.Diff = ""
		payload.EventType = "push"

		err2 := pm.ValidatePayload(context.TODO(), payload)
		expectedErr := errs.ErrInvalidPayload("Missing commits error")
		if err2 == nil {
			t.Errorf("Expected error: %v, Received error: %v", expectedErr, err2)
		}
	}

	checkNilError := func(t *testing.T, coverageMode, parseMode bool) {
		t.Helper()
		cfg.CoverageMode = coverageMode
		cfg.ParseMode = parseMode
		var azureClient core.AzureClient
		pm := NewPayloadManger(azureClient, logger, cfg)
		payload, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Couldn't get payload, error: %v", err)
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

	t.Run("TestValidatePayload for empty repolink", func(t *testing.T) {
		checkEmptyRepoLink(t, false, false)
	})
	t.Run("TestValidatePayload for empty reposlug", func(t *testing.T) {
		checkEmptyRepoSlug(t, false, false)
	})
	t.Run("TestValidatePayload for empty Git provider", func(t *testing.T) {
		checkEmptyGitProvider(t, false, false)
	})
	t.Run("TestValidatePayload for missing BuildID", func(t *testing.T) {
		checkEmptyBuildId(t, false, false)
	})
	t.Run("TestValidatePayload for missing RepoID", func(t *testing.T) {
		checkEmptyRepoId(t, false, false)
	})
	t.Run("TestValidatePayload for missing BranchName", func(t *testing.T) {
		checkEmptyBranchName(t, false, false)
	})
	t.Run("TestValidatePayload for missing OrgID", func(t *testing.T) {
		checkEmptyOrgID(t, false, false)
	})
	t.Run("TestValidatePayload for missing TAS file name", func(t *testing.T) {
		checkEmptyTASFileName(t, false, false)
	})
	t.Run("TestValidatePayload for missing target commit", func(t *testing.T) {
		checkEmptyBuildTargetCommit(t, false, false)
	})
	t.Run("TestValidatePayload for missing target commit in config", func(t *testing.T) {
		checkEmptyTargetCommit(t, false, false)
	})
	t.Run("TestValidatePayload for missing taskid in config", func(t *testing.T) {
		checkMissingTaskID(t, false, false)
	})
	t.Run("TestValidatePayload for pull-request", func(t *testing.T) {
		checkInvalidEvent(t, true, false)
	})
	t.Run("TestValidatePayload for push event and commit length = 0", func(t *testing.T) {
		checkCommitError(t, true, false)
	})
	t.Run("TestValidatePayload for non-empty config TaskID", func(t *testing.T) {
		checkNonEmptyCfgTaskID(t, false, false)
	})
	t.Run("TestValidatePayload for success", func(t *testing.T) {
		checkNilError(t, true, false)
	})
}
