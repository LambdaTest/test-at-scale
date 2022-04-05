package zstd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	execManager := new(mocks.ExecutionManager)
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	_, err2 := New(execManager, logger)
	if err2 != nil {
		t.Errorf("Couldn't initialise a new zstdCompressor, error: %v", err2)
	}
}

func Test_zstdCompressor_createManifestFile(t *testing.T) {
	execManager := new(mocks.ExecutionManager)
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	path := "tar"

	type fields struct {
		logger      lumber.Logger
		execManager core.ExecutionManager
		execPath    string
	}
	type args struct {
		workingDir string
		fileNames  []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"Test createManifestFile", fields{logger: logger, execManager: execManager, execPath: path}, args{"./", []string{"file1", "file2"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &zstdCompressor{
				logger:      tt.fields.logger,
				execManager: tt.fields.execManager,
				execPath:    tt.fields.execPath,
			}
			if err := z.createManifestFile(tt.args.workingDir, tt.args.fileNames...); (err != nil) != tt.wantErr {
				t.Errorf("zstdCompressor.createManifestFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_zstdCompressor_Compress(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	path := "tar"

	// ReceivedStringArg will have args passed to ExecuteInternalCommands
	var ReceivedArgs []string
	execManager := new(mocks.ExecutionManager)
	execManager.On("ExecuteInternalCommands", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("core.CommandType"), mock.AnythingOfType("[]string"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			ReceivedArgs = commands
			return nil
		},
	)

	execManagerErr := new(mocks.ExecutionManager)
	execManagerErr.On("ExecuteInternalCommands", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("core.CommandType"), mock.AnythingOfType("[]string"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			ReceivedArgs = commands
			return errs.New("error from mocked interface")
		},
	)

	type fields struct {
		logger      lumber.Logger
		execManager core.ExecutionManager
		execPath    string
	}
	type args struct {
		ctx                context.Context
		compressedFileName string
		preservePath       bool
		workingDirectory   string
		filesToCompress    []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"Test Compress for success, with preservePath=true", fields{logger: logger, execManager: execManager, execPath: path}, args{context.TODO(), "compressedFileName", true, "./", []string{"f1", "f2"}}, false},

		{"Test Compress for success, with preservePath=false", fields{logger: logger, execManager: execManager, execPath: path}, args{context.TODO(), "compressedFileName", false, "./", []string{"f1", "f2"}}, false},

		{"Test Compress for error", fields{logger: logger, execManager: execManagerErr, execPath: path}, args{context.TODO(), "compressedFileName", true, "./", []string{"f1", "f2"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &zstdCompressor{
				logger:      tt.fields.logger,
				execManager: tt.fields.execManager,
				execPath:    tt.fields.execPath,
			}
			err := z.Compress(tt.args.ctx, tt.args.compressedFileName, tt.args.preservePath, tt.args.workingDirectory, tt.args.filesToCompress...)
			if (err != nil) != tt.wantErr {
				t.Errorf("zstdCompressor.Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			command := fmt.Sprintf("%s --posix -I 'zstd -5 -T0' -cf compressedFileName -C ./ -T %s", z.execPath, filepath.Join(os.TempDir(), manifestFileName))

			if tt.args.preservePath {
				command = fmt.Sprintf("%s -P", command)
			}
			commands := []string{command}
			if !reflect.DeepEqual(ReceivedArgs, commands) {
				t.Errorf("Expected commands: %v, got: %v", commands, ReceivedArgs)
			}
		})
	}
}

func Test_zstdCompressor_Decompress(t *testing.T) {

	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}

	path := "tar"

	// ReceivedStringArg will have args passed to ExecuteInternalCommands
	var ReceivedArgs []string
	execManager := new(mocks.ExecutionManager)
	execManager.On("ExecuteInternalCommands", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("core.CommandType"), mock.AnythingOfType("[]string"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			ReceivedArgs = commands
			return nil
		})

	execManagerErr := new(mocks.ExecutionManager)
	execManagerErr.On("ExecuteInternalCommands", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("core.CommandType"), mock.AnythingOfType("[]string"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(ctx context.Context, commandType core.CommandType, commands []string, cwd string, envMap, secretData map[string]string) error {
			ReceivedArgs = commands
			return errs.New("error from mocked interface")
		})

	type fields struct {
		logger      lumber.Logger
		execManager core.ExecutionManager
		execPath    string
	}
	type args struct {
		ctx              context.Context
		filePath         string
		preservePath     bool
		workingDirectory string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"Tests Decompress for success with preservePath=true", fields{logger: logger, execManager: execManager, execPath: path}, args{ctx: context.TODO(), filePath: "./", preservePath: true, workingDirectory: "./"}, false},

		{"Tests Decompress for success with preservePath=false", fields{logger: logger, execManager: execManager, execPath: path}, args{ctx: context.TODO(), filePath: "./", preservePath: false, workingDirectory: "./"}, false},

		{"Tests Decompress for error", fields{logger: logger, execManager: execManagerErr, execPath: path}, args{ctx: context.TODO(), filePath: "./", preservePath: true, workingDirectory: "./"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &zstdCompressor{
				logger:      tt.fields.logger,
				execManager: tt.fields.execManager,
				execPath:    tt.fields.execPath,
			}
			if err := z.Decompress(tt.args.ctx, tt.args.filePath, tt.args.preservePath, tt.args.workingDirectory); (err != nil) != tt.wantErr {
				t.Errorf("zstdCompressor.Decompress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			command := fmt.Sprintf("%s --posix -I 'zstd -d' -xf ./ -C ./", z.execPath)

			if tt.args.preservePath {
				command = fmt.Sprintf("%s -P", command)
			}
			commands := []string{command}
			if !reflect.DeepEqual(ReceivedArgs, commands) {
				t.Errorf("Expected args: %v, got: %v", commands, ReceivedArgs)
			}
		})
	}
}
