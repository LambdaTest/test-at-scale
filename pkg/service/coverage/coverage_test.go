package coverage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/testutils"
	"github.com/LambdaTest/synapse/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func newCodeCoverageService(logger lumber.Logger, execManager *mocks.ExecutionManager, codeCoveragParentDir string, azureClient *mocks.AzureClient, zstd *mocks.ZstdCompressor, endpoint string) *codeCoverageService {
	return &codeCoverageService{
		logger:               logger,
		execManager:          execManager,
		codeCoveragParentDir: codeCoveragParentDir,
		azureClient:          azureClient,
		zstd:                 zstd,
		httpClient: http.Client{
			Timeout: global.DefaultHTTPTimeout,
		},
		endpoint: endpoint,
	}
}

func initialiseArgs() (logger lumber.Logger, execManager *mocks.ExecutionManager, azureClient *mocks.AzureClient, zstd *mocks.ZstdCompressor) {
	azureClient = new(mocks.AzureClient)
	execManager = new(mocks.ExecutionManager)
	zstdCompressor := new(mocks.ZstdCompressor)

	logger, err := testutils.GetLogger()
	if err != nil {
		fmt.Printf("Couldn't initialise logger, error: %v", err)
	}
	return logger, execManager, azureClient, zstdCompressor
}

func removeCreatedFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}

func Test_codeCoverageService_mergeCodeCoverageFiles(t *testing.T) {
	logger, execManager, azureClient, zstdCompressor := initialiseArgs()

	var receivedArgs string
	execManager.On("ExecuteInternalCommands", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("core.CommandType"), mock.AnythingOfType("[]string"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			receivedArgs = fmt.Sprintf("%+v %+v %+v %+v %+v", commandType, commands, cwd, envMap, secretData)
			return nil
		},
	)

	type args struct {
		ctx                  context.Context
		commitDir            string
		coverageManifestPath string
		threshold            bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    string
	}{
		{"Test",
			args{
				ctx:                  context.TODO(),
				commitDir:            "../../../testutils/testdata",
				coverageManifestPath: "../../../testutils/testdata/coverage",
				threshold:            true,
			},
			false,
			"coveragemerge [/scripts/node_modules/.bin/babel-node /scripts/mapCoverage.js --commitDir ../../../testutils/testdata --coverageFiles '../../../testutils/testdata/coverage/coverage-final.json ../../../testutils/testdata/coverage/sample/coverage-final.json' --coverageManifest ../../../testutils/testdata/coverage]  map[] map[]",
		},
	}

	c := newCodeCoverageService(logger, execManager, "../../../testutils/testdata/coverage", azureClient, zstdCompressor, "endpoint")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.mergeCodeCoverageFiles(tt.args.ctx, tt.args.commitDir, tt.args.coverageManifestPath, tt.args.threshold)
			if err != nil != tt.wantErr {
				t.Errorf("codeCoverageService.mergeCodeCoverageFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if receivedArgs != tt.want {
				t.Errorf("Expected: \n%v\nreceived: \n%v", tt.want, receivedArgs)
			}
		})
	}
}

