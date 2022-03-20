package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/logstream"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

type manager struct {
	logger       lumber.Logger
	secretParser core.SecretParser
	azureClient  core.AzureClient
}

// NewExecutionManager returns new instance of manger
func NewExecutionManager(secretParser core.SecretParser,
	azureClient core.AzureClient,
	logger lumber.Logger) core.ExecutionManager {
	return &manager{logger: logger,
		secretParser: secretParser,
		azureClient:  azureClient}
}

// ExecuteUserCommands executes user commands
func (m *manager) ExecuteUserCommands(ctx context.Context,
	commandType core.CommandType,
	payload *core.Payload,
	runConfig *core.Run,
	secretData map[string]string) error {
	script, err := m.createScript(runConfig.Commands, secretData)
	if err != nil {
		return err
	}
	envVars, err := m.GetEnvVariables(runConfig.EnvMap, secretData)
	if err != nil {
		return err
	}

	azureReader, azureWriter := io.Pipe()
	defer azureWriter.Close()

	blobPath := fmt.Sprintf("%s/%s/%s/%s.log", payload.OrgID, payload.BuildID, os.Getenv("TASK_ID"), commandType)
	errChan := m.StoreCommandLogs(ctx, blobPath, azureReader)

	logWriter := lumber.NewWriter(m.logger)
	defer logWriter.Close()
	multiWriter := io.MultiWriter(logWriter, azureWriter)
	maskWriter := logstream.NewMasker(multiWriter, secretData)

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
	cmd.Dir = global.RepoDir
	cmd.Env = envVars
	cmd.Stdout = maskWriter
	cmd.Stderr = maskWriter

	if startErr := cmd.Start(); startErr != nil {
		m.logger.Errorf("failed to start command: %s, error: %v", commandType, startErr)
		return startErr
	}
	m.logger.Debugf("command of type %s started with id %d", commandType, cmd.Process.Pid)
	if execErr := cmd.Wait(); execErr != nil {
		m.logger.Errorf("command %s, exited with error: %v", commandType, execErr)
		return execErr
	}
	azureWriter.Close()
	if uploadErr := <-errChan; uploadErr != nil {
		m.logger.Errorf("failed to upload logs for command %s, error: %v", commandType, uploadErr)
		return uploadErr
	}
	return nil
}

// ExecuteInternalCommands executes internal commands
func (m *manager) ExecuteInternalCommands(ctx context.Context,
	commandType core.CommandType,
	commands []string,
	cwd string,
	envMap, secretData map[string]string) error {
	bashCommands := strings.Join(commands, " && ")
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", bashCommands)
	if cwd != "" {
		cmd.Dir = cwd
	}
	logWriter := lumber.NewWriter(m.logger)
	defer logWriter.Close()
	cmd.Stderr = logWriter
	cmd.Stdout = logWriter
	m.logger.Debugf("Executing command of type %s", commandType)
	if err := cmd.Run(); err != nil {
		m.logger.Errorf("command of type %s failed with error: %v", commandType, err)
		return err
	}
	return nil
}

// GetEnvVariables gives set environment variable
func (m *manager) GetEnvVariables(envMap, secretData map[string]string) ([]string, error) {
	envVars := os.Environ()
	for k, v := range envMap {
		val, err := m.secretParser.SubstituteSecret(v, secretData)
		if err != nil {
			return nil, err
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, val))
	}
	return envVars, nil
}

// StoreCommandLogs stores the command logs to blob
func (m *manager) StoreCommandLogs(ctx context.Context, blobPath string, reader io.Reader) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		sasURL, err := m.azureClient.GetSASURL(ctx, blobPath, core.LogsContainer)
		if err != nil {
			m.logger.Errorf("failed to genereate SAS URL for path %s, error: %v", blobPath, err)
			errChan <- err
			return
		}
		blobPath, err := m.azureClient.CreateUsingSASURL(ctx, sasURL, reader, "text/plain")
		if err != nil {
			m.logger.Errorf("failed to create SAS URL for path %s, error: %v", blobPath, err)
			errChan <- err
			return
		}
		close(errChan)
		m.logger.Debugf("created blob path %s", blobPath)
	}()
	return errChan
}
