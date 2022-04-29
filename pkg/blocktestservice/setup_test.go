// Package blocktestservice is used for creating the blocklist file
package blocktestservice

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/testutils"
)

func TestNewTestBlockListService(t *testing.T) {
	cfg := config.GlobalNucleusConfig
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}

	want := TestBlockTestService{
		cfg:               cfg,
		logger:            logger,
		endpoint:          "endpoint",
		blockTestEntities: make(map[string][]blocktest),
		errChan:           make(chan error, 1),
		httpClient: http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		}}
	type args struct {
		cfg    *config.NucleusConfig
		logger lumber.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    *TestBlockTestService
		wantErr bool
	}{
		{"Test NewTestBlockListService, it should give new TestBlockListService struct with provided arguments",
			args{
				cfg:    cfg,
				logger: logger,
			},
			&want,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTestBlockTestService(tt.args.cfg, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTestBlockListService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestTestBlockListService_fetchBlockListFromNeuron(t *testing.T) {
	server := httptest.NewServer( // mock server
		http.FileServer(http.Dir("../../testutils/testdata/testblocklistdata/")), // mock data stored at testutils/testdata
	)
	defer server.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/non200" {
			t.Errorf("Expected to request '/non200', got: %v", r.URL)
			return
		}
		w.WriteHeader(503)
		_, err := w.Write([]byte(`{"value":"fixed"}`))
		if err != nil {
			fmt.Printf("Could not write data in httptest server, error: %v", err)
		}
	}))
	defer server2.Close()

	cfg := new(config.NucleusConfig)
	cfg.BuildID = "buildID"
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	httpClient := http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	blocklistedEntities := make(map[string][]blocktest)

	type fields struct {
		cfg                 *config.NucleusConfig
		logger              lumber.Logger
		httpClient          http.Client
		endpoint            string
		blocklistedEntities map[string][]blocktest
		errChan             chan error
	}
	type args struct {
		ctx    context.Context
		repoID string
		branch string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"Test fetchBlocklistFromNeuron",
			fields{
				cfg:                 cfg,
				logger:              logger,
				httpClient:          httpClient,
				endpoint:            server.URL + "/testBlocklist.json",
				blocklistedEntities: blocklistedEntities,
				errChan:             make(chan error, 1),
			},
			args{
				ctx:    context.TODO(),
				repoID: "repoID",
				branch: "branch",
			},
			false,
		},

		{"Test fetchBlocklistFromNeuron for wrong request endpoint",
			fields{
				cfg:                 cfg,
				logger:              logger,
				httpClient:          httpClient,
				endpoint:            "/dne.json",
				blocklistedEntities: blocklistedEntities,
				errChan:             make(chan error, 1),
			},
			args{
				ctx:    context.TODO(),
				repoID: "repoID",
				branch: "branch",
			},
			true,
		},

		{"Test fetchBlocklistFromNeuron for non 200 response",
			fields{
				cfg:                 cfg,
				logger:              logger,
				httpClient:          httpClient,
				endpoint:            server2.URL + "/non200",
				blocklistedEntities: blocklistedEntities,
				errChan:             make(chan error, 1),
			},
			args{
				ctx:    context.TODO(),
				repoID: "repoID",
				branch: "branch",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbs := &TestBlockTestService{
				cfg:               tt.fields.cfg,
				logger:            tt.fields.logger,
				httpClient:        tt.fields.httpClient,
				endpoint:          tt.fields.endpoint,
				blockTestEntities: tt.fields.blocklistedEntities,
				once:              sync.Once{},
				errChan:           tt.fields.errChan,
			}
			if err := tbs.fetchBlockListFromNeuron(tt.args.ctx, tt.args.repoID, tt.args.branch); (err != nil) != tt.wantErr {
				t.Errorf("TestBlockListService.fetchBlockListFromNeuron() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTestBlockListService_GetBlockListedTests(t *testing.T) {
	server := httptest.NewServer( // mock server
		http.FileServer(http.Dir("../../testutils/testdata/testblocklistdata/")), // mock data stored at testutils/testdata
	)
	defer server.Close()

	cfg := new(config.NucleusConfig)
	cfg.BuildID = "buildID"
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	tbs, err := NewTestBlockTestService(cfg, logger)
	if err != nil {
		t.Errorf("New TestBlockListService object couldn't be initialised, error: %v", err)
	}

	tbs.endpoint = server.URL + "/testBlocklist.json"

	type args struct {
		ctx       context.Context
		tasConfig *core.TASConfig
		repoID    string
		branch    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test GetBlockListedTests",
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					SmartRun:  false,
					Framework: "jest",
					Blocklist: []string{"src/test/f1.spec.js", "src/test/f2.spec.js"},
					SplitMode: core.TestSplit,
					Tier:      "small"},
				repoID: "/testBlocklist.json"},
			true}, // Will not get error if the test is run in docker container, so test will fail in docker container
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tbs.GetBlockTests(tt.args.ctx, tt.args.tasConfig, tt.args.repoID, tt.args.branch); (err != nil) != tt.wantErr {
				t.Errorf("TestBlockListService.GetBlockListedTests() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTestBlockListService_populateBlockList(t *testing.T) {
	cfg := config.GlobalNucleusConfig
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}
	httpClient := http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	blocklistLocators := []*blocktestLocator{}
	firstLocator := &blocktestLocator{
		Locator: "src/test/api1.js",
		Status:  "quarantined",
	}

	secondLocator := &blocktestLocator{
		Locator: "src/test/api2.js",
		Status:  "blocklisted",
	}
	blocklistLocators = append(blocklistLocators, firstLocator, secondLocator)

	type fields struct {
		cfg                 *config.NucleusConfig
		logger              lumber.Logger
		httpClient          http.Client
		endpoint            string
		blocklistedEntities map[string][]blocktest
		errChan             chan error
	}
	type args struct {
		blocklistSource   string
		blocktestLocators []*blocktestLocator
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"Test populateBlockList",
			fields{
				cfg:        cfg,
				logger:     logger,
				httpClient: httpClient,
				endpoint:   "/blocktest",
				blocklistedEntities: map[string][]blocktest{
					"src/test/api1.js": {
						blocktest{
							Source:  "src",
							Locator: "loc",
							Status:  "blocklisted",
						},
					},
				},
				// once:    sync.Once{},
				errChan: make(chan error, 1)},
			args{
				blocklistSource:   "./",
				blocktestLocators: blocklistLocators,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbs := &TestBlockTestService{
				cfg:               tt.fields.cfg,
				logger:            tt.fields.logger,
				httpClient:        tt.fields.httpClient,
				endpoint:          tt.fields.endpoint,
				blockTestEntities: tt.fields.blocklistedEntities,
				once:              sync.Once{},
				errChan:           tt.fields.errChan,
			}
			tbs.populateBlockList(tt.args.blocklistSource, tt.args.blocktestLocators)

			expected := "map[src/test/api1.js:[{Source:src Locator:loc Status:blocklisted} {Source:./ Locator:src/test/api1.js## Status:quarantined}] src/test/api2.js:[{Source:./ Locator:src/test/api2.js## Status:blocklisted}]]"
			got := fmt.Sprintf("%+v", tbs.blockTestEntities)
			if expected != got {
				t.Errorf("\nexpected: %v\ngot: %v", expected, got)
			}
		})
	}
}
