package urlmanager

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
)

// GetCloneURL returns repo clone url for given git provider
func GetCloneURL(gitprovider, repoLink, repo, commitID, forkSlug, repoSlug string) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	}
	switch gitprovider {
	case core.GitHub:
		return fmt.Sprintf("%s/%s/zipball/%s", global.APIHostURLMap[gitprovider], repoSlug, commitID), nil
	case core.GitLab:
		return fmt.Sprintf("%s/-/archive/%s/%s-%s.zip", repoLink, commitID, repo, commitID), nil

	case core.Bitbucket:
		if forkSlug != "" {
			forkLink := strings.Replace(repoLink, repoSlug, forkSlug, -1)
			return fmt.Sprintf("%s/get/%s.zip", forkLink, commitID), nil
		}

		return fmt.Sprintf("%s/get/%s.zip", repoLink, commitID), nil

	default:
		return "", errs.ErrUnsupportedGitProvider
	}
}

// GetCommitDiffURL returns commit diff url for given git provider
func GetCommitDiffURL(gitprovider, path, baseCommit, targetCommit, forkSlug string) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	}
	switch gitprovider {
	case core.GitHub:
		return fmt.Sprintf("%s%s/compare/%s...%s", global.APIHostURLMap[gitprovider], path, baseCommit, targetCommit), nil

	case core.GitLab:
		encodedPath := url.QueryEscape(path[1:])
		return fmt.Sprintf("%s/%s/repository/compare?from=%s&to=%s",
			global.APIHostURLMap[gitprovider], encodedPath, baseCommit, targetCommit), nil

	case core.Bitbucket:
		if forkSlug != "" {
			return fmt.Sprintf("%s/repositories%s/diff/%s..%s",
				global.APIHostURLMap[gitprovider], path, fmt.Sprintf("%s:%s", forkSlug, targetCommit), baseCommit), nil
		}
		return fmt.Sprintf("%s/repositories%s/diff/%s..%s", global.APIHostURLMap[gitprovider], path, targetCommit, baseCommit), nil

	default:
		return "", errs.ErrUnsupportedGitProvider
	}
}

// GetPullRequestDiffURL returns PR Diff url for given git provider
func GetPullRequestDiffURL(gitprovider, path string, prNumber int) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	}
	switch gitprovider {
	case core.GitHub:
		return fmt.Sprintf("%s%s/pulls/%d", global.APIHostURLMap[gitprovider], path, prNumber), nil

	case core.GitLab:
		encodedPath := url.QueryEscape(path[1:])
		return fmt.Sprintf("%s/%s/merge_requests/%d/changes", global.APIHostURLMap[gitprovider], encodedPath, prNumber), nil

	case core.Bitbucket:
		return fmt.Sprintf("%s/repositories%s/pullrequests/%d/diff", global.APIHostURLMap[gitprovider], path, prNumber), nil

	default:
		return "", errs.ErrUnsupportedGitProvider
	}
}

// GetFileDownloadURL returns download URL for file in repo
func GetFileDownloadURL(gitprovider, commitID, repoSlug, filePath string) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	}
	switch gitprovider {
	case core.GitHub:
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repoSlug, commitID, filePath), nil
	case core.GitLab:
		repoSlug = url.PathEscape(repoSlug)
		filePath = url.PathEscape(filePath)
		return fmt.Sprintf("%s/%s/repository/files/%s/raw?ref=%s", global.APIHostURLMap[gitprovider], repoSlug, filePath, commitID), nil
	case core.Bitbucket:
		// TODO: check for fork PR
		return fmt.Sprintf("%s/repositories/%s/src/%s/%s", global.APIHostURLMap[gitprovider], repoSlug, commitID, filePath), nil
	default:
		return "", nil
	}
}
