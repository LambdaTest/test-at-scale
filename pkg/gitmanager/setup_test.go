package gitmanager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/LambdaTest/test-at-scale/mocks"
	"github.com/LambdaTest/test-at-scale/pkg/command"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/mock"
)

func Test_downloadFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/archive/zipfile.zip" {
			t.Errorf("Expected to request '/archive/zipfile.zip', got: %v", r.URL)
			return
		}
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		expectedOauth := &core.Oauth{AccessToken: "dummy", Type: core.Bearer}
		if splitToken[1] != expectedOauth.AccessToken {
			t.Errorf("Invalid clone token, expected: %v\nreceived: %v", expectedOauth.AccessToken, splitToken[1])
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't get logger, error: %v", err)
		return
	}
	var httpClient http.Client
	execManager := new(mocks.ExecutionManager)
	execManager.On("ExecuteInternalCommands",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("core.CommandType"),
		mock.AnythingOfType("[]string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("map[string]string"),
		mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			return nil
		},
	)
	gm := &gitManager{
		logger:      logger,
		httpClient:  httpClient,
		execManager: execManager,
		request:     requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{}),
	}
	archiveURL := server.URL + "/archive/zipfile.zip"
	fileName := "copyAndExtracted"
	oauth := &core.Oauth{AccessToken: "dummy", Type: core.Bearer}
	err2 := gm.downloadFile(context.TODO(), archiveURL, fileName, oauth)
	defer removeFile(fileName) // remove the file created after downloading and extracting
	if err2 != nil {
		t.Errorf("Error: %v", err2)
	}
}

func Test_copyAndExtractFile(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't get logger, error: %v", err)
	}
	var httpClient http.Client
	execManager := new(mocks.ExecutionManager)
	execManager.On("ExecuteInternalCommands",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("core.CommandType"),
		mock.AnythingOfType("[]string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("map[string]string"),
		mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			return nil
		},
	)
	gm := &gitManager{
		logger:      logger,
		httpClient:  httpClient,
		execManager: execManager,
		request:     requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{}),
	}
	fileBody := "Hello World!"
	resp := http.Response{
		Body: io.NopCloser(bytes.NewBufferString(fileBody)),
	}
	path := "newFile"
	defer removeFile(path)
	respBodyBuffer := bytes.Buffer{}
	_, err = io.Copy(&respBodyBuffer, resp.Body)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	err2 := gm.copyAndExtractFile(context.TODO(), respBodyBuffer.Bytes(), path)
	if err2 != nil {
		t.Errorf("Error: %v", err2)
		return
	}

	fileContent, err := os.ReadFile("./newFile")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}

	if string(fileContent) != fileBody {
		t.Errorf("Expected file content: %v\nReceived: %v", fileBody, string(fileContent))
	}
}

func TestClone(t *testing.T) {
	checkClone := func(t *testing.T) {
		server := httptest.NewServer( // mock server
			http.FileServer(http.Dir("../../testutils/testdata/archive")), // mock data stored at tests/mock/index.txt
		)
		defer server.Close()
		global.TestEnv = true
		global.TestServer = server.URL
		logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
		if err != nil {
			fmt.Println("Logger can't be established")
		}
		azureClient := new(mocks.AzureClient)
		secretParser := new(mocks.SecretParser)
		execManager := command.NewExecutionManager(secretParser, azureClient, logger)
		gm := NewGitManager(logger, execManager)

		payload, err := testutils.GetPayload()
		if err != nil {
			t.Errorf("Unable to load payload, error %v", err)
		}

		payload.RepoLink = server.URL
		payload.BuildTargetCommit = "testRepo"
		oauth := &core.Oauth{AccessToken: "dummy", Type: core.Bearer}
		commitID := payload.BuildTargetCommit

		err = gm.Clone(context.TODO(), payload, oauth)
		global.TestEnv = false
		expErr := "opening zip archive for reading: creating reader: zip: not a valid zip file"

		defer removeFile("testRepo")
		defer removeFile(commitID + ".zip")
		defer removeFile(global.RepoDir)

		if err != nil && err.Error() != expErr {
			t.Errorf("Error: %v", err)
			return
		}

		_, err2 := os.OpenFile(commitID+".zip", 0440, 0440)
		_, err3 := os.OpenFile("zipFile", 0440, 0440)

		// check if downloaded file exist now
		if errors.Is(err2, os.ErrNotExist) {
			t.Errorf("Could not find the downloaded file, got error: %v", err2)
			return
		}
		if err.Error() == expErr {
			return
		}
		// check if unzipped folder exist
		if errors.Is(err3, os.ErrNotExist) {
			t.Errorf("Could not find the unzipped folder, got error: %v", err3)
			return
		}

		// global.RepoDir does not exist on local
		if err != nil && (errors.Is(err, os.ErrNotExist)) == false {
			t.Errorf("Expected error: %v, Received: %v\n", os.ErrNotExist, err)
			return
		}

		if err == nil {
			if _, err4 := os.OpenFile(global.RepoDir, 0440, 0440); errors.Is(err4, os.ErrNotExist) {
				t.Errorf("Failed to find file in global repodir, got error: %v", err4)
				return
			}
		}
	}
	t.Run("Check the clone function", func(t *testing.T) {
		checkClone(t)
	})
}

func removeFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}
