package health

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name             string
		httpRequest      *http.Request
		wantResponseCode int
		wantStatusText   string
	}{

		{"Test handler health route for success", httptest.NewRequest(http.MethodGet, "/health", nil), 200, http.StatusText(http.StatusOK)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(resp)

			c.Request = tt.httpRequest

			router := gin.Default()
			router.GET("/health", Handler)
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
