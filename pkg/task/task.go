package task

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

// task represents each instance of nucleus spawned by neuron
type task struct {
	ctx      context.Context
	client   http.Client
	endpoint string
	logger   lumber.Logger
}

// New returns new task
func New(ctx context.Context, cfg *config.NucleusConfig, logger lumber.Logger) (core.Task, error) {
	return &task{
		ctx:      ctx,
		client:   http.Client{Timeout: 30 * time.Second},
		logger:   logger,
		endpoint: global.NeuronHost + "/task",
	}, nil
}

func (t *task) UpdateStatus(payload *core.TaskPayload) error {

	t.logger.Debugf("sending status update of task: %s to %s for repository: %s", payload.TaskID, payload.Status, payload.RepoLink)
	reqBody, err := json.Marshal(payload)
	if err != nil {
		t.logger.Errorf("error while json marshal %v", err)
		return err
	}

	req, err := http.NewRequestWithContext(t.ctx, http.MethodPut, t.endpoint, bytes.NewBuffer(reqBody))

	if err != nil {
		t.logger.Errorf("error while creating http request %v", err)
		return err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Errorf("error while sending http request %v", err)
		return err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.logger.Errorf("error while sending http response body %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		t.logger.Errorf("non 200 status code %s", string(respBody))
		return errors.New("non 200 status code")
	}

	return nil

}
