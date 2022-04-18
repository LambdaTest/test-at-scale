package main

// this is cmd/root_cmd.go

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/api"
	"github.com/LambdaTest/test-at-scale/pkg/azure"
	"github.com/LambdaTest/test-at-scale/pkg/blocktestservice"
	"github.com/LambdaTest/test-at-scale/pkg/cachemanager"
	"github.com/LambdaTest/test-at-scale/pkg/command"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/diffmanager"
	"github.com/LambdaTest/test-at-scale/pkg/gitmanager"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/payloadmanager"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/pkg/secret"
	"github.com/LambdaTest/test-at-scale/pkg/server"
	"github.com/LambdaTest/test-at-scale/pkg/service/coverage"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
	"github.com/LambdaTest/test-at-scale/pkg/tasconfigmanager"
	"github.com/LambdaTest/test-at-scale/pkg/task"
	"github.com/LambdaTest/test-at-scale/pkg/testdiscoveryservice"
	"github.com/LambdaTest/test-at-scale/pkg/testexecutionservice"
	"github.com/LambdaTest/test-at-scale/pkg/zstd"
	"github.com/spf13/cobra"
)

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd := cobra.Command{
		Use:     "nucleus",
		Long:    `nucleus is a coordinator binary used as entrypoint in tas containers`,
		Version: global.NUCLEUS_BINARY_VERSION,
		Run:     run,
	}

	// define flags used for this command
	AttachCLIFlags(&rootCmd)

	return &rootCmd
}

func run(cmd *cobra.Command, args []string) {
	// create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// timeout in seconds
	const gracefulTimeout = 5000 * time.Millisecond

	// a WaitGroup for the goroutines to tell us they've stopped
	wg := sync.WaitGroup{}

	cfg, err := config.LoadNucleusConfig(cmd)
	if err != nil {
		fmt.Printf("[Error] Failed to load config: " + err.Error())
		os.Exit(1)
	}

	// patch logconfig file location with root level log file location
	if cfg.LogFile != "" {
		cfg.LogConfig.FileLocation = filepath.Join(cfg.LogFile, "nucleus.log")
	}

	// You can also use logrus implementation
	// by using lumber.InstanceLogrusLogger
	logger, err := lumber.NewLogger(cfg.LogConfig, cfg.Verbose, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	logger.Debugf("Running on local: %t", cfg.LocalRunner)

	if cfg.LocalRunner {
		logger.Infof("Local runner detected , changing IP from: %s to: %s", global.NeuronHost, cfg.SynapseHost)
		global.SetNeuronHost(strings.TrimSpace(cfg.SynapseHost))

		logger.Infof("change neuron host to %s", global.NeuronHost)
	} else {
		global.SetNeuronHost(global.NeuronRemoteHost)
	}
	pl, err := core.NewPipeline(cfg, logger)
	if err != nil {
		logger.Errorf("Unable to create the pipeline: %+v\n", err)
		logger.Errorf("Aborting ...")
		os.Exit(1)
	}

	ts, err := teststats.New(cfg, logger)
	if err != nil {
		logger.Fatalf("failed to initialize test stats service: %v", err)
	}
	azureClient, err := azure.NewAzureBlobEnv(cfg, logger)
	if err != nil {
		logger.Fatalf("failed to initialize azure blob: %v", err)
	}
	if err != nil && !cfg.LocalRunner {
		logger.Fatalf("failed to initialize azure blob: %v", err)
	}

	// attach plugins to pipeline
	pm := payloadmanager.NewPayloadManger(azureClient, logger, cfg)
	secretParser := secret.New(logger)
	tcm := tasconfigmanager.NewTASConfigManager(logger)
	requests := requestutils.New(logger)
	execManager := command.NewExecutionManager(secretParser, azureClient, logger)
	gm := gitmanager.NewGitManager(logger, execManager)
	dm := diffmanager.NewDiffManager(cfg, logger)

	tdResChan := make(chan core.DiscoveryResult)
	tds := testdiscoveryservice.NewTestDiscoveryService(ctx, tdResChan, execManager, requests, logger)
	tes := testexecutionservice.NewTestExecutionService(cfg, execManager, azureClient, ts, logger)
	tbs, err := blocktestservice.NewTestBlockTestService(cfg, logger)
	if err != nil {
		logger.Fatalf("failed to initialize test blocklist service: %v", err)
	}
	router := api.NewRouter(logger, ts, tdResChan)

	t, err := task.New(ctx, requests, logger)
	if err != nil {
		logger.Fatalf("failed to initialize task: %v", err)
	}

	zstd, err := zstd.New(execManager, logger)
	if err != nil {
		logger.Fatalf("failed to initialize zstd compressor: %v", err)
	}
	cache, err := cachemanager.New(zstd, azureClient, logger)
	if err != nil {
		logger.Fatalf("failed to initialize cache manager: %v", err)
	}

	coverageService, err := coverage.New(execManager, azureClient, zstd, cfg, logger)
	if err != nil {
		logger.Fatalf("failed to initialize coverage service: %v", err)
	}

	pl.PayloadManager = pm
	pl.TASConfigManager = tcm
	pl.GitManager = gm
	pl.DiffManager = dm
	pl.TestDiscoveryService = tds
	pl.BlockTestService = tbs
	pl.TestExecutionService = tes
	pl.ExecutionManager = execManager
	pl.CoverageService = coverageService
	pl.TestStats = ts
	pl.Task = t
	pl.CacheStore = cache
	pl.SecretParser = secretParser

	logger.Infof("LambdaTest Nucleus version: %s", global.NUCLEUS_BINARY_VERSION)

	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()
		// starting pipeline
		pl.Start(ctx)
	}()
	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()
		server.ListenAndServe(ctx, router, cfg, logger)
	}()
	// listen for C-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// create channel to mark status of waitgroup
	// this is required to brutally kill application in case of
	// timeout
	done := make(chan struct{})

	// asynchronously wait for all the go routines
	go func() {
		// and wait for all go routines
		wg.Wait()
		logger.Debugf("main: all goroutines have finished.")
		close(done)
	}()

	// wait for signal channel
	select {
	case <-c:
		{
			logger.Debugf("main: received C-c - attempting graceful shutdown ....")
			// tell the goroutines to stop
			logger.Debugf("main: telling goroutines to stop")
			cancel()
			select {
			case <-done:
				logger.Debugf("Go routines exited within timeout")
			case <-time.After(gracefulTimeout):
				logger.Errorf("Graceful timeout exceeded. Brutally killing the application")
			}

		}
	case <-done:
		os.Exit(0)
	}

}
