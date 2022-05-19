package global

import "time"

// TestEnv : to set test env for urlmanager package
var TestEnv bool = false

// TestServer : store server URL of test server while doing mock testing
var TestServer string

// All constant related to nucleus
const (
	CoverageManifestFileName = "manifest.json"
	HomeDir                  = "/home/nucleus"
	WorkspaceCacheDir        = "/workspace-cache"
	RepoDir                  = HomeDir + "/repo"
	CodeCoverageDir          = RepoDir + "/coverage"
	DefaultHTTPTimeout       = 45 * time.Second
	DefaultGitCloneTimeout   = 30 * time.Minute
	SamplingTime             = 5 * time.Millisecond
	RepoSecretPath           = "/vault/secrets/reposecrets"
	OauthSecretPath          = "/vault/secrets/oauth"
	NeuronRemoteHost         = "http://neuron-service.phoenix"
	BlockTestFileLocation    = RepoDir + "/blocktests.json"
	SecretRegex              = `\${{\s*secrets\.(.*?)\s*}}`
	ExecutionResultChunkSize = 50
	TestLocatorsDelimiter    = "#TAS#"
	ExpiryDelta              = 15 * time.Minute
	NewTASVersion            = 2
	ModulePath               = "MODULE_PATH"
)

// FrameworkRunnerMap is map of framework with there respective runner location
var FrameworkRunnerMap = map[string]string{
	"jasmine": "./node_modules/lambdatest/packages/jasmine-runner/dist/src/jasmine-runner.js",
	"mocha":   "./node_modules/lambdatest/packages/mocha-runner/dist/src/mocha-runner.js",
	"jest":    "./node_modules/lambdatest/packages/jest-runner/dist/src/jest-runner.js",
}

// APIHostURLMap is map of git provider with there api url
var APIHostURLMap = map[string]string{
	"github":    "https://api.github.com/repos",
	"gitlab":    "https://gitlab.com/api/v4/projects",
	"bitbucket": "https://api.bitbucket.org/2.0",
}

// InstallRunnerCmds  are list of command used to install custom runner
var InstallRunnerCmds = []string{"tar -xzf /custom-runners/custom-runners.tgz"}

// NeuronHost is neuron host end point
var NeuronHost string

// SetNeuronHost is setter for NeuronHost
func SetNeuronHost(host string) {
	NeuronHost = host
}
