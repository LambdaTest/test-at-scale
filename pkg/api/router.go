package api

import (
	"github.com/LambdaTest/test-at-scale/pkg/api/health"
	"github.com/LambdaTest/test-at-scale/pkg/api/results"
	"github.com/LambdaTest/test-at-scale/pkg/api/testlist"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
	"github.com/gin-gonic/gin"
)

// Router for nucleus
type Router struct {
	logger           lumber.Logger
	testStatsService *teststats.ProcStats
	tdResChan        chan core.DiscoveryResult
}

// NewRouter returns instance of Router
func NewRouter(logger lumber.Logger, ts *teststats.ProcStats, tdResChan chan core.DiscoveryResult) Router {
	return Router{
		logger:           logger,
		testStatsService: ts,
		tdResChan:        tdResChan,
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
	router.POST("/test-list", testlist.Handler(r.logger, r.tdResChan))

	return router

}
