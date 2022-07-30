package cron

import (
	"context"
	"sync"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/robfig/cron/v3"
)

// Setup initializes all crons on service startup
func Setup(ctx context.Context, wg *sync.WaitGroup, logger lumber.Logger, runner core.DockerRunner) {
	defer wg.Done()

	c := cron.New()
	if _, err := c.AddFunc("@every 5m", func() { cleanupBuildCache(runner) }); err != nil {
		logger.Errorf("error setting up cron")
		return
	}
	c.Start()

	<-ctx.Done()
	c.Stop()
	logger.Infof("Caller has requested graceful shutdown. Returning.....")
}

func cleanupBuildCache(runner core.DockerRunner) {
	runner.RemoveOldVolumes(context.Background())
}
