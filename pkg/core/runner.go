package core

import (
	"context"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/errs"
)

// Specs denotes system specification
type Specs struct {
	CPU float32
	RAM int64
}

// TierOpts is const map which map each tier to specs
var TierOpts = map[Tier]Specs{
	Internal: {CPU: 0.5, RAM: 384},
	XSmall:   {CPU: 1, RAM: 2000},
	Small:    {CPU: 2, RAM: 4000},
	Medium:   {CPU: 4, RAM: 8000},
	Large:    {CPU: 8, RAM: 16000},
	XLarge:   {CPU: 16, RAM: 32000},
}

// ContainerStatus contains status of container
type ContainerStatus struct {
	Done  bool
	Error errs.Err
}

// ContainerImageConfig contains registry config for docker
type ContainerImageConfig struct {
	AuthRegistry string
	Image        string
	Mode         config.ModeType
	PullPolicy   config.PullPolicyType
}

// DockerRunner defines operations for docker
type DockerRunner interface {
	// Creates the execution enging
	Create(context.Context, *RunnerOptions) ContainerStatus

	// Run runs the execution engine
	Run(context.Context, *RunnerOptions) ContainerStatus

	//WaitForRunning waits for runner to get completed
	WaitForCompletion(ctx context.Context, r *RunnerOptions) error

	// Destroy the execution engine
	Destroy(ctx context.Context, r *RunnerOptions) error

	// GetInfo will get resources details of the infra
	GetInfo(context.Context) (float32, int64)

	// Initiate runs docker containers
	Initiate(context.Context, *RunnerOptions, chan ContainerStatus)

	// PullImage will pull image from remote
	PullImage(containerImageConfig *ContainerImageConfig, r *RunnerOptions) error

	// KillRunningDocker kills  container spawn by synapse
	KillRunningDocker(ctx context.Context)
}

// RunnerOptions provides the the required instructions for execution engine.
type RunnerOptions struct {
	ContainerID               string            `json:"container_id"`
	DockerImage               string            `json:"docker_image"`
	ContainerPort             int               `json:"container_port"`
	HostPort                  int               `json:"host_port"`
	Label                     map[string]string `json:"label"`
	NameSpace                 string            `json:"name_space"`
	ServiceAccount            string            `json:"service_account"`
	PodName                   string            `json:"pod_name"`
	ContainerName             string            `json:"container_name"`
	ContainerArgs             []string          `json:"container_args"`
	ContainerCommands         []string          `json:"container_commands"`
	HostVolumePath            string            `json:"host_volume_path"`
	PersistentVolumeClaimName string            `json:"persistent_volume_claim_name"`
	Env                       []string          `json:"env"`
	OrgID                     string            `json:"org_id"`
	Vault                     *VaultOpts        `json:"vault"`
	LogfilePath               string            `json:"logfile_path"`
	PodType                   PodType           `json:"pod_type"`
	Tier                      Tier              `json:"tier"`
}

// VaultOpts provides the vault path options
type VaultOpts struct {
	// SecretPath path of the repo secrets.
	SecretPath string
	// TokenPath path of the user token.
	TokenPath string
	// RoleName vault role name
	RoleName string
	// Namespace is the default vault namespace
	Namespace string
}

// PodType specifies the type of pod
type PodType string

// Values that PodType can take
const (
	NucleusPod  PodType = "nucleus"
	CoveragePod PodType = "coverage"
)
