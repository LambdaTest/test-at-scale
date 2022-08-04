// Package gitmanager is used for cloning repo
package gitmanager

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/pkg/urlmanager"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
	"github.com/cenkalti/backoff/v4"
	"github.com/mholt/archiver/v3"
)

type GitLabSingleFileResponse struct {
	Content string `json:"content"`
}

type gitManager struct {
	logger      lumber.Logger
	httpClient  http.Client
	execManager core.ExecutionManager
	request     core.Requests
}

const (
	authorization = "Authorization"
)

// NewGitManager returns a new GitManager
func NewGitManager(logger lumber.Logger, execManager core.ExecutionManager) core.GitManager {
	return &gitManager{
		logger: logger,
		httpClient: http.Client{
			Timeout: global.DefaultGitCloneTimeout,
		},
		execManager: execManager,
		request:     requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{}),
	}
}

func (gm *gitManager) Clone(ctx context.Context, payload *core.Payload, oauth *core.Oauth) error {
	repoLink := payload.RepoLink
	repoItems := strings.Split(repoLink, "/")
	repoName := repoItems[len(repoItems)-1]
	commitID := payload.BuildTargetCommit

	archiveURL, err := urlmanager.GetCloneURL(payload.GitProvider, repoLink, repoName, commitID, payload.ForkSlug, payload.RepoSlug)
	if err != nil {
		gm.logger.Errorf("failed to get clone url for provider %s, error %v", payload.GitProvider, err)
		return err
	}

	gm.logger.Debugf("cloning from %s", archiveURL)
	err = gm.downloadFile(ctx, archiveURL, commitID+".zip", oauth)
	if err != nil {
		gm.logger.Errorf("failed to download file %v", err)
		return err
	}

	if err = gm.initGit(ctx, payload, oauth); err != nil {
		gm.logger.Errorf("failed to initialize git, error %v", err)
		return err
	}

	return nil
}

// downloadFile clones the archive from github and extracts the file if it is a zip file.
func (gm *gitManager) downloadFile(ctx context.Context, archiveURL, fileName string, oauth *core.Oauth) error {
	header := getHeaderMap(oauth)
	respBody, stausCode, err := gm.request.MakeAPIRequest(ctx, http.MethodGet, archiveURL, nil, nil, header)
	if err != nil {
		return err
	}
	if stausCode >= http.StatusMultipleChoices {
		return fmt.Errorf("received non 200 status code [%d]", stausCode)
	}

	err = gm.copyAndExtractFile(ctx, respBody, fileName)
	if err != nil {
		gm.logger.Errorf("failed to copy file %v", err)
		return err
	}
	return nil
}

// copyAndExtractFile copies the content of http response directly to the local storage
// and extracts the file if it is a zip file.
func (gm *gitManager) copyAndExtractFile(ctx context.Context, respBody []byte, path string) error {
	out, err := os.Create(path)
	if err != nil {
		gm.logger.Errorf("failed to create file err %v", err)
		return err
	}
	_, err = out.Write(respBody)
	if err != nil {
		gm.logger.Errorf("failed to write to file %v", err)
		out.Close()
		return err
	}
	out.Close()

	// if zip file, then unarchive the file in same path
	if filepath.Ext(path) == ".zip" {
		zip := archiver.NewZip()
		zip.OverwriteExisting = true
		if err = zip.Unarchive(path, fmt.Sprintf("%s/clonedir", filepath.Dir(path))); err != nil {
			gm.logger.Errorf("failed to unarchive file %v", err)
			return err
		}
	}

	commands := []string{
		fmt.Sprintf("mkdir %s", global.RepoDir),
		fmt.Sprintf("mv %s/clonedir/*/* %s", filepath.Dir(path), global.RepoDir),
	}

	err = gm.execManager.ExecuteInternalCommands(ctx, core.RenameCloneFile, commands, filepath.Dir(path), nil, nil)
	if err != nil {
		return err
	}

	return err
}

func (gm *gitManager) initGit(ctx context.Context, payload *core.Payload, oauth *core.Oauth) error {
	branch := payload.BranchName
	repoLink := payload.RepoLink
	if payload.GitProvider == core.Bitbucket && payload.ForkSlug != "" {
		repoLink = strings.Replace(repoLink, payload.RepoSlug, payload.ForkSlug, -1)
	}

	repoURL, perr := url.Parse(repoLink)
	if perr != nil {
		return perr
	}

	if oauth.Type == core.Basic {
		decodedToken, err := base64.StdEncoding.DecodeString(oauth.AccessToken)
		if err != nil {
			gm.logger.Errorf("Failed to decode basic oauth token for RepoID %s: %s", payload.RepoID, err)
			return err
		}

		creds := strings.Split(string(decodedToken), ":")
		repoURL.User = url.UserPassword(creds[0], creds[1])
	} else {
		repoURL.User = url.UserPassword("x-token-auth", oauth.AccessToken)
		if payload.GitProvider == core.GitLab {
			repoURL.User = url.UserPassword("oauth2", oauth.AccessToken)
		}
	}

	urlWithToken := repoURL.String()
	commands := []string{
		"git init",
		fmt.Sprintf("git remote add origin %s.git", repoLink),
		fmt.Sprintf("git config --global url.%s.InsteadOf %s", urlWithToken, repoLink),
		fmt.Sprintf("git fetch --depth=1 origin +%s:refs/remotes/origin/%s", payload.BuildTargetCommit, branch),
		fmt.Sprintf("git config --global --remove-section url.%s", urlWithToken),
		fmt.Sprintf("git checkout --progress --force -B %s refs/remotes/origin/%s", branch, branch),
	}
	if err := gm.execManager.ExecuteInternalCommands(ctx, core.InitGit, commands, global.RepoDir, nil, nil); err != nil {
		return err
	}
	return nil
}

func (gm *gitManager) DownloadFileByCommit(ctx context.Context, gitProvider, repoSlug,
	commitID, filePath string, oauth *core.Oauth) (string, error) {
	downloadURL, err := urlmanager.GetFileDownloadURL(gitProvider, commitID, repoSlug, filePath)
	if err != nil {
		return "", err
	}
	header := getHeaderMap(oauth)
	respBody, stausCode, err := gm.request.MakeAPIRequest(ctx, http.MethodGet, downloadURL, nil, nil, header)
	if err != nil {
		return "", err
	}
	if stausCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("received non 200 status code [%d]", stausCode)
	}
	path := utils.GenerateUUID() + ".yml"
	out, err := os.Create(path)
	if err != nil {
		gm.logger.Errorf("failed to create file err %v", err)
		return "", err
	}
	_, err = out.Write(respBody)
	if err != nil {
		gm.logger.Errorf("failed to copy file %v", err)
		out.Close()
		return "", err
	}
	out.Close()
	return path, nil
}

func getHeaderMap(oauth *core.Oauth) map[string]string {
	header := map[string]string{}

	header[authorization] = fmt.Sprintf("%s %s", oauth.Type, oauth.AccessToken)
	return header
}
