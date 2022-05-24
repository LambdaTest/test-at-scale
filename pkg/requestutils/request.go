package requestutils

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

type requests struct {
	logger lumber.Logger
	client http.Client
}

func New(logger lumber.Logger) core.Requests {
	return &requests{
		logger: logger,
		client: http.Client{Timeout: global.DefaultHTTPTimeout},
	}
}

func (r *requests) MakeAPIRequestWithAuth(ctx context.Context, httpMethod, endpoint string, body []byte, params,
	auth map[string]string) (rawBody []byte, statusCode int, err error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		r.logger.Errorf("error while parsing endpoint %s, %v", endpoint, err)
		return nil, http.StatusInternalServerError, err
	}
	q := u.Query()
	for id, val := range params {
		q.Set(id, val)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, httpMethod, u.String(), bytes.NewBuffer(body))
	if err != nil {
		r.logger.Errorf("error while creating http request %v", err)
		return nil, http.StatusInternalServerError, err
	}
	for id, val := range auth {
		req.Header.Add(id, val)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Errorf("error while sending http request %v", err)
		return nil, http.StatusInternalServerError, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		r.logger.Errorf("error while sending http response body %v", err)
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode != http.StatusOK {
		r.logger.Errorf("non 200 status code %s", string(respBody))
		return nil, resp.StatusCode, errors.New("non 200 status code")
	}

	return respBody, resp.StatusCode, nil
}
