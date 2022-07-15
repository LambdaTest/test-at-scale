package task

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
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
	query, headers := utils.GetDefaultQueryAndHeaders()
	if _, _, err := t.requests.MakeAPIRequest(ctx, http.MethodPut, t.endpoint, reqBody, query, headers); err != nil {
		return err
	}

	return nil

}
