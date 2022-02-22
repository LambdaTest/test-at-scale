package global

import "time"

// All constant related to nucleus
const (
	CoverageManifestFileName = "manifest.json"
	HomeDir                  = "/home/nucleus"
	RepoDir                  = HomeDir + "/repo"
	CodeCoverageDir          = RepoDir + "/coverage"
	DefaultHTTPTimeout       = 45 * time.Second
	SamplingTime             = 5 * time.Millisecond
	RepoSecretPath           = "/vault/secrets/reposecrets"
	OauthSecretPath          = "/vault/secrets/oauth"
	NeuronRemoteHost         = "http://neuron-service.phoenix"
	BlocklistedFileLocation  = RepoDir + "/blocklist.json"
	SecretRegex              = `\${{\s*secrets\.(.*?)\s*}}`
	ExecutionResultChunkSize = 50
	TestLocatorsDelimiter    = "#TAS#"
)

// FrameworkRunnerMap is map of framework with there respective runner location
var FrameworkRunnerMap = map[string]string{
	"jasmine": "./node_modules/.bin/jasmine-runner",
	"mocha":   "./node_modules/.bin/mocha-runner",
	"jest":    "./node_modules/.bin/jest-runner",
}

// RawContentURLMap is map of git provider with there raw content url
var RawContentURLMap = map[string]string{
	"github": "https://raw.githubusercontent.com",
}

// APIHostURLMap is map of git provider with there api url
var APIHostURLMap = map[string]string{
	"github": "https://api.github.com/repos",
	"gitlab": "https://gitlab.com/api/v4/projects",
}

// InstallRunnerCmd  are list of command used to install custom runner
var InstallRunnerCmd = []string{"tar", "-xzf", "/custom-runners/custom-runners.tgz"}

// NeuronHost is neuron host end point
var NeuronHost string

// SetNeuronHost is setter for NeuronHost
func SetNeuronHost(host string) {
	NeuronHost = host
}
