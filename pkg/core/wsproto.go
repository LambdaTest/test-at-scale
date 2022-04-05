package core

// MessageType defines type of message
type MessageType string

// StatusType defines type job status
type StatusType string

// StatType defines type of resource status
type StatType string

// types of messages
const (
	MsgLogin         MessageType = "login"
	MsgLogout        MessageType = "logout"
	MsgTask          MessageType = "task"
	MsgInfo          MessageType = "info"
	MsgError         MessageType = "error"
	MsgResourceStats MessageType = "resourcestats"
	MsgJobInfo       MessageType = "jobinfo"
)

// JobInfo types
const (
	JobCompleted StatusType = "complete"
	JobStarted   StatusType = "started"
	JobFailed    StatusType = "failed"
)

// ResourceStats types
const (
	ResourceRelease StatType = "release"
	ResourceCapture StatType = "capture"
)

// Message struct
type Message struct {
	Type    MessageType `json:"type"`
	Content []byte      `json:"content"`
	Success bool        `json:"success"`
}

// LoginDetails struct
type LoginDetails struct {
	Name      string  `json:"name"`
	SynapseID string  `json:"synapse_id"`
	SecretKey string  `json:"secret_key"`
	CPU       float32 `json:"cpu"`
	RAM       int64   `json:"ram"`
}

// ResourceStats struct for CPU, RAM details
type ResourceStats struct {
	Status StatType `json:"status"`
	CPU    float32  `json:"cpu"`
	RAM    int64    `json:"ram"`
}

// JobInfo stuct for job updates info
type JobInfo struct {
	Status  StatusType `json:"status"`
	JobID   string     `json:"job_id"`
	ID      string     `json:"id"`
	Mode    string     `json:"mode"`
	BuildID string     `json:"build_id"`
}
