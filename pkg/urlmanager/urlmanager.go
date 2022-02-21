package urlmanager

import (
	"fmt"
	"net/url"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/global"
)

// GetDownloadURL returns file download url for given git provider
func GetDownloadURL(gitprovider, repoSlug, commitID, fileName string) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	} else {
		switch gitprovider {
		case core.GitHub:
			return fmt.Sprintf("%s/%s/%s/%s", global.RawContentURLMap[gitprovider], repoSlug, commitID, fileName), nil

		case core.GitLab:
			encodedPath := url.QueryEscape(repoSlug)
			return fmt.Sprintf("%s/%s/repository/files/%s/raw?ref=%s", global.APIHostURLMap[gitprovider], encodedPath, fileName, commitID), nil
		default:
			return "", errs.ErrUnsupportedGitProvider
		}
	}
}

// GetCloneURL returns repo clone url for given git provider
func GetCloneURL(gitprovider, repoLink, repo, commitID string) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	} else {
		switch gitprovider {
		case core.GitHub:
			return fmt.Sprintf("%s/archive/%s.zip", repoLink, commitID), nil
		case core.GitLab:
			return fmt.Sprintf("%s/-/archive/%s/%s-%s.zip", repoLink, commitID, repo, commitID), nil
		default:
			return "", errs.ErrUnsupportedGitProvider
		}
	}
}

// GetCommitDiffURL returns commit diff url for given git provider
func GetCommitDiffURL(gitprovider, path, baseCommit, targetCommit string) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	} else {
		switch gitprovider {
		case core.GitHub:
			return fmt.Sprintf("%s%s/compare/%s...%s", global.APIHostURLMap[gitprovider], path, baseCommit, targetCommit), nil

		case core.GitLab:
			encodedPath := url.QueryEscape(path[1:])
			return fmt.Sprintf("%s/%s/repository/compare?from=%s&to=%s", global.APIHostURLMap[gitprovider], encodedPath, baseCommit, targetCommit), nil

		default:
			return "", errs.ErrUnsupportedGitProvider
		}
	}
}

// GetPullRequestDiffURL returns PR Diff url for given git provider
func GetPullRequestDiffURL(gitprovider, path string, prNumber int) (string, error) {
	if global.TestEnv {
		return global.TestServer, nil
	} else {
		switch gitprovider {
		case core.GitHub:
			return fmt.Sprintf("%s%s/pulls/%d", global.APIHostURLMap[gitprovider], path, prNumber), nil

		case core.GitLab:
			encodedPath := url.QueryEscape(path[1:])
			return fmt.Sprintf("%s/%s/merge_requests/%d/changes", global.APIHostURLMap[gitprovider], encodedPath, prNumber), nil

		default:
			return "", errs.ErrUnsupportedGitProvider
		}
	}
}
