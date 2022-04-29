package coverage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"golang.org/x/sync/errgroup"

	"github.com/LambdaTest/test-at-scale/pkg/fileutils"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

const (
	coverageJSONFileName = "coverage-final.json"
	mergedcoverageJSON   = "coverage-merged.json"
	compressedFileName   = "coverage-files.tzst"
	manifestJSONFileName = "manifest.json"
	coverageFilePath     = "/scripts/mapCoverage.js"
)

type codeCoverageService struct {
	logger               lumber.Logger
	execManager          core.ExecutionManager
	codeCoveragParentDir string
	azureClient          core.AzureClient
	zstd                 core.ZstdCompressor
	httpClient           http.Client
	endpoint             string
}

// New returns a new instance of CoverageService
func New(execManager core.ExecutionManager,
	azureClient core.AzureClient,
	zstd core.ZstdCompressor,
	cfg *config.NucleusConfig,
	logger lumber.Logger) (core.CoverageService, error) {
	// if coverage mode not enabled do not initialize the service
	if !cfg.CoverageMode {
		return nil, nil
	}
	if _, err := os.Stat(global.CodeCoverageDir); os.IsNotExist(err) {
		return nil, errors.New("coverage directory not mounted")
	}
	return &codeCoverageService{
		logger:               logger,
		execManager:          execManager,
		azureClient:          azureClient,
		zstd:                 zstd,
		codeCoveragParentDir: global.CodeCoverageDir,
		endpoint:             global.NeuronHost + "/coverage",
		httpClient: http.Client{
			Timeout: global.DefaultHTTPTimeout,
		}}, nil

}

//mergeCodeCoverageFiles merge all the coverage.json into single entity
func (c *codeCoverageService) mergeCodeCoverageFiles(ctx context.Context, commitDir, coverageManifestPath string, threshold bool) error {
	if _, err := os.Stat(commitDir); os.IsNotExist(err) {
		c.logger.Errorf("coverage files not found, skipping merge")
		return nil
	}

	coverageFiles := make([]string, 0)
	if err := filepath.WalkDir(commitDir, func(path string, d fs.DirEntry, err error) error {
		//add all individual coverage json files
		if d.Name() == coverageJSONFileName {
			coverageFiles = append(coverageFiles, path)
		}
		return nil
	}); err != nil {
		return err
	}

	if len(coverageFiles) < 1 {
		return errors.New("no coverage dirs found")
	}

	command := fmt.Sprintf("/scripts/node_modules/.bin/babel-node %s --commitDir %s --coverageFiles '%s'",
		coverageFilePath, commitDir, strings.Join(coverageFiles, " "))
	if threshold {
		command = fmt.Sprintf("%s --coverageManifest %s", command, coverageManifestPath)
	}
	commands := []string{command}
	return c.execManager.ExecuteInternalCommands(ctx, core.CoverageMerge, commands, "", nil, nil)
}

