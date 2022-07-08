package logwriter

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

type (
	BufferLogWriter struct {
		subModule string
		buffer    *bytes.Buffer
		logger    lumber.Logger
	}

	AzureLogWriter struct {
		azureClient core.AzureClient
		blobPath    string
		logger      lumber.Logger
	}
)

func NewAzureLogWriter(azureClient core.AzureClient,
	blobPath string,
	logger lumber.Logger) core.LogWriterStrategy {
	return &AzureLogWriter{
		azureClient: azureClient,
		blobPath:    blobPath,
		logger:      logger,
	}
}

func NewBufferLogWriter(subModule string,
	buffer *bytes.Buffer,
	logger lumber.Logger) core.LogWriterStrategy {
	return &BufferLogWriter{
		subModule: subModule,
		buffer:    buffer,
		logger:    logger,
	}
}

func (b *BufferLogWriter) Write(ctx context.Context, reader io.Reader) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		if _, err := fmt.Fprintf(b.buffer, "\n<------ PRE RUN for %s  ------> \n", b.subModule); err != nil {
			b.logger.Debugf("Error writing the logs separator for submodule %s, error %v", b.subModule, err)
			errChan <- err
			return
		}
		if _, err := b.buffer.ReadFrom(reader); err != nil {
			b.logger.Debugf("Error writing the logs to buffer for submodule %s, error %v", b.subModule, err)
			errChan <- err
			return
		}
		close(errChan)
		b.logger.Debugf("written logs for sub module %s to buffer", b.subModule)
	}()
	return errChan
}

func (a *AzureLogWriter) Write(ctx context.Context, reader io.Reader) <-chan error {
	errChan := make(chan error, 1)
	go func() {
		sasURL, err := a.azureClient.GetSASURL(ctx, a.blobPath, core.LogsContainer)
		if err != nil {
			a.logger.Errorf("failed to genereate SAS URL for path %s, error: %v", a.blobPath, err)
			errChan <- err
			return
		}
		blobPath, err := a.azureClient.CreateUsingSASURL(ctx, sasURL, reader, "text/plain")
		if err != nil {
			a.logger.Errorf("failed to create SAS URL for path %s, error: %v", blobPath, err)
			errChan <- err
			return
		}
		close(errChan)
		a.logger.Debugf("created blob path %s", blobPath)
	}()
	return errChan
}
