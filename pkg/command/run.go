package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/logstream"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
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
	secretData map[string]string,
	logwriter core.LogWriterStrategy,
	cwd string) error {
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

	errChan := logwriter.Write(ctx, azureReader)
	defer m.closeAndWriteLog(azureWriter, errChan, commandType)
	logWriter := lumber.NewWriter(m.logger)
	defer logWriter.Close()
	multiWriter := io.MultiWriter(logWriter, azureWriter)
	maskWriter := logstream.NewMasker(multiWriter, secretData)

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
	cmd.Dir = cwd
	cmd.Env = envVars
	cmd.Stdout = maskWriter
	cmd.Stderr = maskWriter

	if startErr := cmd.Start(); startErr != nil {
		m.logger.Errorf("failed to start command: %s, error: %v", commandType, startErr)
		return startErr
	}
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

func (m *manager) closeAndWriteLog(azureWriter *io.PipeWriter, errChan <-chan error, commandType core.CommandType) {
	azureWriter.Close()
	if uploadErr := <-errChan; uploadErr != nil {
		m.logger.Errorf("failed to upload logs for command %s, error: %v", commandType, uploadErr)
	}
}
