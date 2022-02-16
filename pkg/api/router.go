package api

import (
	"github.com/LambdaTest/synapse/pkg/api/health"
	"github.com/LambdaTest/synapse/pkg/api/results"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/service/teststats"
	"github.com/gin-gonic/gin"
)

// Router for nucleus
type Router struct {
	logger           lumber.Logger
	testStatsService *teststats.ProcStats
}

// NewRouter returns instance of Router
func NewRouter(logger lumber.Logger, ts *teststats.ProcStats) Router {
	return Router{
		logger:           logger,
		testStatsService: ts,
	}
}

//Handler function will perform all route operations
func (r Router) Handler() *gin.Engine {

	r.logger.Infof("Setting up routes")
	router := gin.Default()
	// corsConfig := cors.DefaultConfig()
	// corsConfig.AllowAllOrigins = true
	// corsConfig.AddAllowHeaders("authorization", "cache-control", "pragma")
	// router.Use(cors.New(corsConfig))
	router.GET("/health", health.Handler)
	router.POST("/results", results.Handler(r.logger, r.testStatsService))

	return router

}
