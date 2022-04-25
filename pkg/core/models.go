// Package core is the backbone of the tunnel client,
// it defines  the tunnel lifecycle and allows attaching hooks for functionality
// as plugins.
package core

import (
	"net/http"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

// ExecutionID type
type ExecutionID string

// Tier type of synapse
type Tier string

// CommandType defines type of command
type CommandType string

// ContainerType defines types of container
type ContainerType string

// TaskTier values.
const (
	Internal Tier = "internal"
	XSmall   Tier = "xsmall"
	Small    Tier = "small"
	Medium   Tier = "medium"
	Large    Tier = "large"
	XLarge   Tier = "xlarge"
)

// PostMergeStrategyName type
type PostMergeStrategyName string

// All  const of type PostMergeStrategyName
const (
	AfterNCommitStrategy PostMergeStrategyName = "after_n_commits"
)

// SplitMode is the mode for splitting tests
type SplitMode string

// list of support test splitting modes
const (
	FileSplit SplitMode = "file"
	TestSplit SplitMode = "test"
)

// Types of Command string
const (
	PreRun         CommandType = "prerun"
	PostRun        CommandType = "postrun"
	InstallRunners CommandType = "installrunners"
	Execution      CommandType = "execution"
	Discovery      CommandType = "discovery"
	Zstd           CommandType = "zstd"
	CoverageMerge  CommandType = "coveragemerge"
	InstallNodeVer CommandType = "installnodeversion"
	InitGit        CommandType = "initgit"
)

// Types of containers
const (
	CacheContainer   ContainerType = "cache"
	LogsContainer    ContainerType = "logs"
	PayloadContainer ContainerType = "container-payload"
)

// EventType represents the webhook event
type EventType string

const (
	// EventPush represents the push event.
	EventPush EventType = "push"
	// EventPullRequest represents the pull request event.
	EventPullRequest EventType = "pull-request"
)

// CommitChangeList defines  information related to commits
type CommitChangeList struct {
	Sha      string   `json:"Sha"`
	Link     string   `json:"Link"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
	Message  string   `json:"message"`
}

// Payload defines structure of payload
type Payload struct {
	RepoSlug                   string             `json:"repo_slug"`
	ForkSlug                   string             `json:"fork_slug"`
	RepoLink                   string             `json:"repo_link"`
	BuildTargetCommit          string             `json:"build_target_commit"`
	BuildBaseCommit            string             `json:"build_base_commit"`
	TaskID                     string             `json:"task_id"`
	BranchName                 string             `json:"branch_name"`
	BuildID                    string             `json:"build_id"`
	RepoID                     string             `json:"repo_id"`
	OrgID                      string             `json:"org_id"`
	GitProvider                string             `json:"git_provider"`
	PrivateRepo                bool               `json:"private_repo"`
	EventType                  EventType          `json:"event_type"`
	Diff                       string             `json:"diff_url"`
	PullRequestNumber          int                `json:"pull_request_number"`
	Commits                    []CommitChangeList `json:"commits"`
	TasFileName                string             `json:"tas_file_name"`
	Locators                   string             `json:"locators"`
	LocatorAddress             string             `json:"locator_address"`
	ParentCommitCoverageExists bool               `json:"parent_commit_coverage_exists"`
	LicenseTier                Tier               `json:"license_tier"`
	CollectCoverage            bool               `json:"collect_coverage"`
}

// Pipeline defines all attributes of Pipeline
type Pipeline struct {
	Cfg                  *config.NucleusConfig
	Payload              *Payload
	Logger               lumber.Logger
	PayloadManager       PayloadManager
	TASConfigManager     TASConfigManager
	GitManager           GitManager
	ExecutionManager     ExecutionManager
	DiffManager          DiffManager
	CacheStore           CacheStore
	TestDiscoveryService TestDiscoveryService
	BlockTestService     BlockTestService
	TestExecutionService TestExecutionService
	CoverageService      CoverageService
	TestStats            TestStats
	Task                 Task
	SecretParser         SecretParser
	HTTPClient           http.Client
}

type DiscoveryResult struct {
	Tests           []TestPayload      `json:"tests"`
	ImpactedTests   []string           `json:"impactedTests"`
	TestSuites      []TestSuitePayload `json:"testSuites"`
	ExecuteAllTests bool               `json:"executeAllTests"`
	Parallelism     int                `json:"parallelism"`
	SplitMode       SplitMode          `json:"splitMode"`
	RepoID          string             `json:"repoID"`
	BuildID         string             `json:"buildID"`
	CommitID        string             `json:"commitID"`
	TaskID          string             `json:"taskID"`
	OrgID           string             `json:"orgID"`
	Branch          string             `json:"branch"`
	Tier            Tier               `json:"tier"`
}

// ExecutionResult represents the request body for test and test suite execution
type ExecutionResult struct {
	TestPayload      []TestPayload      `json:"testResults"`
	TestSuitePayload []TestSuitePayload `json:"testSuiteResults"`
}

// ExecutionResults represents collection of execution results
type ExecutionResults struct {
	TaskID   string            `json:"taskID"`
	BuildID  string            `json:"buildID"`
	RepoID   string            `json:"repoID"`
	OrgID    string            `json:"orgID"`
	CommitID string            `json:"commitID"`
	Results  []ExecutionResult `json:"results"`
}

// TestPayload represents the request body for test execution
type TestPayload struct {
	TestID          string             `json:"testID"`
	Detail          string             `json:"_detail"`
	SuiteID         string             `json:"suiteID"`
	Suites          []string           `json:"_suites"`
	Title           string             `json:"title"`
	FullTitle       string             `json:"fullTitle"`
	Name            string             `json:"name"`
	Duration        int                `json:"duration"`
	FilePath        string             `json:"file"`
	Line            string             `json:"line"`
	Col             string             `json:"col"`
	CurrentRetry    int                `json:"currentRetry"`
	Status          string             `json:"status"`
	CommitID        string             `json:"commitID"`
	DAG             []string           `json:"dependsOn"`
	Filelocator     string             `json:"locator"`
	BlocklistSource string             `json:"blocklistSource"`
	Blocklisted     bool               `json:"blocklist"`
	StartTime       time.Time          `json:"start_time"`
	EndTime         time.Time          `json:"end_time"`
	Stats           []TestProcessStats `json:"stats"`
}

// TestSuitePayload represents the request body for test suite execution
type TestSuitePayload struct {
	SuiteID         string             `json:"suiteID"`
	SuiteName       string             `json:"suiteName"`
	ParentSuiteID   string             `json:"parentSuiteID"`
	BlocklistSource string             `json:"blocklistSource"`
	Blocklisted     bool               `json:"blocklist"`
	StartTime       time.Time          `json:"start_time"`
	EndTime         time.Time          `json:"end_time"`
	Duration        int                `json:"duration"`
	Status          string             `json:"status"`
	Stats           []TestProcessStats `json:"stats"`
	TotalTests      int                `json:"totalTests"`
}

// TestProcessStats process stats associated with each test
type TestProcessStats struct {
	Memory     uint64    `json:"memory_consumed,omitempty"`
	CPU        float64   `json:"cpu_percentage,omitempty"`
	Storage    uint64    `json:"storage,omitempty"`
	RecordTime time.Time `json:"record_time"`
}

// Status represents the task status
type Status string

// Const related to task status
const (
	Initiating Status = "initiating"
	Running    Status = "running"
	Failed     Status = "failed"
	Aborted    Status = "aborted"
	Passed     Status = "passed"
	Error      Status = "error"
)

// TaskPayload repersent task response given by nucleus to neuron
type TaskPayload struct {
	TaskID      string    `json:"task_id"`
	Status      Status    `json:"status"`
	RepoSlug    string    `json:"repo_slug"`
	RepoLink    string    `json:"repo_link"`
	RepoID      string    `json:"repo_id"`
	OrgID       string    `json:"org_id"`
	GitProvider string    `json:"git_provider"`
	CommitID    string    `json:"commit_id,omitempty"`
	BuildID     string    `json:"build_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Remark      string    `json:"remark,omitempty"`
	Type        TaskType  `json:"type"`
}

// CoverageManifest for post processing coverage job
type CoverageManifest struct {
	Removedfiles      []string           `json:"removed_files"`
	AllFilesExecuted  bool               `json:"all_files_executed"`
	CoverageThreshold *CoverageThreshold `json:"coverage_threshold,omitempty"`
}

const (
	// FileAdded file added in commit
	FileAdded int = iota + 1
	// FileRemoved file removed in commit
	FileRemoved
	// FileModified file modified in commit
	FileModified
)

const (
	// GitHub as git provider
	GitHub string = "github"
	// GitLab as git provider
	GitLab string = "gitlab"
	// Bitbucket as git provider
	Bitbucket string = "bitbucket"
)

type TokenType string

const (
	// Bearer as token type
	Bearer TokenType = "Bearer"
	// Basic as token type
	Basic TokenType = "Basic"
)

// Oauth represents the sructure of Oauth
type Oauth struct {
	AccessToken  string    `json:"access_token"`
	Expiry       time.Time `json:"expiry"`
	RefreshToken string    `json:"refresh_token"`
	Type         TokenType `json:"token_type,omitempty"`
}

// TASConfig represents the .tas.yml file
type TASConfig struct {
	SmartRun          bool               `yaml:"smartRun"`
	Framework         string             `yaml:"framework" validate:"required,oneof=jest mocha jasmine"`
	Blocklist         []string           `yaml:"blocklist"`
	Postmerge         *Merge             `yaml:"postMerge" validate:"omitempty"`
	Premerge          *Merge             `yaml:"preMerge" validate:"omitempty"`
	Cache             *Cache             `yaml:"cache" validate:"omitempty"`
	Prerun            *Run               `yaml:"preRun" validate:"omitempty"`
	Postrun           *Run               `yaml:"postRun" validate:"omitempty"`
	Parallelism       int                `yaml:"parallelism"`
	SplitMode         SplitMode          `yaml:"splitMode" validate:"oneof=test file"`
	SkipCache         bool               `yaml:"skipCache"`
	ConfigFile        string             `yaml:"configFile" validate:"omitempty"`
	CoverageThreshold *CoverageThreshold `yaml:"coverageThreshold" validate:"omitempty"`
	Tier              Tier               `yaml:"tier" validate:"oneof=xsmall small medium large xlarge"`
	NodeVersion       string             `yaml:"nodeVersion" validate:"omitempty,semver"`
	ContainerImage    string             `yaml:"containerImage"`
}

// CoverageThreshold reprents the code coverage threshold
type CoverageThreshold struct {
	Branches   float64 `yaml:"branches" json:"branches" validate:"number,min=0,max=100"`
	Lines      float64 `yaml:"lines" json:"lines" validate:"number,min=0,max=100"`
	Functions  float64 `yaml:"functions" json:"functions" validate:"number,min=0,max=100"`
	Statements float64 `yaml:"statements" json:"statements" validate:"number,min=0,max=100"`
	PerFile    bool    `yaml:"perFile" json:"perFile"`
}

// Cache represents the user's cached directories
type Cache struct {
	Key   string   `yaml:"key" validate:"required"`
	Paths []string `yaml:"paths" validate:"required"`
}

// Modifier defines struct for modifier
type Modifier struct {
	Type   string
	Config string
	Cli    string
}

// Run represents  pre and post runs
type Run struct {
	Commands []string          `yaml:"command" validate:"omitempty,gt=0"`
	EnvMap   map[string]string `yaml:"env" validate:"omitempty,gt=0"`
}

// Merge represents pre and post merge
type Merge struct {
	Patterns []string          `yaml:"pattern" validate:"required,gt=0"`
	EnvMap   map[string]string `yaml:"env" validate:"omitempty,gt=0"`
}

// Stability defines struct for stability
type Stability struct {
	ConsecutiveRuns int `yaml:"consecutive_runs"`
}

// TaskType specifies the type of a Task
type TaskType string

// Task Type values.
const (
	DiscoveryTask TaskType = "discover"
	ExecutionTask TaskType = "execute"
	FlakyTask     TaskType = "flaky"
)

// TestStatus stores tests status
type TestStatus string

const (
	Blocklisted TestStatus = "blocklisted"
	Quarantined TestStatus = "quarantined"
)