// MergeAndUpload compress the file and upload in azure blob
func (c *codeCoverageService) MergeAndUpload(ctx context.Context, payload *core.Payload) error {
	var parentCommitDir, repoDir string
	var g errgroup.Group
	// change variable name
	repoDir = filepath.Join(c.codeCoveragParentDir, payload.OrgID, payload.RepoID)
	repoBlobPath := path.Join(payload.GitProvider, payload.OrgID, payload.RepoID)

	// skip downloading if parent commit does not exists for the repository
	if payload.ParentCommitCoverageExists {
		coverage, err := c.getParentCommitCoverageDir(payload.RepoID, payload.BuildBaseCommit)
		if err != nil {
			return err
		}
		if err = c.downloadAndDecompressParentCommitDir(ctx, coverage, repoDir); err != nil {
			return err
		}
		parentCommitDir = filepath.Join(repoDir, coverage.ParentCommit)
	}
	coveragePayload := make([]coverageData, 0, len(payload.Commits))

	for _, commit := range payload.Commits {
		commitDir := filepath.Join(repoDir, commit.Sha)
		c.logger.Debugf("commit directory %s", commitDir)

		if _, err := os.Stat(commitDir); os.IsNotExist(err) {
			c.logger.Errorf("code coverage directory not found commit id %s", commit.Sha)
			return err
		}
		coverageManifestPath := filepath.Join(commitDir, manifestJSONFileName)

		manifestPayload, err := c.parseManifestFile(coverageManifestPath)
		if err != nil {
			c.logger.Errorf("failed to parse manifest file: %s, error :%v", commitDir, err)
			return err
		}
		//skip copy of parent directory if all test files executed
		if !manifestPayload.AllFilesExecuted {
			if err := c.copyFromParentCommitDir(parentCommitDir, commitDir, manifestPayload.Removedfiles...); err != nil {
				c.logger.Errorf("failed to copy coverage files from %s to %s, error :%v", parentCommitDir, commitDir, err)
				return err
			}
		}
		thresholdEnabled := false
		if manifestPayload.CoverageThreshold != nil {
			thresholdEnabled = true
		}
		if err := c.mergeCodeCoverageFiles(ctx, commitDir, coverageManifestPath, thresholdEnabled); err != nil {
			c.logger.Errorf("failed to merge coverage files %v", err)
			return err
		}
		c.logger.Debugf("compressed file name %v", compressedFileName)

		g.Go(func() error {
			if err := c.zstd.Compress(ctx, compressedFileName, false, repoDir, commit.Sha); err != nil {
				c.logger.Errorf("failed to compress coverage files %v", err)
				return err
			}
			_, err := c.uploadFile(ctx, repoBlobPath, compressedFileName, commit.Sha)
			return err
		})

		var blobURL string
		g.Go(func() error {
			blobURL, err = c.uploadFile(ctx, repoBlobPath, filepath.Join(commitDir, mergedcoverageJSON), commit.Sha)
			return err
		})

		var totalCoverage json.RawMessage
		g.Go(func() error {
			totalCoverage, err = c.getTotalCoverage(filepath.Join(commitDir, mergedcoverageJSON))
			return err
		})
		if err = g.Wait(); err != nil {
			c.logger.Errorf("failed to upload files to azure blob %v", err)
			return err
		}
		blobURL = strings.TrimSuffix(blobURL, fmt.Sprintf("/%s", mergedcoverageJSON))
		coveragePayload = append(coveragePayload, coverageData{BuildID: payload.BuildID, RepoID: payload.RepoID, CommitID: commit.Sha, BlobLink: blobURL, TotalCoverage: totalCoverage})
		//current commit dir becomes parent for next commit
		parentCommitDir = commitDir
	}
	return c.sendCoverageData(coveragePayload)
}

func (c *codeCoverageService) uploadFile(ctx context.Context, blobPath, filename, commitID string) (blobURL string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	mimeType := "application/json"
	if filepath.Ext(filename) == ".tzst" {
		mimeType = "application/zstd"
	}
	blobURL, err = c.azureClient.Create(ctx, fmt.Sprintf("%s/%s/%s", blobPath, commitID, filepath.Base(filename)), file, mimeType)
	return
}

func (c *codeCoverageService) parseManifestFile(filepath string) (core.CoverageManifest, error) {
	manifestPayload := core.CoverageManifest{}
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		c.logger.Errorf("manifest file not found in path %s", filepath)
		return manifestPayload, err
	}
	body, err := ioutil.ReadFile(filepath)
	if err != nil {
		return manifestPayload, err
	}

	err = json.Unmarshal(body, &manifestPayload)
	return manifestPayload, err
}

