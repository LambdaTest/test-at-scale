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
	// LoadConfig loads the TASConfig from the given path
	LoadConfig(ctx context.Context, path string, eventType EventType, parseMode bool) (*TASConfig, error)
}

// GitManager manages the cloning of git repositories
type GitManager interface {
	// Clone repository from TAS config
	Clone(ctx context.Context, payload *Payload, cloneToken string) error
}

// DiffManager manages the diff findings for the given payload
type DiffManager interface {
	GetChangedFiles(ctx context.Context, payload *Payload, cloneToken string) (map[string]int, error)
}

// TestDiscoveryService services discovery of tests
type TestDiscoveryService interface {
	// Discover executes the test discovery scripts.
	Discover(ctx context.Context, tasConfig *TASConfig, payload *Payload, secretData map[string]string, diff map[string]int, diffExists bool) error
}

// BlockTestService is used for fetching blocklisted tests
type BlockTestService interface {
	GetBlockTests(ctx context.Context, tasConfig *TASConfig, repo, branch string) error
}

// TestExecutionService services execution of tests
type TestExecutionService interface {
	// Run executes the test execution scripts.
	Run(ctx context.Context, tasConfig *TASConfig, payload *Payload, coverageDirectory string, secretMap map[string]string) (*ExecutionResults, error)
}

// CoverageService services coverage of tests
type CoverageService interface {
	MergeAndUpload(ctx context.Context, payload *Payload) error
}

// YMLParserService parses the .tas.yml files
type YMLParserService interface {
	// ParseAndValidate the YML file and validades it
	ParseAndValidate(ctx context.Context, payload *Payload) error
}

// TestStats is used for servicing stat collection
type TestStats interface {
	CaptureTestStats(pid int32, collectStats bool) error
}

// Task is a service to update task status at neuron
type Task interface {
	// UpdateStatus updates status of the task
	UpdateStatus(payload *TaskPayload) error
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
type CacheStore interface {
	// Download downloads cache present at cacheKey
	Download(ctx context.Context, cacheKey string) error
	// Upload creates, compresses and uploads cache at cacheKey
	Upload(ctx context.Context, cacheKey string, itemsToCompress ...string) error
	// CacheWorkspace caches the workspace onto a mounted volume
	CacheWorkspace(ctx context.Context) error
	// ExtractWorkspace extracts the workspace cache from mounted volume
	ExtractWorkspace(ctx context.Context) error
}

// SecretParser defines operation for parsing the vault secrets in given path
type SecretParser interface {
	GetOauthSecret(filepath string) (*Oauth, error)
	GetRepoSecret(string) (map[string]string, error)
	SubstituteSecret(command string, secretData map[string]string) (string, error)
}

// ExecutionManager has responsibility for executing the preRun, postRun and internal commands
type ExecutionManager interface {
	// ExecuteUserCommands executes the preRun or postRun commands given by user in his yaml.
	ExecuteUserCommands(ctx context.Context, commandType CommandType, payload *Payload, runConfig *Run, secretData map[string]string) error
	// ExecuteInternalCommands executes the commands like installing runners and test discovery.
	ExecuteInternalCommands(ctx context.Context, commandType CommandType, commands []string, cwd string, envMap, secretData map[string]string) error
	// GetEnvVariables get the environment variables from the env map given by user.
	GetEnvVariables(envMap, secretData map[string]string) ([]string, error)
	// StoreCommandLogs stores the command logs in the azure.
	StoreCommandLogs(ctx context.Context, blobPath string, reader io.Reader) <-chan error
}
