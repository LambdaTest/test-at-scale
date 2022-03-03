package zstd

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
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
	args := []string{z.execPath, "--posix", "-I", "'zstd -5 -T0'", "-cf", compressedFileName, "-C", workingDirectory, "-T", filepath.Join(os.TempDir(), manifestFileName)}
	if preservePath {
		args = append(args, "-P")
	}
	if err := z.execManager.ExecuteInternalCommands(ctx, core.Zstd, args, workingDirectory, nil, nil); err != nil {
		z.logger.Errorf("error while zstd compression %v", err)
		return err
	}
	return nil
}

//Decompress performs the decompression operation for the given file
func (z *zstdCompressor) Decompress(ctx context.Context, filePath string, preservePath bool, workingDirectory string) error {
	args := []string{z.execPath, "--posix", "-I", "'zstd -d'", "-xf", filePath, "-C", workingDirectory}
	if preservePath {
		args = append(args, "-P")
	}
	if err := z.execManager.ExecuteInternalCommands(ctx, core.Zstd, args, workingDirectory, nil, nil); err != nil {
		z.logger.Errorf("error while zstd decompression %v", err)
		return err
	}
	return nil
}
