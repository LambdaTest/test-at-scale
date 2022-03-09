package results

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/LambdaTest/synapse/pkg/service/teststats"
	"github.com/LambdaTest/synapse/testutils"
	"github.com/gin-gonic/gin"
)

// NOTE: Tests in this package are meant to be run in a Linux environment

func TestHandler(t *testing.T) {
	logger, _ := testutils.GetLogger()
	cfg, _ := testutils.GetConfig()

	ts, err := teststats.New(cfg, logger)
	if err != nil {
		t.Errorf("Error creating teststats service: %v", err)
	}

	tests := []struct {
		name             string
		httpRequest      *http.Request
		wantResponseCode int
		wantStatusText   string
	}{

		{"Test handler result route", httptest.NewRequest(http.MethodPost, "/results", bytes.NewBuffer([]byte(`{"TaskID" : "123"}`))), 200, http.StatusText(http.StatusOK)},

		{"Test handler result route for error in jsonBinding and hence http.StatusBadRequest", httptest.NewRequest(http.MethodPost, "/results", nil), http.StatusBadRequest, `{"message":"EOF"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(resp)

			c.Request = tt.httpRequest

			router := gin.Default()
			router.POST("/results", Handler(logger, ts))
			router.ServeHTTP(resp, c.Request)

			fmt.Printf("Responsecode: %v\n", resp.Code)
			if resp.Code != tt.wantResponseCode {
				t.Errorf("Router.Handler() responseCode = %v, want = %v\n", resp.Code, tt.wantResponseCode)
				return
			}

			if !reflect.DeepEqual(resp.Body.String(), tt.wantStatusText) {
				t.Errorf("Router.Handler() statusText = %v, want = %v\n", resp.Body.String(), tt.wantStatusText)
			}
		})
	}

}
