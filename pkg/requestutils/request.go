package requestutils

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

type requests struct {
	logger lumber.Logger
	client http.Client
}

func New(logger lumber.Logger) core.Requests {
	return &requests{
		logger: logger,
		client: http.Client{Timeout: 30 * time.Second},
	}
}

func (r *requests) MakeAPIRequest(ctx context.Context, httpMethod, endpoint string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, httpMethod, endpoint, bytes.NewBuffer(body))

	if err != nil {
		r.logger.Errorf("error while creating http request %v", err)
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Errorf("error while sending http request %v", err)
		return err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		r.logger.Errorf("error while sending http response body %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		r.logger.Errorf("non 200 status code %s", string(respBody))
		return errors.New("non 200 status code")
	}

	return nil
}
