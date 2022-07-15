package requestutils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/cenkalti/backoff/v4"
)

type requests struct {
	logger       lumber.Logger
	client       http.Client
	retryBackoff backoff.BackOff
}

func New(logger lumber.Logger, requestTimeout time.Duration, retryBackoff backoff.BackOff) core.Requests {
	return &requests{
		logger:       logger,
		client:       http.Client{Timeout: requestTimeout},
		retryBackoff: retryBackoff,
	}
}

func (r *requests) MakeAPIRequest(
	ctx context.Context,
	httpMethod, endpoint string,
	body []byte,
	query map[string]interface{},
	headers map[string]string,
) (respBody []byte, statusCode int, err error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		r.logger.Errorf("error while parsing endpoint %s, %v", endpoint, err)
		return nil, 0, err
	}
	q := u.Query()
	for id, val := range query {
		v := reflect.ValueOf(val)
		// nolint:exhaustive
		switch v.Kind() {
		case reflect.Array:
		case reflect.Slice:
			for i := 0; i < v.Len(); i += 1 {
				q.Add(id, v.Index(i).String())
			}
		default:
			q.Set(id, fmt.Sprintf("%v", val))
		}
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, httpMethod, u.String(), bytes.NewBuffer(body))
	if err != nil {
		r.logger.Errorf("error while creating http request %v", err)
		return nil, 0, err
	}
	for id, val := range headers {
		req.Header.Add(id, val)
	}

	operation := func() error {
		resp, errD := r.client.Do(req)
		if errD != nil {
			r.logger.Errorf("error while sending http request %v", errD)
			return errD
		}
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		if 500 <= statusCode && statusCode < 600 {
			return fmt.Errorf("status code %d received", statusCode)
		}
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			r.logger.Errorf("error while reading http response body %v", err)
			return nil
		}
		return nil
	}
	if errR := backoff.Retry(operation, r.retryBackoff); errR != nil {
		r.logger.Errorf("Retry limit exceeded. Error %+v", errR)
		return respBody, statusCode, errors.New("retry limit exceeded")
	}
	if statusCode != http.StatusOK {
		r.logger.Errorf("non 200 status code %s", statusCode)
		return respBody, statusCode, errors.New("non 200 status code")
	}
	return respBody, statusCode, err
}
