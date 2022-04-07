// Package testexecutionservice is used for executing tests
package testexecutionservice

import (
	"context"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/stretchr/testify/mock"
)

// These tests are meant to be run on a Linux machine

func TestNewTestExecutionService(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	cfg := new(config.NucleusConfig)
	cfg.ConsecutiveRuns = 1
	cfg.CollectStats = true
	var ts *teststats.ProcStats
	azureClient := new(mocks.AzureClient)
	execManager := new(mocks.ExecutionManager)

	type args struct {
		execManager core.ExecutionManager
		azureClient core.AzureClient
		ts          *teststats.ProcStats
		logger      lumber.Logger
	}
	tests := []struct {
		name string
		args args
		want *testExecutionService
	}{
		{"TestNewTestExecutionService",
			args{execManager, azureClient, ts, logger},
			&testExecutionService{logger, azureClient, cfg, ts, execManager}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestExecutionService(cfg, tt.args.execManager,
				tt.args.azureClient, tt.args.ts, tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestExecutionService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_testExecutionService_GetLocatorsFile(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	var ts *teststats.ProcStats
	azureClient := new(mocks.AzureClient)
	execManager := new(mocks.ExecutionManager)
	azureClient.On("GetSASURL", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("core.ContainerType")).Return("sasURL", nil)
	azureClient.On("FindUsingSASUrl", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string")).Return(io.NopCloser(strings.NewReader("Hello, world!")), nil)

	type fields struct {
		logger      lumber.Logger
		azureClient core.AzureClient
		ts          *teststats.ProcStats
		execManager core.ExecutionManager
	}
	type args struct {
		ctx            context.Context
		locatorAddress string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{"Test GetLocatorsFile",
			fields{
				logger:      logger,
				azureClient: azureClient,
				ts:          ts,
				execManager: execManager,
			},
			args{
				ctx:            context.TODO(),
				locatorAddress: "locAddr",
			},
			"/tmp/locators",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tes := &testExecutionService{
				logger:      tt.fields.logger,
				azureClient: tt.fields.azureClient,
				ts:          tt.fields.ts,
				execManager: tt.fields.execManager,
			}
			got, err := tes.GetLocatorsFile(tt.args.ctx, tt.args.locatorAddress)
			if (err != nil) != tt.wantErr {
				t.Errorf("testExecutionService.GetLocatorsFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("testExecutionService.GetLocatorsFile() = %v, want %v", got, tt.want)
			}
			file, err := ioutil.ReadFile(got)
			if err != nil {
				t.Errorf("testExecutionService.GetLocatorsFile() error in opening file = %v", err)
				return
			}
			if string(file) != "Hello, world!" {
				t.Errorf("testExecutionService.GetLocatorsFile() = %v, want %v", string(file), "Hello, world!")
			}
		})
	}
}
