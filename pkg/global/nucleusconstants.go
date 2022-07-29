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
	RepoCacheDir             = RepoDir + "/__tas"
	DefaultAPITimeout        = 45 * time.Second
	DefaultGitCloneTimeout   = 30 * time.Minute
	SamplingTime             = 5 * time.Millisecond
	RepoSecretPath           = "/vault/secrets/reposecrets"
	OauthSecretPath          = "/vault/secrets/oauth"
	NeuronRemoteHost         = "http://neuron-service.phoenix.svc.cluster.local"
	BlockTestFileLocation    = "/tmp/blocktests.json"
	SecretRegex              = `\${{\s*secrets\.(.*?)\s*}}`
	ExecutionResultChunkSize = 50
	TestLocatorsDelimiter    = "#TAS#"
	ExpiryDelta              = 15 * time.Minute
	NewTASVersion            = 2
	ModulePath               = "MODULE_PATH"
	PackageJSON              = "package.json"
	SubModuleName            = "SUBMODULE_NAME"
	ArgPattern               = "--pattern"
	ArgConfig                = "--config"
	ArgDiff                  = "--diff"
	ArgCommand               = "--command"
	ArgLocator               = "--locator-file"
	ArgFrameworVersion       = "--frameworkVersion"
	DefaultTASVersion        = "1.0.0"
)

// FrameworkRunnerMap is map of framework with there respective runner location
var FrameworkRunnerMap = map[string]string{
	"jasmine": "./node_modules/.bin/jasmine-runner",
	"mocha":   "./node_modules/.bin/mocha-runner",
	"jest":    "./node_modules/.bin/jest-runner",
	"golang":  "/home/nucleus/server",
	"junit":   "mvn",
}

// APIHostURLMap is map of git provider with there api url
var APIHostURLMap = map[string]string{
	"github":    "https://api.github.com/repos",
	"gitlab":    "https://gitlab.com/api/v4/projects",
	"bitbucket": "https://api.bitbucket.org/2.0",
}

// InstallRunnerCmds  are list of command used to install custom runner
var InstallRunnerCmds = []string{"tar -xzf /custom-runners/custom-runners.tgz"}

var MavenSurefireVersionPluginGetCmds = []string{"xmlstarlet sel -t -v \"_:project/_:build/_:plugins/_:plugin[_:artifactId='maven-surefire-plugin']/_:version\" pom.xml"}
var MavenSurefireVersionPluginManagementGetCmds = []string{"xmlstarlet sel -t -v \"_:project/_:build/_:pluginManagement/_:plugins/_:plugin[_:artifactId='maven-surefire-plugin']/_:version\" pom.xml"}
var MavenSurefirePluginDependencyUpdateCmds = "xmlstarlet ed -O --inplace -a  \"/_:project/_:build/_:plugins/_:plugin[_:artifactId='maven-surefire-plugin']/_:artifactId\" --type elem --name \"dependencies\" -s //dependencies -t elem -n \"dependency\" -s //dependency -t elem -n \"groupId\" -v \"%s\" -s //dependency -t elem -n \"artifactId\" -v \"%s\" -s //dependency -t elem -n \"version\" -v \"1.0-SNAPSHOT\" pom.xml"
var MavenSurefirePluginManagementDependencyUpdateCmds = "xmlstarlet ed -O --inplace -a  \"/_:project/_:build/_:pluginManagement/_:plugins/_:plugin[_:artifactId='maven-surefire-plugin']/_:artifactId\" --type elem --name \"dependencies\" -s //dependencies -t elem -n \"dependency\" -s //dependency -t elem -n \"groupId\" -v \"%s\" -s //dependency -t elem -n \"artifactId\" -v \"%s\" -s //dependency -t elem -n \"version\" -v \"1.0-SNAPSHOT\" pom.xml"

// NeuronHost is neuron host end point
var NeuronHost string

// SetNeuronHost is setter for NeuronHost
func SetNeuronHost(host string) {
	NeuronHost = host
}

var FrameworkLanguageMap = map[string]string{
	"jasmine": "javascript",
	"mocha":   "javascript",
	"jest":    "javascript",
	"golang":  "golang",
	"junit":   "java",
}

var SurefireVersionMap = map[string]string{
	"2.19":     "com.lambdatest.surefire-2.19",
	"2.22":     "com.lambdatest.surefire-2.22",
	"3.0.0-M1": "com.lambdatest.surefire-3.0.0-M2",
	"3.0.0-M2": "com.lambdatest.surefire-3.0.0-M2",
	"3.0.0-M3": "com.lambdatest.surefire-3.0.0-M4",
	"3.0.0-M4": "com.lambdatest.surefire-3.0.0-M4",
	"3.0.0-M5": "com.lambdatest.surefire-3.0.0-M5",
	"3.0.0-M6": "com.lambdatest.surefire-3.0.0-M7",
	"3.0.0-M7": "com.lambdatest.surefire-3.0.0-M7",
}
