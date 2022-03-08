package global

import (
	"time"
)

// all constant related to synapse
const (
	GracefulTimeout      = 100000 * time.Millisecond
	ProxyServerPort      = "8000"
	DirectoryPermissions = 0755
	FilePermissions      = 0755
	GitConfigFileName    = "oauth"
	RepoSecretsFileName  = "reposecrets"
	SynapseContainerURL  = "http://synapse:8000"
)

// SocketURL lambdatest url for synapse socket
var SocketURL map[string]string

// TASCloudURL url to send reports
var TASCloudURL map[string]string

func init() {
	SocketURL = map[string]string{
		"stage": "wss://stage-api.tas.lambdatest.com/ws/",
		"dev":   "ws://host.docker.internal/ws/",
		"prod":  "wss://api.tas.lambdatest.com/ws/",
	}
	TASCloudURL = map[string]string{
		"stage": "https://stage-api.tas.lambdatest.com",
		"dev":   "http://host.docker.internal",
		"prod":  "https://api.tas.lambdatest.com",
	}
}
