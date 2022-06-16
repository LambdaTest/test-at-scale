package zstd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

type zstdCompressor struct {
	logger      lumber.Logger
	execManager core.ExecutionManager
	execPath    string
}

const (
	manifestFileName = "manifest.txt"
	executableName   = "tar"
)

//New return zStandard compression manager
func New(execManager core.ExecutionManager, logger lumber.Logger) (core.ZstdCompressor, error) {
	path, err := exec.LookPath(executableName)
	if err != nil {
		logger.Errorf("failed to find path for tar, error:%v", err)
		return nil, err
	}

	return &zstdCompressor{logger: logger, execManager: execManager, execPath: path}, nil
}

func (z *zstdCompressor) createManifestFile(workingDir string, fileNames ...string) error {
	return ioutil.WriteFile(filepath.Join(os.TempDir(), manifestFileName), []byte(strings.Join(fileNames, "\n")), 0660)
}

// Compress compress the list of files
func (z *zstdCompressor) Compress(ctx context.Context, compressedFileName string, preservePath bool, workingDirectory string, filesToCompress ...string) error {
	if err := z.createManifestFile(workingDirectory, filesToCompress...); err != nil {
		z.logger.Errorf("failed to create manifest file %v", err)
		return err
	}
	command := fmt.Sprintf("%s --posix -I 'zstd -5 -T0' -cf %s -C %s -T %s", z.execPath, compressedFileName, workingDirectory, filepath.Join(os.TempDir(), manifestFileName))
	if preservePath {
		command = fmt.Sprintf("%s -P", command)
	}
	commands := []string{command}
	if err := z.execManager.ExecuteInternalCommands(ctx, core.Zstd, commands, workingDirectory, nil, nil); err != nil {
		z.logger.Errorf("error while zstd compression %v", err)
		return err
	}
	return nil
}

//Decompress performs the decompression operation for the given file
func (z *zstdCompressor) Decompress(ctx context.Context, filePath string, preservePath bool, workingDirectory string) error {
	command := fmt.Sprintf("%s --posix -I 'zstd -d' -xf %s -C %s", z.execPath, filePath, workingDirectory)
	if preservePath {
		command = fmt.Sprintf("%s -P", command)
	}
	commands := []string{command}
	if err := z.execManager.ExecuteInternalCommands(ctx, core.Zstd, commands, workingDirectory, nil, nil); err != nil {
		z.logger.Errorf("error while zstd decompression %v", err)
		return err
	}
	return nil
}
