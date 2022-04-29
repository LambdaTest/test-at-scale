package gitmanager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/command"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
)

func CreateDirectory(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Printf("Error: %v", err)
		}
	}
}

func removeFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}

func Test_copyAndExtractFile(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't get logger, error: %v", err)
	}
	var httpClient http.Client
	gm := &gitManager{
		logger:     logger,
		httpClient: httpClient,
	}
	fileBody := "Hello World!"
	resp := http.Response{
		Body: ioutil.NopCloser(bytes.NewBufferString(fileBody)),
	}
	path := "newFile"
	err2 := gm.copyAndExtractFile(context.TODO(), &resp, path)
	if err2 != nil {
		t.Errorf("Error: %v", err2)
		return
	}
	fileContent, err := ioutil.ReadFile("./newFile")
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	if string(fileContent) != fileBody {
		t.Errorf("Expected file content: %v\nReceived: %v", fileBody, string(fileContent))
	}
	defer removeFile(path)
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

func Test_gitManager_downloadFile(t *testing.T) {
	type fields struct {
		logger      lumber.Logger
		httpClient  http.Client
		execManager core.ExecutionManager
	}
	type args struct {
		ctx        context.Context
		archiveURL string
		fileName   string
		oauth      *core.Oauth
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := &gitManager{
				logger:      tt.fields.logger,
				httpClient:  tt.fields.httpClient,
				execManager: tt.fields.execManager,
			}
			if err := gm.downloadFile(tt.args.ctx, tt.args.archiveURL, tt.args.fileName, tt.args.oauth); (err != nil) != tt.wantErr {
				t.Errorf("gitManager.downloadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
