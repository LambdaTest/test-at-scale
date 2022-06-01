package cron

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
	"github.com/robfig/cron/v3"
)

const (
	buildCacheExpiry time.Duration = 4 * time.Hour
	buildCacheDir    string        = "/tmp/synapse"
)

// Setup initializes all crons on service startup
func Setup(ctx context.Context, wg *sync.WaitGroup, logger lumber.Logger) {
	defer wg.Done()

	c := cron.New()
	if _, err := c.AddFunc("@every 5m", func() { cleanupBuildCache(logger) }); err != nil {
		logger.Errorf("error setting up cron")
		return
	}
	c.Start()

	<-ctx.Done()
	c.Stop()
	logger.Infof("Caller has requested graceful shutdown. Returning.....")
}

func cleanupBuildCache(logger lumber.Logger) {
	files, err := os.ReadDir(buildCacheDir)
	if err != nil {
		logger.Errorf("error in reading directory: %s", err)
		return
	}
	for _, file := range files {
		now := time.Now()
		info, err := file.Info()
		if err != nil {
			logger.Errorf("error retrieving file info")
		}
		if diff := now.Sub(info.ModTime()); diff > buildCacheExpiry {
			filePath := fmt.Sprintf("%s/%s", buildCacheDir, file.Name())
			if err := utils.DeleteDirectory(filePath); err != nil {
				logger.Errorf("error deleting directory: %s", err.Error())
			}
		}
	}
}
