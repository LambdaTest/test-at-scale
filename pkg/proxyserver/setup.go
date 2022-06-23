package proxyserver

import (
	"context"
	"net/http"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// ListenAndServe  starts proxy http server for synapse
func ListenAndServe(ctx context.Context, proxyHandler *ProxyHandler, config *config.SynapseConfig, logger lumber.Logger) error {
	gin.SetMode(gin.ReleaseMode)
	logger.Infof("Setting up HTTP handler")

	errChan := make(chan error)

	// HTTP server instance
	srv := &http.Server{
		Addr:    ":" + global.ProxyServerPort,
		Handler: http.HandlerFunc(proxyHandler.HandlerProxy),
	}
	// channel to signal server process exit
	done := make(chan struct{})
	go func() {
		logger.Infof("Starting server on port %s", global.ProxyServerPort)
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("listen: %#v", err)
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Infof("Caller has requested graceful shutdown. shutting down the server")
		if err := srv.Shutdown(ctx); err != nil {
			logger.Errorf("Server Shutdown:", "error", err)
		}
		return nil
	case err := <-errChan:
		return err
	case <-done:
		return nil
	}
}
