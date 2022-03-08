package main

// this is cmd/root_cmd.go

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/proxyserver"
	"github.com/LambdaTest/synapse/pkg/runner/docker"
	"github.com/LambdaTest/synapse/pkg/secrets"
	"github.com/LambdaTest/synapse/pkg/synapse"
	"github.com/LambdaTest/synapse/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd := cobra.Command{
		Use:     "synapse",
		Long:    `Synapse is an opensource runner for TAS`,
		Version: global.SYNAPSE_BINARY_VERSION,
		Run:     run,
	}

	// define flags used for this command
	if err := AttachCLIFlags(&rootCmd); err != nil {
		fmt.Println("Error in attaching cli flags")
	}

	return &rootCmd
}

func run(cmd *cobra.Command, args []string) {
	// create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// set necessary os env
	setEnv()
	// a WaitGroup for the goroutines to tell us they've stopped
	wg := sync.WaitGroup{}

	// Load environment variables from .env if available
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Warning: No .env file found\n")
	}

	cfg, err := config.LoadSynapseConfig(cmd)
	if err != nil {
		fmt.Printf("Failed to load config: %s", err.Error())
	}

	err = config.LoadRepoSecrets(cmd, cfg)
	if err != nil {
		fmt.Printf("Error loading repository secrets: %v", err)
	}

	// patch logconfig file location with root level log file location
	if cfg.LogFile != "" {
		cfg.LogConfig.FileLocation = filepath.Join(cfg.LogFile, "synapse.log")
	}

	// You can also use logrus implementation
	// by using lumber.InstanceLogrusLogger
	logger, err := lumber.NewLogger(cfg.LogConfig, cfg.Verbose, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	if err := config.ValidateCfg(cfg, logger); err != nil {
		logger.Fatalf("Error loading synapse config: %v", err)
	}
	secretsManager := secrets.New(cfg, logger)

	runner, err := docker.New(secretsManager, logger, cfg)
	if err != nil {
		logger.Fatalf("could not instantiate k8s runner %v", err)
	}

	synapse := synapse.New(runner, logger, secretsManager)

	proxyHandler, err := proxyserver.NewProxyHandler(logger)
	if err != nil {
		logger.Fatalf("Could not instantiate proxyhandler %v", err)
	}

	wg.Add(1)
	go synapse.InitiateConnection(ctx, &wg)

	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()
		if err := proxyserver.ListenAndServe(ctx, proxyHandler, cfg, logger); err != nil {
			logger.Fatalf("Error starting proxy server: %v", err)
		}
	}()

	// listen for C-cInterrupt
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
			logger.Debugf("main: received OS Interrupt signal, attempting graceful shutdown ....")
			// tell the goroutines to stop
			logger.Debugf("main: telling goroutines to stop")
			cancel()
			select {
			case <-done:
				logger.Debugf("Go routines exited within timeout")
			case <-time.After(global.GracefulTimeout):
				logger.Errorf("Graceful timeout exceeded. Brutally killing the application")
			}

		}
	case <-done:
		os.Exit(0)
	}

}

func setEnv() {
	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(true))
	os.Setenv(global.LocalEnv, strconv.FormatBool(true))
	os.Setenv(global.SynapseHostEnv, utils.GetOutboundIP())
	os.Setenv(global.NetworkEnvName, "test-at-scale")

}