func Test_codeCoverageService_uploadFile(t *testing.T) {
	logger, execManager, azureClient, zstdCompressor := initialiseArgs()

	var calledArgs string
	azureClient.On("Create", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("*os.File"), mock.AnythingOfType("string")).Return(
		func(ctx context.Context, path string, reader io.Reader, mimeType string) string {
			st, _ := ioutil.ReadAll(reader)
			calledArgs = fmt.Sprintf("%v %v %v", path, string(st), mimeType)
			return "blobURL"
		},
		func(ctx context.Context, path string, reader io.Reader, mimeType string) error {
			return nil
		},
	)

	type args struct {
		ctx      context.Context
		blobPath string
		filename string
		commitID string
	}
	tests := []struct {
		name        string
		args        args
		wantBlobURL string
		wantArgs    string
		wantErr     bool
	}{
		{"Test uploadFile",
			args{
				ctx:      context.TODO(),
				blobPath: "blobpath",
				filename: "../../../testutils/testdata/coverage/coverage-final.json",
				commitID: "cID",
			},
			"blobURL",
			`blobpath/cID/coverage-final.json {
    "cover1" : "f1"
} application/json`,
			false,
		},
	}
	c := newCodeCoverageService(logger, execManager, "", azureClient, zstdCompressor, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBlobURL, err := c.uploadFile(tt.args.ctx, tt.args.blobPath, tt.args.filename, tt.args.commitID)
			if (err != nil) != tt.wantErr {
				t.Errorf("codeCoverageService.uploadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotBlobURL != tt.wantBlobURL {
				t.Errorf("codeCoverageService.uploadFile() = %v, want %v", gotBlobURL, tt.wantBlobURL)
				return
			}
			if tt.wantArgs != calledArgs {
				t.Errorf("Expected: \n%v\nreceived: \n%v", tt.wantArgs, calledArgs)
			}
		})
	}
}

