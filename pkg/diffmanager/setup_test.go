// Package diffmanager is used for cloning repo
package diffmanager

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/testUtils"
)

func Test_GetChanagedFiles(t *testing.T) {
	server := httptest.NewServer( // mock server
		http.FileServer(http.Dir("../../testUtils")), // mock data stored at testUtils/testdata/index.txt
	)
	defer server.Close()

	logger, err := testUtils.GetLogger()
	if err != nil {
		t.Errorf("Can't get logger, received: %s", err)
	}
	config, err := testUtils.GetConfig()
	if err != nil {
		t.Errorf("Can't get logger, received: %s", err)
	}
	dm := NewDiffManager(config, logger)

	checkPRdiff := func(t *testing.T, location, gitprovider string) {
		t.Helper()
		p, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Unable to get payload, error %v", err)
		}
		p.RepoLink = server.URL
		p.EventType = "pull-request"
		p.PullRequestNumber = 2
		p.GitProvider = gitprovider
		p.RepoLink = server.URL + location
		cloneToken := ""
		global.APIHostURLMap[p.GitProvider] = server.URL
		resp, err := dm.GetChangedFiles(context.TODO(), p, cloneToken)
		if gitprovider == "" {
			if err != errs.ErrUnsupportedGitProvider {
				t.Errorf("Expected error: %v, received error: %v", errs.ErrUnsupportedGitProvider, err)
			}
			return
		}
		if location == "/notfound/" {
			expErr := errors.New("non 200 response")
			if err.Error() != expErr.Error() {
				t.Errorf("Expected error: %s, received error: %s", expErr, err)
			}
			return
		}
		expResp := testUtils.GetGitDiff()
		if err != nil {
			t.Errorf("error in getting changed files, error %v", err.Error())
		} else if gitprovider == "github" && !reflect.DeepEqual(resp, expResp) {
			t.Errorf("Expected: %+v, received: %+v", expResp, resp)
		} else if gitprovider == "gitlab" && len(resp) != 17 {
			t.Errorf("Expected map entries: 17, received: %v, received map: %v", len(resp), resp)
		}
	}
	checkCommitDiff := func(t *testing.T, location, baseCommit, targetCommit, gitProvider string) {
		t.Helper()
		p, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Unable to get payload, error %v", err)
		}
		p.RepoLink = server.URL
		p.EventType = "push"
		p.PullRequestNumber = 2
		p.GitProvider = gitProvider
		p.RepoLink = server.URL + location
		cloneToken := ""
		p.BaseCommit = baseCommit
		p.TargetCommit = targetCommit
		global.APIHostURLMap[p.GitProvider] = server.URL
		resp, err := dm.GetChangedFiles(context.TODO(), p, cloneToken)
		if gitProvider == "gittest" {
			if resp != nil || err == nil {
				t.Errorf("Expected error: 'unsupoorted git provider', received: %v\nexpected response: nil, received: %v", err, resp)
			}
			return
		}
		if baseCommit == "" || location == "/notfound/" {
			if err != nil || resp != nil {
				t.Errorf("Received error: %v, response: %v", err, resp)
			}
			return
		}
		expResp := make(map[string]int)
		if err != nil {
			t.Errorf("error in getting changed files, error %v", err.Error())
		} else if !reflect.DeepEqual(resp, expResp) {
			t.Errorf("Expected: %+v, received: %+v", expResp, resp)
		}
	}
	checkGitlabCommitDiff := func(t *testing.T, location, baseCommit, targetCommit, gitProvider string, st int) {
		t.Helper()
		data, err := testUtils.GetGitlabCommitDiff()
		if err != nil {
			t.Errorf("Received error in getting test gitlab commit diff, error: %v", err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/testdata/repository/compare" {
				t.Errorf("Expected to request '/testdata/repository/compare', got: %v", r.URL.Path)
			}
			w.WriteHeader(st)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		}))
		defer server.Close()
		p, err := testUtils.GetPayload()
		if err != nil {
			t.Errorf("Unable to get payload, error %v", err)
		}
		p.RepoLink = server.URL
		p.EventType = "push"
		p.PullRequestNumber = 2
		p.GitProvider = gitProvider
		p.RepoLink = server.URL + location
		cloneToken := ""
		p.BaseCommit = baseCommit
		p.TargetCommit = targetCommit
		global.APIHostURLMap[p.GitProvider] = server.URL
		resp, err := dm.GetChangedFiles(context.TODO(), p, cloneToken)
		if baseCommit == "" || location == "/notfound/" {
			if err != nil || resp != nil {
				t.Errorf("Received error: %v, response: %v", err, resp)
			}
			return
		}
		if err != nil {
			t.Errorf("error in getting changed files, error %v", err.Error())
		} else if len(resp) != 202 {
			t.Errorf("Expected map length: 202, received: %v\nreceived map: %v", len(resp), resp)
		}
	}
	t.Run("TestDiffManager: PRdiff for github gitprovider", func(t *testing.T) {
		checkPRdiff(t, "/testdata", "github")
	})
	t.Run("TestDiffManager: PRdiff for gitlab gitprovider", func(t *testing.T) {
		checkPRdiff(t, "/testdata", "gitlab")
	})
	t.Run("TestDiffManager: PRdiff for unsupported gitprovider", func(t *testing.T) {
		checkPRdiff(t, "/testdata", "")
	})
	t.Run("TestDiffManager: PRdiff for non 200 status", func(t *testing.T) {
		checkPRdiff(t, "/notfound/", "github")
	})
	t.Run("TestDiffManager: Commitdiff", func(t *testing.T) {
		checkCommitDiff(t, "/testdata", "abc", "xyz", "github")
	})
	t.Run("TestDiffManager: Commitdiff", func(t *testing.T) {
		checkGitlabCommitDiff(t, "/testdata", "abc", "xyz", "gitlab", 200)
	})
	t.Run("TestDiffManager: Commitdiff for empty base commit", func(t *testing.T) {
		checkCommitDiff(t, "/tests/", "", "", "github")
	})
	t.Run("TestDiffManager: Commitdiff for non 200 status", func(t *testing.T) {
		checkCommitDiff(t, "/notfound/", "abc", "xyz", "github")
	})
	t.Run("TestDiffManager: Commitdiff for non github and gitlab", func(t *testing.T) {
		checkCommitDiff(t, "/notfound/", "abc", "xyz", "gittest")
	})
}

func Test_updateWithOr(t *testing.T) {
	check := func(t *testing.T) {
		dm := &diffManager{}
		m := make(map[string]int)
		key := "key"
		val := rand.Intn(1000)
		dm.updateWithOr(m, key, val)
		if ans, exists := m[key]; !exists || ans != val {
			t.Errorf("Expected: %v, received: %v", val, m[key])
		}
		newVal := rand.Intn(1000)
		dm.updateWithOr(m, key, newVal)
		if ans, exists := m[key]; !exists || ans != (val|newVal) {
			t.Errorf("Expected: %v, received: %v", val|newVal, m[key])
		}
	}
	t.Run("Test_updateWithOr", func(t *testing.T) {
		check(t)
	})
}
