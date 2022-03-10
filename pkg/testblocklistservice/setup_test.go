// Package testblocklistservice is used for creating the blocklist file
package testblocklistservice

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/testutils"
)

func TestNewTestBlockListService(t *testing.T) {
	cfg := config.GlobalNucleusConfig
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}

	want := TestBlockListService{
		cfg:                 cfg,
		logger:              logger,
		endpoint:            "endpoint",
		blocklistedEntities: make(map[string][]blocklist),
		errChan:             make(chan error, 1),
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
		want    *TestBlockListService
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
			_, err := NewTestBlockListService(tt.args.cfg, tt.args.logger)
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
	blocklistedEntities := make(map[string][]blocklist)

	type fields struct {
		cfg                 *config.NucleusConfig
		logger              lumber.Logger
		httpClient          http.Client
		endpoint            string
		blocklistedEntities map[string][]blocklist
		errChan             chan error
	}
	type args struct {
		ctx    context.Context
		repoID string
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
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbs := &TestBlockListService{
				cfg:                 tt.fields.cfg,
				logger:              tt.fields.logger,
				httpClient:          tt.fields.httpClient,
				endpoint:            tt.fields.endpoint,
				blocklistedEntities: tt.fields.blocklistedEntities,
				once:                sync.Once{},
				errChan:             tt.fields.errChan,
			}
			if err := tbs.fetchBlockListFromNeuron(tt.args.ctx, tt.args.repoID); (err != nil) != tt.wantErr {
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

	cfg := config.GlobalNucleusConfig
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialise logger, error: %v", err)
	}

	tbs, err := NewTestBlockListService(cfg, logger)
	if err != nil {
		t.Errorf("New TestBlockListService object couldn't be initialised, error: %v", err)
	}

	tbs.endpoint = server.URL + "/testBlocklist.json"

	type args struct {
		ctx       context.Context
		tasConfig *core.TASConfig
		repoID    string
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
					Tier:      "small"},
				repoID: "/testBlocklist.json"},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tbs.GetBlockListedTests(tt.args.ctx, tt.args.tasConfig, tt.args.repoID); (err != nil) != tt.wantErr {
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

	type fields struct {
		cfg                 *config.NucleusConfig
		logger              lumber.Logger
		httpClient          http.Client
		endpoint            string
		blocklistedEntities map[string][]blocklist
		errChan             chan error
	}
	type args struct {
		blocklistSource   string
		blocklistLocators []string
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
				endpoint:   "/blocklist",
				blocklistedEntities: map[string][]blocklist{
					"src/test/api1.js": {
						blocklist{
							Source:  "src",
							Locator: "loc",
						},
					},
				},
				// once:    sync.Once{},
				errChan: make(chan error, 1)},
			args{
				blocklistSource:   "./",
				blocklistLocators: []string{"src/test/api1.js", "src/test/api2.js##"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbs := &TestBlockListService{
				cfg:                 tt.fields.cfg,
				logger:              tt.fields.logger,
				httpClient:          tt.fields.httpClient,
				endpoint:            tt.fields.endpoint,
				blocklistedEntities: tt.fields.blocklistedEntities,
				once:                sync.Once{},
				errChan:             tt.fields.errChan,
			}
			tbs.populateBlockList(tt.args.blocklistSource, tt.args.blocklistLocators)

			expected := "map[src/test/api1.js:[{Source:src Locator:loc} {Source:./ Locator:src/test/api1.js##}] src/test/api2.js:[{Source:./ Locator:src/test/api2.js##}]]"
			got := fmt.Sprintf("%+v", tbs.blocklistedEntities)
			if expected != got {
				t.Errorf("\nexpected: %v\ngot: %v", expected, got)
			}
		})
	}
}
