// Package gitmanager is used for cloning repo
package gitmanager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/urlmanager"
	"github.com/mholt/archiver/v3"
)

type gitManager struct {
	logger      lumber.Logger
	httpClient  http.Client
	execManager core.ExecutionManager
}

// NewGitManager returns a new GitManager
func NewGitManager(logger lumber.Logger, execManager core.ExecutionManager) core.GitManager {
	return &gitManager{
		logger: logger,
		httpClient: http.Client{
			Timeout: global.DefaultHTTPTimeout,
		},
		execManager: execManager,
	}
}

func (gm *gitManager) Clone(ctx context.Context, payload *core.Payload, cloneToken string) error {
	repoLink := payload.RepoLink
	repoItems := strings.Split(repoLink, "/")
	repoName := repoItems[len(repoItems)-1]
	orgName := repoItems[len(repoItems)-2]
	commitID := payload.BuildTargetCommit
	archiveURL, err := urlmanager.GetCloneURL(payload.GitProvider, repoLink, repoName, commitID)
	if err != nil {
		gm.logger.Errorf("failed to get clone url for provider %s, error %v", payload.GitProvider, err)
		return err
	}
	gm.logger.Debugf("cloning from %s", archiveURL)
	err = gm.downloadFile(ctx, archiveURL, commitID+".zip", cloneToken)
	if err != nil {
		gm.logger.Errorf("failed to download file %v", err)
		return err
	}

	filename := repoName + "-" + commitID
	if payload.GitProvider == core.Bitbucket {
		// commitID[:12] bitbucket shorthand commit sha
		filename = orgName + "-" + repoName + "-" + commitID[:12]
	}

	if err = os.Rename(filename, global.RepoDir); err != nil {
		gm.logger.Errorf("failed to rename dir, error %v", err)
		return err
	}

	if err = gm.initGit(ctx, payload, cloneToken); err != nil {
		gm.logger.Errorf("failed to initialize git, error %v", err)
		return err
	}

	return nil
}

// downloadFile clones the archive from github and extracts the file if it is a zip file.
func (gm *gitManager) downloadFile(ctx context.Context, archiveURL, fileName, cloneToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return err
	}
	if cloneToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cloneToken))
	}
	resp, err := gm.httpClient.Do(req)
	if err != nil {
		gm.logger.Errorf("error while making http request %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		gm.logger.Errorf("non 200 status while cloning from endpoint %s, status %d ", archiveURL, resp.StatusCode)
		return errs.ErrApiStatus
	}
	err = gm.copyAndExtractFile(resp, fileName)
	if err != nil {
		gm.logger.Errorf("failed to copy file %v", err)
		return err
	}
	return nil
}

// copyAndExtractFile copies the content of http response directly to the local storage
// and extracts the file if it is a zip file.
func (gm *gitManager) copyAndExtractFile(resp *http.Response, path string) error {
	out, err := os.Create(path)
	if err != nil {
		gm.logger.Errorf("failed to create file err %v", err)
		return err
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		gm.logger.Errorf("failed to copy file %v", err)
		out.Close()
		return err
	}
	out.Close()

	// if zip file, then unarchive the file in same path
	if filepath.Ext(path) == ".zip" {
		zip := archiver.NewZip()
		zip.OverwriteExisting = true
		if err := zip.Unarchive(path, filepath.Dir(path)); err != nil {
			gm.logger.Errorf("failed to unarchive file %v", err)
			return err

		}
	}
	return err
}

func (gm *gitManager) initGit(ctx context.Context, payload *core.Payload, cloneToken string) error {
	branch := payload.BranchName
	repoURL, perr := url.Parse(payload.RepoLink)
	if perr != nil {
		return perr
	}
	repoURL.User = url.UserPassword("x-token-auth", cloneToken)
	urlWithToken := repoURL.String()
	commands := []string{
		"git init",
		fmt.Sprintf("git remote add origin %s.git", payload.RepoLink),
		fmt.Sprintf("git config --global url.%s.InsteadOf %s", urlWithToken, payload.RepoLink),
		fmt.Sprintf("git fetch --depth=1 origin +%s:refs/remotes/origin/%s", payload.BuildTargetCommit, branch),
		fmt.Sprintf("git config --global --remove-section url.%s", urlWithToken),
		fmt.Sprintf("git checkout --progress --force -B %s refs/remotes/origin/%s", branch, branch),
	}
	if err := gm.execManager.ExecuteInternalCommands(ctx, core.InitGit, commands, global.RepoDir, nil, nil); err != nil {
		return err
	}
	return nil
}
