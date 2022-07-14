// Package blocktestservice is used for creating the blocklist file
package blocktestservice

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

const (
	delimiter = "##"
)

// blocktest represents the blocked test suites and test cases.
type blocktest struct {
	Source  string `json:"source"`
	Locator string `json:"locator"`
	Status  string `json:"status"`
}

// blocktestAPIResponse fetch blocked test cases from neuron API
type blocktestAPIResponse struct {
	Name        string `json:"test_name"`
	TestLocator string `json:"test_locator"`
	Status      string `json:"status"`
}

// blocktestLocator stores locator and its status info
type blocktestLocator struct {
	Locator string `json:"locator"`
	Status  string `json:"status"`
}

// TestBlockTestService represents an instance of ConfManager instance
type TestBlockTestService struct {
	cfg               *config.NucleusConfig
	requests          core.Requests
	logger            lumber.Logger
	endpoint          string
	blockTestEntities map[string][]blocktest
	once              sync.Once
	errChan           chan error
}

// NewTestBlockTestService creates and returns a new TestBlockTestService instance
func NewTestBlockTestService(cfg *config.NucleusConfig, requests core.Requests, logger lumber.Logger) *TestBlockTestService {
	return &TestBlockTestService{
		cfg:               cfg,
		logger:            logger,
		requests:          requests,
		endpoint:          global.NeuronHost + "/blocktest",
		blockTestEntities: make(map[string][]blocktest),
		errChan:           make(chan error, 1),
	}
}

func (tbs *TestBlockTestService) fetchBlockListFromNeuron(ctx context.Context, branch string) error {
	var inp []blocktestAPIResponse
	query, headers := utils.GetDefaultQueryAndHeaders()
	query["branch"] = branch

	rawBytes, statusCode, err := tbs.requests.MakeAPIRequest(ctx, http.MethodGet, tbs.endpoint, nil, query, headers)
	if statusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	if jsonErr := json.Unmarshal(rawBytes, &inp); jsonErr != nil {
		tbs.logger.Errorf("Unable to fetch blocklist response: %v", jsonErr)
		return jsonErr
	}
	// populate bl

	blocktestLocators := make([]*blocktestLocator, 0, len(inp))
	for i := range inp {
		blockLocator := new(blocktestLocator)
		blockLocator.Locator = inp[i].TestLocator
		blockLocator.Status = inp[i].Status
		blocktestLocators = append(blocktestLocators, blockLocator)
	}
	tbs.populateBlockList("api", blocktestLocators)
	return nil
}

// GetBlockTests provides list of blocked test cases
func (tbs *TestBlockTestService) GetBlockTests(ctx context.Context, blocklistYAML []string, branch string) error {
	tbs.once.Do(func() {

		blocktestLocators := make([]*blocktestLocator, 0, len(blocklistYAML))
		for _, locator := range blocklistYAML {
			blockLocator := new(blocktestLocator)
			blockLocator.Locator = locator
			blockLocator.Status = string(core.Blocklisted)
			blocktestLocators = append(blocktestLocators, blockLocator)
		}

		tbs.populateBlockList("yml", blocktestLocators)

		if err := tbs.fetchBlockListFromNeuron(ctx, branch); err != nil {
			tbs.logger.Errorf("Unable to fetch remote blocklist: %v. Ignoring remote response", err)
			tbs.errChan <- err
			return
		}
		tbs.logger.Infof("Block tests: %+v", tbs.blockTestEntities)

		// write blocklistest tests on disk
		marshalledBlocklist, err := json.Marshal(tbs.blockTestEntities)
		if err != nil {
			tbs.logger.Errorf("Unable to json marshal blocklist: %+v", err)
			tbs.errChan <- err
			return
		}

		if err = ioutil.WriteFile(global.BlockTestFileLocation, marshalledBlocklist, 0644); err != nil {
			tbs.logger.Errorf("Unable to write blocklist file: %+v", err)
			tbs.errChan <- err
			return
		}
		tbs.blockTestEntities = nil
	})
	select {
	case err := <-tbs.errChan:
		return err
	default:
		return nil
	}
}

func (tbs *TestBlockTestService) populateBlockList(blocktestSource string, blocktestLocators []*blocktestLocator) {
	i := 0
	for _, test := range blocktestLocators {
		// locators must end with delimiter
		if !strings.HasSuffix(test.Locator, delimiter) {
			test.Locator += delimiter
		}
		i = strings.Index(test.Locator, delimiter)
		// TODO: handle duplicate entries and ignore its individual suites or testcases in blocklist if file is blocklisted

		entity := blocktest{Source: blocktestSource, Locator: test.Locator, Status: test.Status}
		if val, ok := tbs.blockTestEntities[test.Locator[:i]]; ok {
			tbs.blockTestEntities[test.Locator[:i]] = append(val, entity)
		} else {
			tbs.blockTestEntities[test.Locator[:i]] = append([]blocktest{},
				blocktest{Source: blocktestSource, Locator: test.Locator, Status: test.Status})
		}
	}
}
