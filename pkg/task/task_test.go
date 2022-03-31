package task

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/requestutils"
	"github.com/LambdaTest/synapse/testutils"
)

func TestTask_UpdateStatus(t *testing.T) {

	check := func(t *testing.T, st int) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/task" {
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
		requests := requestutils.New(logger)
		if err != nil {
			fmt.Println("Logger can't be established")
		}

		taskPayload, err := testutils.GetTaskPayload()
		if err != nil {
			t.Errorf("Couldn't get task payload, received: %v", err)
		}
		_, err2 := New(context.TODO(), requests, logger)
		if err2 != nil {
			t.Errorf("New task couldn't initialised, received: %v", err)
		}
		tk := &task{
			ctx:      context.TODO(),
			requests: requests,
			logger:   logger,
			endpoint: server.URL + "/task",
		}

		updateStatusErr := tk.UpdateStatus(taskPayload)

		if st != 200 {
			expectedErr := "non 200 status code"
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
		check(t, 200) // statusOk = 200
	})
	t.Run("TestUpdateStatus check for non statusOK", func(t *testing.T) {
		check(t, 404) // statusNotFound
	})
}

func TestTask_UpdateStatusForError(t *testing.T) {

	checkErr := func(t *testing.T, st int) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/task" {
				t.Errorf("Expected to request '/task', got: %v", r.URL)
			}
			w.WriteHeader(st)
			w.Header().Set("Content-Type", "application/json")
			// w.Write([]byte(`{"value":"fixed"}`))
		}))
		defer server.Close()

		logger, err := lumber.NewLogger(lumber.LoggingConfig{ConsoleLevel: lumber.Debug}, true, 1)
		requests := requestutils.New(logger)
		if err != nil {
			fmt.Println("Logger can't be established")
		}

		taskPayload, err := testutils.GetTaskPayload()
		if err != nil {
			t.Errorf("Couldn't get task payload, received: %v", err)
		}
		tk := &task{
			ctx:      context.TODO(),
			requests: requests,
			logger:   logger,
			endpoint: server.URL + "/task",
		}

		updateStatusErr := tk.UpdateStatus(taskPayload)

		if st != 200 {
			expectedErr := "non 200 status code"
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
