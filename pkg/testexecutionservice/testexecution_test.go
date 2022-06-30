// Package testexecutionservice is used for executing tests
package testexecutionservice

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/pkg/service/teststats"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/cenkalti/backoff/v4"
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
	requests := requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{})

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
			&testExecutionService{logger, azureClient, cfg, ts, execManager, requests, global.NeuronHost + "/report"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTestExecutionService(cfg, requests, tt.args.execManager,
				tt.args.azureClient, tt.args.ts, tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTestExecutionService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shuffleLocators(t *testing.T) {
	locatorArrValue := []core.LocatorConfig{{
		Locator: "Locator_A"},
		{
			Locator: "Locator_B"},
		{
			Locator: "Locator_C"}}

	type args struct {
		locatorArr      []core.LocatorConfig
		locatorFilePath string
	}

	tests := []struct {
		name string
		args args
	}{
		{"Test_shuffleLocators",
			args{locatorArrValue, "/tmp/locators"}}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := shuffleLocators(tt.args.locatorArr, tt.args.locatorFilePath); err != nil {
				t.Errorf("shuffleLocators() throws error %v", err)
			}

			content, err := os.ReadFile(tt.args.locatorFilePath)
			if err != nil {
				t.Errorf("In test_shuffleLocators error in opening file = %v", err)
				return
			}
			t.Logf(string(content))
			// Now let's unmarshall the data into `payload`
			var payload core.InputLocatorConfig
			err = json.Unmarshal(content, &payload)
			if err != nil {
				t.Errorf("Error in unmarshlling = %v", err)
				return
			}
			if payload.Locators[0].Locator == "Locator_A" &&
				payload.Locators[1].Locator == "Locator_B" &&
				payload.Locators[2].Locator == "Locator_C" {
				t.Errorf("Shuffling could not be done, order is same as original")
			}
		})
	}
}

func Test_extractLocators(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	locatorArrValue := []core.LocatorConfig{{
		Locator: "Locator_A"},
		{
			Locator: "Locator_B"},
		{
			Locator: "Locator_C"}}
	type args struct {
		locatorFilePath string
		flakyTestAlgo   string
		logger          lumber.Logger
	}
	tests := []struct {
		name string
		args args
		want []core.LocatorConfig
	}{
		{"Test_extractLocators",
			args{"/tmp/locators", core.RunningXTimesShuffle, logger},
			locatorArrValue}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload core.InputLocatorConfig
			payload.Locators = locatorArrValue
			file, _ := json.Marshal(payload)
			_ = os.WriteFile(tt.args.locatorFilePath, file, global.FilePermissionWrite)
			if err != nil {
				t.Errorf("In test_extractLocators error in writing to file = %v", err)
				return
			}
			locatorArr, err := extractLocators(tt.args.locatorFilePath, tt.args.flakyTestAlgo, tt.args.logger)
			if err != nil {
				t.Errorf("extractLocators() throws error %v", err)
			}

			if !reflect.DeepEqual(locatorArrValue, locatorArr) {
				t.Errorf("extractLocators(), array got %s, want %s", locatorArr, locatorArrValue)
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
	azureClient.On("GetSASURL",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("core.ContainerType"),
	).Return("sasURL", nil)
	azureClient.On("FindUsingSASUrl",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("string"),
	).Return(io.NopCloser(strings.NewReader("Hello, world!")), nil)

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
			got, err := tes.getLocatorsFile(tt.args.ctx, tt.args.locatorAddress)
			if (err != nil) != tt.wantErr {
				t.Errorf("testExecutionService.GetLocatorsFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("testExecutionService.GetLocatorsFile() = %v, want %v", got, tt.want)
			}
			file, err := os.ReadFile(got)
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
