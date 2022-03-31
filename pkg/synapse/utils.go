package synapse

import (
	"encoding/json"

	"github.com/LambdaTest/synapse/pkg/core"
)

// CreateLoginMessage creates message of type login
func CreateLoginMessage(loginDetails core.LoginDetails) core.Message {
	loginDetailsJson, err := json.Marshal(loginDetails)
	if err != nil {
		return core.Message{}
	}
	return core.Message{
		Type:    core.MsgLogin,
		Content: loginDetailsJson,
		Success: true,
	}
}

// CreateLogoutMessage creates message of type logout
func CreateLogoutMessage() core.Message {
	return core.Message{
		Type:    core.MsgLogout,
		Content: []byte(""),
		Success: true,
	}
}

// CreateJobInfo creates jobInfo based on status and runner
func CreateJobInfo(status core.StatusType, runnerOpts *core.RunnerOptions) core.JobInfo {
	jobInfo := core.JobInfo{
		Status:  status,
		JobID:   runnerOpts.Label[JobID],
		BuildID: runnerOpts.Label[BuildID],
		ID:      runnerOpts.Label[ID],
		Mode:    runnerOpts.Label[Mode],
	}
	return jobInfo
}

// CreateJobUpdateMessage creates message of type job updates
func CreateJobUpdateMessage(jobInfo core.JobInfo) core.Message {

	jobInfoJson, err := json.Marshal(jobInfo)
	if err != nil {
		return core.Message{}
	}
	return core.Message{
		Type:    core.MsgJobInfo,
		Content: []byte(jobInfoJson),
		Success: true,
	}
}

// CreateResourceStatsMessage creates message of type resource stats
func CreateResourceStatsMessage(resourceStats core.ResourceStats) core.Message {
	resourceStatsJson, err := json.Marshal(resourceStats)
	if err != nil {
		return core.Message{}
	}
	return core.Message{
		Type:    core.MsgResourceStats,
		Content: resourceStatsJson,
		Success: true,
	}
}

// GetResources returns dummy resources based on pod type
func GetResources(tierOpts core.Tier) core.Specs {
	if val, ok := core.TierOpts[tierOpts]; ok {
		return val
	}
	return core.Specs{CPU: 0, RAM: 0}
}
