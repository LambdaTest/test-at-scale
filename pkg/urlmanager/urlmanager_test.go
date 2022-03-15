package urlmanager

import (
	"net/url"
	"testing"

	"github.com/LambdaTest/synapse/pkg/global"
)

func TestGetCloneURL(t *testing.T) {
	type args struct {
		gitprovider string
		repoLink    string
		repo        string
		commitID    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"For github as git provider", args{"github", "https://github.com/nexe", "nexe", "abc"}, "https://github.com/nexe/archive/abc.zip", false},
		{"For non-github and gitlab as git provider", args{"gittest", "https://github.com/nexe", "nexe", "abc"}, "", true},
		{"For gitlab as git provider", args{"gitlab", "https://gitlab.com/nexe", "nexe", "abc"}, "https://gitlab.com/nexe/-/archive/abc/nexe-abc.zip", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCloneURL(tt.args.gitprovider, tt.args.repoLink, tt.args.repo, tt.args.commitID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCloneURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCloneURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCommitDiffURL(t *testing.T) {
	type args struct {
		gitprovider  string
		path         string
		baseCommit   string
		targetCommit string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"For github as git provider", args{"github", "/tests/nexe", "abc", "xyz"}, "https://api.github.com/repos/tests/nexe/compare/abc...xyz", false},
		{"For non-github and gitlab as git provider", args{"gittest", "tests/nexe", "abc", "xyz"}, "", true},
		{"For gitlab as git provider", args{"gitlab", "/tests/nexe", "abc", "xyz"}, global.APIHostURLMap["gitlab"] + "/" + url.QueryEscape("tests/nexe") + "/repository/compare?from=abc&to=xyz", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCommitDiffURL(tt.args.gitprovider, tt.args.path, tt.args.baseCommit, tt.args.targetCommit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitDiffURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCommitDiffURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPullRequestDiffURL(t *testing.T) {
	type args struct {
		gitprovider string
		path        string
		prNumber    int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"For github as git provider", args{"github", "/tests/nexe", 2}, "https://api.github.com/repos/tests/nexe/pulls/2", false},
		{"For non-github and gitlab as git provider", args{"gittest", "tests/nexe", 2}, "", true},
		{"For gitlab as git provider", args{"gitlab", "/tests/nexe", 2}, global.APIHostURLMap["gitlab"] + "/" + url.QueryEscape("tests/nexe") + "/merge_requests/2/changes", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPullRequestDiffURL(tt.args.gitprovider, tt.args.path, tt.args.prNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPullRequestDiffURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetPullRequestDiffURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