func (c *codeCoverageService) downloadAndDecompressParentCommitDir(ctx context.Context, coverage parentCommitCoverage, repoDir string) error {
	u, err := url.Parse(coverage.Bloblink)
	if err != nil {
		c.logger.Errorf("failed to parse blob link %s, error :%v", coverage.Bloblink, err)
		return err
	}
	u.Path = path.Join(u.Path, compressedFileName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Errorf("error while making http request %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 status while cloning from endpoint %s, status %d ", u.String(), resp.StatusCode)
	}

	parentCommitFilePath := filepath.Join(repoDir, coverage.ParentCommit+".tzst")
	c.logger.Debugf("parent commit file path %s", parentCommitFilePath)
	out, err := os.Create(parentCommitFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	// decompress the file in temp directory as we cannot decompress inside azure file volume
	if err := c.zstd.Decompress(ctx, parentCommitFilePath, false, os.TempDir()); err != nil {
		c.logger.Errorf("failed to decompress parent commit directory %v", err)
		return err
	}

	srcPath := filepath.Join(os.TempDir(), coverage.ParentCommit)
	destPath := filepath.Join(repoDir, coverage.ParentCommit)
	// copy the coverage directories to shared volume,
	// chmod is not allowed inside azure file volume so that is skipped Ref: https://stackoverflow.com/questions/58301985/permissions-on-azure-file
	if err := fileutils.CopyDir(srcPath, destPath, false); err != nil {
		c.logger.Errorf("failed to copy directory from src %s to dest %s, error %v", srcPath, destPath, err)
		return err
	}
	return nil
}

func (c *codeCoverageService) copyFromParentCommitDir(parentCommitDir, commitDir string, removedFiles ...string) error {
	if _, err := os.Stat(parentCommitDir); os.IsNotExist(err) {
		c.logger.Errorf("Parent Commit Directory %s not found", parentCommitDir)
		return err
	}
	if err := filepath.WalkDir(parentCommitDir, func(path string, info fs.DirEntry, err error) error {
		if info.IsDir() && info.Name() != filepath.Base(parentCommitDir) {
			if len(removedFiles) > 0 {
				for index, removedfile := range removedFiles {
					//if testfile is now removed don't copy to current commit directory
					if info.Name() == removedfile {
						//remove file from slice
						removedFiles = append(removedFiles[:index], removedFiles[index+1:]...)
						return filepath.SkipDir
					}
				}
			}
			testfileDir := filepath.Join(commitDir, info.Name())

			//TODO: check if copied dir size is not 0
			//if file already exists then don't copy from parent directory
			if _, err := os.Stat(testfileDir); os.IsNotExist(err) {
				if err := fileutils.CopyDir(path, testfileDir, false); err != nil {
					c.logger.Errorf("failed to copy directory from src %s to dest %s, error %v", path, testfileDir, err)
					return err
				}
			}
			//all files copied now we can move next sub directory
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (c *codeCoverageService) getParentCommitCoverageDir(repoID, commitID string) (coverage parentCommitCoverage, err error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		c.logger.Errorf("error while parsing endpoint %s, %v", c.endpoint, err)
		return coverage, err
	}
	q := u.Query()
	q.Set("repoID", repoID)
	q.Set("commitID", commitID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		c.logger.Errorf("failed to create new request %v", err)
		return coverage, err
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		c.logger.Errorf("error while getting coverage details for parent commitID %s, %v", commitID, err)
		return coverage, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Errorf("error while getting coverage data, status_code %d", resp.StatusCode)
		return coverage, errors.New("non 200 status")
	}
	payload := parentCommitCoverage{}
	decode := json.NewDecoder(resp.Body)

	if err := decode.Decode(&payload); err != nil {
		c.logger.Errorf("failed to decode response body %v", err)
		return coverage, err
	}
	c.logger.Infof("Got parent directory bloblink %s, commitID:%s", payload.Bloblink, payload.ParentCommit)

	return payload, nil
}

func (c *codeCoverageService) sendCoverageData(payload []coverageData) error {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		c.logger.Errorf("failed to marshal request body %v", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		c.logger.Errorf("failed to create new request %v", err)
		return err
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		c.logger.Errorf("error while sending coverage data %v", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Errorf("error while sending coverage data, status code %d", resp.StatusCode)
		return errors.New("non 200 status")
	}
	return nil
}

func (c *codeCoverageService) getTotalCoverage(filepath string) (json.RawMessage, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		c.logger.Errorf("coverage summary file not found in path %s", filepath)
		return nil, err
	}
	body, err := ioutil.ReadFile(filepath)
	if err != nil {
		c.logger.Errorf("failed to read coverage summary json, error: %v", err)
		return nil, err
	}

	var payload map[string]json.RawMessage
	if err = json.Unmarshal(body, &payload); err != nil {
		c.logger.Errorf("failed to unmarshal coverage summary json, error: %v", err)
		return nil, err
	}

	totalCoverage, ok := payload["total"]
	if !ok {
		c.logger.Errorf("total coverage summary not found in map")
		return nil, errors.New("total coverage summary not found in map")
	}
	return totalCoverage, nil
}
