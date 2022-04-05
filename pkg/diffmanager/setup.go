// Package diffmanager is used for cloning repo
package diffmanager

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/urlmanager"
)

//TODO: add logger

type diffManager struct {
	cfg    *config.NucleusConfig
	client http.Client
	logger lumber.Logger
}

type gitLabDiffList struct {
	CommitDiff []gitLabDiff `json:"diffs"`
	PRDiff     []gitLabDiff `json:"changes"`
}
type gitLabDiff struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

// NewDiffManager Instantiate DiffManager
func NewDiffManager(cfg *config.NucleusConfig, logger lumber.Logger) *diffManager {
	return &diffManager{
		cfg:    cfg,
		logger: logger,
		client: http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

// Updated values with "or" operation
func (dm *diffManager) updateWithOr(m map[string]int, key string, value int) {
	if _, exists := m[key]; !exists {
		m[key] = 0
	}
	m[key] = m[key] | value
}

func (dm *diffManager) getCommitDiff(gitprovider, repoURL string, oauth *core.Oauth, baseCommit, targetCommit, forkSlug string) ([]byte, error) {
	if baseCommit == "" {
		dm.logger.Debugf("basecommit is empty for gitprovider %v error %v", gitprovider, errs.ErrGitDiffNotFound)
		return nil, errs.ErrGitDiffNotFound
	}
	url, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}

	apiURLString, err := urlmanager.GetCommitDiffURL(gitprovider, url.Path, baseCommit, targetCommit, forkSlug)
	if err != nil {
		dm.logger.Errorf("failed to get api url for gitprovider: %v error: %v", gitprovider, err)
		return nil, err
	}
	apiURL, err := url.Parse(apiURLString)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, err
	}
	if oauth.AccessToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("%s %s", oauth.Type, oauth.AccessToken))
	}
	req.Header.Add("Accept", "application/vnd.github.v3.diff")
	resp, err := dm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//TODO: Handle initial commit case
	if resp.StatusCode != http.StatusOK {
		return nil, errs.ErrGitDiffNotFound
	}
	return ioutil.ReadAll(resp.Body)
}

func (dm *diffManager) getPRDiff(gitprovider, repoURL string, prNumber int, oauth *core.Oauth) ([]byte, error) {
	parsedUrl, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}
	diffURL, err := urlmanager.GetPullRequestDiffURL(gitprovider, parsedUrl.Path, prNumber)
	if err != nil {
		dm.logger.Errorf("failed to get diff url error: %v", err)
		return nil, err
	}
	changeListURL, err := url.Parse(diffURL)
	if err != nil {
		dm.logger.Errorf("failed to get changelist url error: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, changeListURL.String(), nil)
	if err != nil {
		dm.logger.Errorf("failed to create http request for changelist url error: %v", err)
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", oauth.Type, oauth.AccessToken))
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := dm.client.Do(req)

	if err != nil {
		dm.logger.Errorf("failed to get changedlist url api error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("non 200 response")
	}

	return ioutil.ReadAll(resp.Body)

}

func (dm *diffManager) parseDiff(diff string) map[string]int {
	m := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "--- a/") {
			// removed
			dm.updateWithOr(m, line[6:], core.FileRemoved)
		} else if strings.HasPrefix(line, "+++ b/") {
			// added or updated
			dm.updateWithOr(m, line[6:], core.FileAdded)
		}
	}
	return m
}

func (dm *diffManager) parseGitLabDiff(eventType core.EventType, diff []byte) (map[string]int, error) {
	m := make(map[string]int)
	var diffList gitLabDiffList
	err := json.Unmarshal(diff, &diffList)
	if err != nil {
		dm.logger.Errorf("failed to unmarshall diff %v error %v", string(diff), err)
		return nil, err
	}
	diffs := diffList.PRDiff
	if eventType == core.EventPush {
		diffs = diffList.CommitDiff
	}
	for _, diff := range diffs {
		if diff.DeletedFile {
			// removed
			dm.updateWithOr(m, diff.OldPath, core.FileRemoved)
		} else if diff.NewFile {
			// added
			dm.updateWithOr(m, diff.NewPath, core.FileAdded)
		} else {
			// updated
			dm.updateWithOr(m, diff.NewPath, core.FileModified)
		}
	}
	return m, nil
}

func (dm *diffManager) parseGitDiff(gitprovider string, eventType core.EventType, diff []byte) (map[string]int, error) {
	switch gitprovider {
	case core.GitHub, core.Bitbucket:
		return dm.parseDiff(string(diff)), nil
	case core.GitLab:
		return dm.parseGitLabDiff(eventType, diff)
	default:
		return nil, errs.ErrUnsupportedGitProvider
	}
}

// GetChangedFiles Figure out changed files
func (dm *diffManager) GetChangedFiles(ctx context.Context, payload *core.Payload, oauth *core.Oauth) (map[string]int, error) {
	// map to store file and type of change (added, removed, modified)
	var m map[string]int

	var diff []byte
	var err error
	if payload.EventType == core.EventPullRequest {
		diff, err = dm.getPRDiff(payload.GitProvider, payload.RepoLink, payload.PullRequestNumber, oauth)
		if err != nil {
			dm.logger.Errorf("failed to parse pr diff for gitprovider: %s error: %v", payload.GitProvider, err)
			return nil, err
		}
	} else {
		diff, err = dm.getCommitDiff(payload.GitProvider, payload.RepoLink, oauth, payload.BuildBaseCommit, payload.BuildTargetCommit, payload.ForkSlug)
		if err != nil {
			dm.logger.Errorf("failed to get commit diff for gitprovider: %s error: %v", payload.GitProvider, err)
			return nil, err
		}
	}

	m, err = dm.parseGitDiff(payload.GitProvider, payload.EventType, diff)
	if err != nil {
		dm.logger.Errorf("failed to parse gitdiff for gitprovider: %s error: %v", payload.GitProvider, err)
		return nil, err
	}
	return m, nil
}
