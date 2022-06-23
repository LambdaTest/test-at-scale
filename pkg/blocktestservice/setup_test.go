// Package blocktestservice is used for creating the blocklist file
package blocktestservice

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/cenkalti/backoff/v4"
)

const buildID = "buildID"

func TestBlockListService_fetchBlockListFromNeuron(t *testing.T) {
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
	cfg.BuildID = buildID
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	blocklistedEntities := make(map[string][]blocktest)

	type args struct {
		ctx      context.Context
		endpoint string
		repoID   string
		branch   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test fetchBlocklistFromNeuron",
			args{
				ctx:      context.TODO(),
				endpoint: server.URL + "/testBlocklist.json",
				repoID:   "repoID",
				branch:   "branch",
			},
			false,
		},

		{"Test fetchBlocklistFromNeuron for wrong request endpoint",
			args{
				ctx:      context.TODO(),
				endpoint: "/dne.json",
				repoID:   "repoID",
				branch:   "branch",
			},
			true,
		},

		{"Test fetchBlocklistFromNeuron for non 200 response",
			args{
				ctx:      context.TODO(),
				endpoint: server2.URL + "/non200",
				repoID:   "repoID",
				branch:   "branch",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tbs := &TestBlockTestService{
				cfg:               cfg,
				logger:            logger,
				endpoint:          tt.args.endpoint,
				blockTestEntities: blocklistedEntities,
				once:              sync.Once{},
				errChan:           make(chan error, 1),
				requests:          requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{}),
			}
			if err := tbs.fetchBlockListFromNeuron(tt.args.ctx, tt.args.branch); (err != nil) != tt.wantErr {
				t.Errorf("TestBlockListService.fetchBlockListFromNeuron() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockListService_GetBlockListedTests(t *testing.T) {
	server := httptest.NewServer( // mock server
		http.FileServer(http.Dir("../../testutils/testdata/testblocklistdata/")), // mock data stored at testutils/testdata
	)
	defer server.Close()

	cfg := new(config.NucleusConfig)
	cfg.BuildID = buildID
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	requests := requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{})
	tbs := NewTestBlockTestService(cfg, requests, logger)

	tbs.endpoint = server.URL + "/testBlocklist.json"

	type args struct {
		ctx       context.Context
		tasConfig *core.TASConfig
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
			},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blYML := tt.args.tasConfig.Blocklist
			if err := tbs.GetBlockTests(tt.args.ctx, blYML, tt.args.branch); (err != nil) != tt.wantErr {
				t.Errorf("TestBlockListService.GetBlockListedTests() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockListService_populateBlockList(t *testing.T) {
	cfg := config.GlobalNucleusConfig
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
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
				cfg:      cfg,
				logger:   logger,
				endpoint: "/blocktest",
				blocklistedEntities: map[string][]blocktest{
					"src/test/api1.js": {
						blocktest{
							Source:  "src",
							Locator: "loc",
							Status:  "blocklisted",
						},
					},
				},
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
				endpoint:          tt.fields.endpoint,
				blockTestEntities: tt.fields.blocklistedEntities,
				once:              sync.Once{},
				errChan:           tt.fields.errChan,
			}
			tbs.populateBlockList(tt.args.blocklistSource, tt.args.blocktestLocators)
			expected := map[string][]blocktest{"src/test/api1.js": {{"src", "loc", "blocklisted"}, {"./", "src/test/api1.js##", "quarantined"}},
				"src/test/api2.js": {{"./", "src/test/api2.js##", "blocklisted"}}}
			got := tbs.blockTestEntities
			if !reflect.DeepEqual(expected, got) {
				t.Errorf("\nexpected: %v\ngot: %v", expected, got)
			}
		})
	}
}
