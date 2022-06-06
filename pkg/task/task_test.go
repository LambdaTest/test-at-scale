package task

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/cenkalti/backoff/v4"
)

var noContext = context.Background()

const (
	taskE  = "/task"
	non200 = "non 200 status code"
)

func TestTask_UpdateStatus(t *testing.T) {
	check := func(t *testing.T, st int) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != taskE {
				t.Errorf("Expected to request '/task', got: %v", r.URL)
				return
			}
			w.WriteHeader(st)
			_, err := w.Write([]byte(`{"value":"fixed"}`))
			if err != nil {
				fmt.Printf("Could not write data in httptest server, error: %v", err)
			}
		}))
		defer server.Close()

		logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
		requests := requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{})
		if err != nil {
			fmt.Println("Logger can't be established")
		}

		taskPayload, err := testutils.GetTaskPayload()
		if err != nil {
			t.Errorf("Couldn't get task payload, received: %v", err)
		}
		_, err2 := New(requests, logger)
		if err2 != nil {
			t.Errorf("New task couldn't initialized, received: %v", err)
		}
		tk := &task{
			requests: requests,
			logger:   logger,
			endpoint: server.URL + taskE,
		}

		updateStatusErr := tk.UpdateStatus(noContext, taskPayload)

		if st != 200 {
			expectedErr := non200
			if updateStatusErr == nil {
				t.Errorf("Expected: %s, Received: %s", expectedErr, updateStatusErr)
			}
			return
		}
		if updateStatusErr != nil {
			t.Errorf("Received: %v", updateStatusErr)
		}
	}

	t.Run("TestUpdateStatus check for statusOK", func(t *testing.T) {
		check(t, 200)
	})
	t.Run("TestUpdateStatus check for non statusOK", func(t *testing.T) {
		check(t, 404)
	})
}

func TestTask_UpdateStatusForError(t *testing.T) {
	checkErr := func(t *testing.T, st int) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != taskE {
				t.Errorf("Expected to request '/task', got: %v", r.URL)
			}
			w.WriteHeader(st)
			w.Header().Set("Content-Type", "application/json")
		}))
		defer server.Close()

		logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
		requests := requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{})
		if err != nil {
			fmt.Println("Logger can't be established")
		}

		taskPayload, err := testutils.GetTaskPayload()
		if err != nil {
			t.Errorf("Couldn't get task payload, received: %v", err)
		}
		tk := &task{
			requests: requests,
			logger:   logger,
			endpoint: server.URL + taskE,
		}

		updateStatusErr := tk.UpdateStatus(noContext, taskPayload)

		if st != 200 {
			expectedErr := non200
			if expectedErr != updateStatusErr.Error() {
				t.Errorf("Expected: %s, Received: %s", expectedErr, updateStatusErr)
			}
			return
		}
		if updateStatusErr != nil {
			t.Errorf("Received: %v", updateStatusErr)
		}
	}
	t.Run("TestUpdateStatus check for error", func(t *testing.T) {
		checkErr(t, 404) // statusNotFound
	})
}
