package coverage

import "encoding/json"

type parentCommitCoverage struct {
	Bloblink     string `json:"blob_link"`
	ParentCommit string `json:"parent_commit"`
}

type coverageData struct {
	BuildID       string          `json:"build_id"`
	RepoID        string          `json:"repo_id"`
	CommitID      string          `json:"commit_id"`
	BlobLink      string          `json:"blob_link"`
	TotalCoverage json.RawMessage `json:"total_coverage"`
}