func Test_codeCoverageService_parseManifestFile(t *testing.T) {
	logger, execManager, azureClient, zstdCompressor := initialiseArgs()

	type args struct {
		filepath string
	}
	tests := []struct {
		name    string
		args    args
		want    core.CoverageManifest
		wantErr bool
	}{
		{"Test parseManifestFile for success",
			args{filepath: "../../../testutils/testdata/coverage/coverage-final.json"},
			core.CoverageManifest{},
			false,
		},
		{"Test parseManifestFile",
			args{filepath: "../../../testutils/testdata/coverage/dne.json"},
			core.CoverageManifest{},
			true,
		},
	}
	c := newCodeCoverageService(logger, execManager, "", azureClient, zstdCompressor, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.parseManifestFile(tt.args.filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("codeCoverageService.parseManifestFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("codeCoverageService.parseManifestFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_codeCoverageService_downloadAndDecompressParentCommitDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coverage-files.tzst" {
			t.Errorf("Expected to request '/coverage-files.tzst', got: %v", r.URL)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	logger, execManager, azureClient, zstdCompressor := initialiseArgs()
	zstdCompressor.On("Decompress", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), false, mock.AnythingOfType("string")).Return(
		func(ctx context.Context, filePath string, preservePath bool, workingDirectory string) error {
			return nil
		},
	)

	type args struct {
		ctx      context.Context
		coverage parentCommitCoverage
		repoDir  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add success case, currently on local tempdir can't be created
		{"Test downloadAndDecompressParentCommitDir",
			args{ctx: context.TODO(), coverage: parentCommitCoverage{Bloblink: server.URL, ParentCommit: "parentCommit"}, repoDir: "../../../testutils/testdata"},
			true,
		},
	}
	c := newCodeCoverageService(logger, execManager, "", azureClient, zstdCompressor, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.downloadAndDecompressParentCommitDir(tt.args.ctx, tt.args.coverage, tt.args.repoDir)

			defer removeCreatedFile(filepath.Join(tt.args.repoDir, tt.args.coverage.ParentCommit+".tzst"))

			if (err != nil) != tt.wantErr {
				t.Errorf("codeCoverageService.downloadAndDecompressParentCommitDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_codeCoverageService_getParentCommitCoverageDir(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if r.URL.RawQuery == "commitID=non200&repoID=non200" {
			w.WriteHeader(300)
			return
		}

		if r.URL.RawQuery == "commitID=payloadError&repoID=payloadDecodeError" {
			_, writeErr := fmt.Fprintln(w, `{"undefined_field"}`)
			if writeErr != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")

		_, writeErr := fmt.Fprintln(w, `{"blob_link": "http://fakeblob.link", "parent_commit" : "fake_parent_commit"}`)
		if writeErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	logger, execManager, azureClient, zstdCompressor := initialiseArgs()
	type args struct {
		repoID   string
		commitID string
	}
	tests := []struct {
		name         string
		args         args
		wantCoverage parentCommitCoverage
		wantErr      bool
	}{
		{"Test getParentCommitCoverageDir", args{repoID: "dummyRepoID", commitID: "dummyCommitID"}, parentCommitCoverage{Bloblink: "http://fakeblob.link", ParentCommit: "fake_parent_commit"}, false},

		{"Test getParentCommitCoverageDir for non 200 status error", args{repoID: "non200", commitID: "non200"}, parentCommitCoverage{}, true},

		{"Test getParentCommitCoverageDir for payloadDecodeError", args{repoID: "payloadDecodeError", commitID: "payloadError"}, parentCommitCoverage{}, true},
	}
	c := newCodeCoverageService(logger, execManager, "", azureClient, zstdCompressor, ts.URL)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCoverage, err := c.getParentCommitCoverageDir(tt.args.repoID, tt.args.commitID)
			if (err != nil) != tt.wantErr {
				t.Errorf("codeCoverageService.getParentCommitCoverageDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCoverage, tt.wantCoverage) {
				t.Errorf("codeCoverageService.getParentCommitCoverageDir() = %v, want %v", gotCoverage, tt.wantCoverage)
			}
		})
	}
}

func Test_codeCoverageService_sendCoverageData(t *testing.T) {
	payload := []coverageData{
		{
			BuildID:       "buildID1",
			RepoID:        "repoID1",
			CommitID:      "commitID1",
			BlobLink:      "blobLink1",
			TotalCoverage: json.RawMessage([]byte(`{"bar":"baz"}`)),
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/endpoint", func(res http.ResponseWriter, req *http.Request) {
		body, _ := ioutil.ReadAll(req.Body)
		expResp := `[{"build_id":"buildID1","repo_id":"repoID1","commit_id":"commitID1","blob_link":"blobLink1","total_coverage":{"bar":"baz"}}]`
		if !reflect.DeepEqual(string(body), expResp) {
			t.Errorf("Expected response body: %v, got: %v\n", expResp, string(body))
		}
		res.WriteHeader(200)
	})
	mux.HandleFunc("/endpoint-err", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(404)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	logger, execManager, azureClient, zstdCompressor := initialiseArgs()
	type args struct {
		payload []coverageData
	}
	tests := []struct {
		name     string
		args     args
		endpoint string
		wantErr  bool
	}{
		{"Test sendCoverageData for success", args{payload: payload}, "/endpoint", false},

		{"Test sendCoverageData for non 200 status", args{payload: payload}, "/endpoint-err", true},
	}
	for _, tt := range tests {
		c := newCodeCoverageService(logger, execManager, "", azureClient, zstdCompressor, ts.URL+tt.endpoint)
		t.Run(tt.name, func(t *testing.T) {
			if err := c.sendCoverageData(tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("codeCoverageService.sendCoverageData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_codeCoverageService_getTotalCoverage(t *testing.T) {
	logger, execManager, azureClient, zstdCompressor := initialiseArgs()
	c := newCodeCoverageService(logger, execManager, "", azureClient, zstdCompressor, "")
	type args struct {
		filepath string
	}
	tests := []struct {
		name    string
		args    args
		want    json.RawMessage
		wantErr bool
	}{
		{"Test getTotalCoverage", args{"../../../testutils/testdata/coverage/sample/coverage-final.json"}, json.RawMessage([]byte(`"80%"`)), false},

		{"Test getTotalCoverage for no field of total coverage", args{"../../../testutils/testdata/coverage/coverage-final.json"}, json.RawMessage{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.getTotalCoverage(tt.args.filepath)
			if (err != nil) != tt.wantErr {
				t.Errorf("codeCoverageService.getTotalCoverage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.want) > 0 && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("codeCoverageService.getTotalCoverage() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}
