package synapse

import (
	"encoding/json"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestCreateLoginMessage(t *testing.T) {
	loginDetails := core.LoginDetails{
		SecretKey: "dummysecretkey",
		CPU:       4,
		RAM:       4096,
	}
	loginMessage := CreateLoginMessage(loginDetails)
	loginDetailsJSON, err := json.Marshal(loginDetails)
	if err != nil {
		t.Errorf("error in marshaling login details: %v", err)
	}
	assert.Equal(t, loginDetailsJSON, loginMessage.Content)
	assert.Equal(t, core.MsgLogin, loginMessage.Type)
}

func TestCreateLogoutMessage(t *testing.T) {
	logoutMessage := CreateLogoutMessage()
	assert.Empty(t, logoutMessage.Content)
	assert.Equal(t, core.MsgLogout, logoutMessage.Type)
}

func TestCreateJobUpdateMessage(t *testing.T) {
	jobInfo := core.JobInfo{
		Status:  core.JobCompleted,
		JobID:   "dummyjobid",
		ID:      "dummyid",
		Mode:    "nucleus",
		BuildID: "dummybuildid",
	}
	jobInfoMessage := CreateJobUpdateMessage(jobInfo)
	jobInfoJSON, err := json.Marshal(jobInfo)
	if err != nil {
		t.Errorf("error in marshaling job info: %v", err)
	}
	assert.Equal(t, jobInfoJSON, jobInfoMessage.Content)
	assert.Equal(t, core.MsgJobInfo, jobInfoMessage.Type)
}

func TestCreateResourceStatsMessage(t *testing.T) {
	resourceStats := core.ResourceStats{
		Status: core.ResourceRelease,
		CPU:    2,
		RAM:    2000,
	}
	resourceStatsMessage := CreateResourceStatsMessage(resourceStats)
	resourceStatsJSON, err := json.Marshal(resourceStats)
	if err != nil {
		t.Errorf("error in marshaling job info: %v", err)
	}
	assert.Equal(t, resourceStatsJSON, resourceStatsMessage.Content)
	assert.Equal(t, core.MsgResourceStats, resourceStatsMessage.Type)
}
