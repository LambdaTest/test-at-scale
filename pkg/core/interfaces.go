package core

import (
	"context"
	"io"
)

// PayloadManager defines operations for payload
type PayloadManager interface {
	// ValidatePayload validates the nucleus payload
	ValidatePayload(ctx context.Context, payload *Payload) error
	// FetchPayload used for fetching the payload used for running nucleus
	FetchPayload(ctx context.Context, payloadAddress string) (*Payload, error)
}

// TASConfigManager defines operations for tas config
type TASConfigManager interface {
	// LoadAndValidate loads and returns the tas config
	LoadAndValidate(ctx context.Context, version int, path string, eventType EventType, licenseTier Tier,
		tasFilePathInRepo string) (interface{}, error)

	// GetVersion returns TAS yml version
	GetVersion(path string) (int, error)

	// GetTasConfigFilePath returns file path of tas config
	GetTasConfigFilePath(payload *Payload) (string, error)
}

// GitManager manages the cloning of git repositories
type GitManager interface {
	// Clone repository from TAS config
	Clone(ctx context.Context, payload *Payload, oauth *Oauth) error
	// DownloadFileByCommit download file from repo for given commit
	DownloadFileByCommit(ctx context.Context, gitProvider, repoSlug, commitID, filePath string, oauth *Oauth) (string, error)
}

// DiffManager manages the diff findings for the given payload
type DiffManager interface {
	GetChangedFiles(ctx context.Context, payload *Payload, oauth *Oauth) (map[string]int, error)
}

// TestDiscoveryService services discovery of tests
type TestDiscoveryService interface {
	// Discover executes the test discovery scripts.
	Discover(ctx context.Context, args *DiscoveyArgs) (*DiscoveryResult, error)

	// SendResult sends discovery result to TAS server
	SendResult(ctx context.Context, testDiscoveryResult *DiscoveryResult) error
}

// BlockTestService is used for fetching blocklisted tests
type BlockTestService interface {
	GetBlockTests(ctx context.Context, blocklistYAML []string, branch string) error
}

// TestExecutionService services execution of tests
type TestExecutionService interface {
	// Run executes the test execution scripts
	Run(ctx context.Context, testExecutionArgs *TestExecutionArgs) (results *ExecutionResults, err error)
	// SendResults sends the test execution results to the TAS server.
	SendResults(ctx context.Context, payload *ExecutionResults) (resp *TestReportResponsePayload, err error)
}

// CoverageService services coverage of tests
type CoverageService interface {
	MergeAndUpload(ctx context.Context, payload *Payload) error
}

// TestStats is used for servicing stat collection
type TestStats interface {
	CaptureTestStats(pid int32, collectStats bool) error
}

// Task is a service to update task status at neuron
type Task interface {
	// UpdateStatus updates status of the task
	UpdateStatus(ctx context.Context, payload *TaskPayload) error
}

// NotifMessage  defines struct for notification message
type NotifMessage struct {
	Type   string
	Value  string
	Status string
	Error  string
}

// AzureClient defines operation for working with azure store
type AzureClient interface {
	FindUsingSASUrl(ctx context.Context, sasURL string) (io.ReadCloser, error)
	Find(ctx context.Context, path string) (io.ReadCloser, error)
	Create(ctx context.Context, path string, reader io.Reader, mimeType string) (string, error)
	CreateUsingSASURL(ctx context.Context, sasURL string, reader io.Reader, mimeType string) (string, error)
	GetSASURL(ctx context.Context, purpose SASURLPurpose, query map[string]interface{}) (string, error)
	Exists(ctx context.Context, path string) (bool, error)
}

// ZstdCompressor performs zstd compression and decompression
type ZstdCompressor interface {
	Compress(ctx context.Context, compressedFileName string, preservePath bool, workingDirectory string, filesToCompress ...string) error
	Decompress(ctx context.Context, filePath string, preservePath bool, workingDirectory string) error
}

// CacheStore defines operation for working with the cache
//go:generate mockery  --name  CacheStore  --keeptree  --output  ../mocks/CacheStore.go
type CacheStore interface {
	// Download downloads cache present at cacheKey
	Download(ctx context.Context, cacheKey string) error
	// Upload creates, compresses and uploads cache at cacheKey
	Upload(ctx context.Context, cacheKey string, itemsToCompress ...string) error
	// CacheWorkspace caches the workspace onto a mounted volume
	CacheWorkspace(ctx context.Context, subModule string) error
	// ExtractWorkspace extracts the workspace cache from mounted volume
	ExtractWorkspace(ctx context.Context, subModule string) error
}

// SecretParser defines operation for parsing the vault secrets in given path
type SecretParser interface {
	// GetOauthSecret parses the oauth secret for given path
	GetOauthSecret(filepath string) (*Oauth, error)
	// GetRepoSecret parses the repo secret for given path
	GetRepoSecret(string) (map[string]string, error)
	// SubstituteSecret replace secret placeholders with their respective values
	SubstituteSecret(command string, secretData map[string]string) (string, error)
	// Expired reports whether the token is expired.
	Expired(token *Oauth) bool
}

// ExecutionManager has responsibility for executing the preRun, postRun and internal commands
type ExecutionManager interface {
	// ExecuteUserCommands executes the preRun or postRun commands given by user in his yaml.
	ExecuteUserCommands(ctx context.Context,
		commandType CommandType,
		payload *Payload,
		runConfig *Run,
		secretData map[string]string,
		logwriter LogWriterStrategy,
		cwd string) error

	// ExecuteInternalCommands executes the commands like installing runners and test discovery.
	ExecuteInternalCommands(ctx context.Context,
		commandType CommandType,
		commands []string,
		cwd string, envMap,
		secretData map[string]string) error
	// GetEnvVariables get the environment variables from the env map given by user.
	GetEnvVariables(envMap, secretData map[string]string) ([]string, error)
}

// Requests is a util interface for making API Requests
type Requests interface {
	// MakeAPIRequest makes an HTTP request with auth
	MakeAPIRequest(ctx context.Context, httpMethod, endpoint string, body []byte, params map[string]interface{},
		headers map[string]string) (rawbody []byte, statusCode int, err error)
}

// ListSubModuleService will sends the submodule count to TAS server
type ListSubModuleService interface {
	// Send sends count of submodules to TAS server
	Send(ctx context.Context, buildID string, totalSubmodule int) error
}

// Driver has the responsibility to run discovery and test execution
type Driver interface {
	// RunDiscovery runs the test discovery
	RunDiscovery(ctx context.Context, payload *Payload,
		taskPayload *TaskPayload, oauth *Oauth, coverageDir string, secretMap map[string]string) error
	// RunExecution runs the test execution
	RunExecution(ctx context.Context, payload *Payload,
		taskPayload *TaskPayload, oauth *Oauth, coverageDir string, secretMap map[string]string) error
}

// LogWriterStrategy interface is used to tag all log writing strategy
type LogWriterStrategy interface {
	// Write reads data from io.Reader and write it to various data stream
	Write(ctx context.Context, reader io.Reader) <-chan error
}

// Builder builds the driver for given tas yml version
type Builder interface {
	// GetDriver returns driver for use
	GetDriver(version int, ymlFilePath string) (Driver, error)
}
