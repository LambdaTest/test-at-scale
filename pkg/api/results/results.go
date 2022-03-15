package results

import (
	"net/http"

	"github.com/LambdaTest/synapse/pkg/service/teststats"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/gin-gonic/gin"
)

//Handler captures the test execution results from nucleus
func Handler(logger lumber.Logger, ts *teststats.ProcStats) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := core.ExecutionResults{}
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		go func() {
			ts.ExecutionResultInputChannel <- request
		}()
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}
