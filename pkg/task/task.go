package task

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

// task represents each instance of nucleus spawned by neuron
type task struct {
	requests core.Requests
	endpoint string
	logger   lumber.Logger
}

// New returns new task
func New(requests core.Requests, logger lumber.Logger) (core.Task, error) {
	return &task{
		requests: requests,
		logger:   logger,
		endpoint: global.NeuronHost + "/task",
	}, nil
}

func (t *task) UpdateStatus(ctx context.Context, payload *core.TaskPayload) error {
	t.logger.Debugf("sending status update of task: %s to %s for repository: %s", payload.TaskID, payload.Status, payload.RepoLink)
	reqBody, err := json.Marshal(payload)
	if err != nil {
		t.logger.Errorf("error while json marshal %v", err)
		return err
	}

	if _, err := t.requests.MakeAPIRequestWithAuth(ctx, http.MethodPut, t.endpoint, reqBody); err != nil {
		return err
	}

	return nil

}
