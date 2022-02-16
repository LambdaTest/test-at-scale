package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler for health API
func Handler(c *gin.Context) {
	c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
}
