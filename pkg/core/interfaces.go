package core

import (
	"bytes"
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
	// LoadAndValidateV1 loads and validates the TASConfig from the given path for V1 Tas YML
	LoadAndValidateV1(ctx context.Context, path string, eventType EventType, licenseTier Tier) (*TASConfig, error)

	// LoadAndValidateV2 loads and validates the TASConfig from the given path for V2 Tas YML
	LoadAndValidateV2(ctx context.Context, path string, eventType EventType, licenseTier Tier) (*TASConfigV2, error)

	// GetVersion returns TAS yml version
	GetVersion(path string) (int, error)
}

// GitManager manages the cloning of git repositories
type GitManager interface {
	// Clone repository from TAS config
	Clone(ctx context.Context, payload *Payload, oauth *Oauth) error
}

// DiffManager manages the diff findings for the given payload
type DiffManager interface {
	GetChangedFiles(ctx context.Context, payload *Payload, oauth *Oauth) (map[string]int, error)
}

// TestDiscoveryService services discovery of tests
type TestDiscoveryService interface {
	// Discover executes the test discovery scripts.
	Discover(ctx context.Context, tasConfig *TASConfig, payload *Payload, secretData map[string]string,
		diff map[string]int, diffExists bool) error
	// Discoverv executes the test discovery scripts for TAS V2.
	DiscoverV2(ctx context.Context, subModule *SubModule, payload *Payload, secretData map[string]string,
		tasConfig *TASConfigV2, diff map[string]int, diffExists bool) error
	// UpdateSubmoduleList sends count of submodules to TAS server
	UpdateSubmoduleList(ctx context.Context, buildID string, totalSubmodule int) error
}

// BlockTestService is used for fetching blocklisted tests
type BlockTestService interface {
	GetBlockTests(ctx context.Context, blocklistYAML []string, repo, branch string) error
	GetBlocklistYMLV1(tasConfig *TASConfig) []string
	GetBlocklistYMLV2(submodule *SubModule) []string
}

// TestExecutionService services execution of tests
type TestExecutionService interface {
	// RunV1 executes the test execution scripts for TAS version 1
	RunV1(ctx context.Context, tasConfig *TASConfig,
		payload *Payload, coverageDirectory string, secretMap map[string]string) (results *ExecutionResults, err error)
	// SendResults sends the test execution results to the TAS server.
	SendResults(ctx context.Context, payload *ExecutionResults) (resp *TestReportResponsePayload, err error)
	// RunV2 executes the test execution scripts for TAS version 2
	RunV2(ctx context.Context,
		tasConfig *TASConfigV2,
		subModule *SubModule,
		payload *Payload,
		coverageDir string,
		envMap map[string]string,
		target []string,
		secretData map[string]string) (*ExecutionResults, error)
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
	GetSASURL(ctx context.Context, containerPath string, containerType ContainerType) (string, error)
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
	// CacheWorkspaceV2 caches the workspace onto a mounted volume
	// CacheWorkspaceV2(ctx context.Context, subModule string) error
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
		cwd string) error

	// ExecuteUserCommands executes the preRun or postRun commands given by user in his yaml. for tas version 2
	ExecuteUserCommandsV2(ctx context.Context, commandType CommandType, payload *Payload, runConfig *Run,
		secretData map[string]string, cwd, subModule string, buffer *bytes.Buffer) error

	// ExecuteInternalCommands executes the commands like installing runners and test discovery.
	ExecuteInternalCommands(ctx context.Context,
		commandType CommandType,
		commands []string,
		cwd string, envMap,
		secretData map[string]string) error
	// GetEnvVariables get the environment variables from the env map given by user.
	GetEnvVariables(envMap, secretData map[string]string) ([]string, error)
	// StoreCommandLogs stores the command logs in the azure.
	StoreCommandLogs(ctx context.Context, blobPath string, reader io.Reader) <-chan error
}

// Requests is a util interface for making API Requests
type Requests interface {
	// MakeAPIRequest makes an HTTP request
	MakeAPIRequest(ctx context.Context, httpMethod, endpoint string, body []byte) ([]byte, error)
}
