// Package blocktestservice is used for creating the blocklist file
package blocktestservice

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

const (
	delimiter = "##"
)

// blocktest represents the blocklisted test suites and test cases.
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

// TestBlockListService represents an instance of ConfManager instance
type TestBlockListService struct {
	cfg               *config.NucleusConfig
	logger            lumber.Logger
	httpClient        http.Client
	endpoint          string
	blockTestEntities map[string][]blocktest
	once              sync.Once
	errChan           chan error
}

// NewTestBlockListService creates and returns a new TestBlockListService instance
func NewTestBlockListService(cfg *config.NucleusConfig, logger lumber.Logger) (*TestBlockListService, error) {

	return &TestBlockListService{
		cfg:               cfg,
		logger:            logger,
		endpoint:          global.NeuronHost + "/blocktest",
		blockTestEntities: make(map[string][]blocktest),
		errChan:           make(chan error, 1),
		httpClient: http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		}}, nil
}

func (tbs *TestBlockListService) fetchBlockListFromNeuron(ctx context.Context, repoID, branch string) error {
	var inp []blocktestAPIResponse
	u, err := url.Parse(tbs.endpoint)
	if err != nil {
		tbs.logger.Errorf("error while parsing endpoint %s, %v", tbs.endpoint, err)
		return err
	}
	q := u.Query()
	q.Set("repoID", repoID)
	q.Set("branch", branch)
	q.Set("buildID", tbs.cfg.BuildID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		tbs.logger.Errorf("Unable to fetch blocklist response: %+v", err)
		return err
	}

	resp, err := tbs.httpClient.Do(req)
	if err != nil {
		tbs.logger.Errorf("Unable to fetch blocklist response: %v", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		err = errors.New("non 200 status")
		tbs.logger.Errorf("Unable to fetch blocklist response: %v", err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		tbs.logger.Errorf("Unable to fetch blocklist response: %v", err)
		return err
	}

	if jsonErr := json.Unmarshal(body, &inp); jsonErr != nil {
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
func (tbs *TestBlockListService) GetBlockTests(ctx context.Context, tasConfig *core.TASConfig, repoID, branch string) error {

	tbs.once.Do(func() {

		blocktestLocators := make([]*blocktestLocator, 0, len(tasConfig.Blocklist))
		for _, locator := range tasConfig.Blocklist {
			blockLocator := new(blocktestLocator)
			blockLocator.Locator = locator
			blockLocator.Status = string(core.Blocklisted)
			blocktestLocators = append(blocktestLocators, blockLocator)
		}

		tbs.populateBlockList("yml", blocktestLocators)

		if err := tbs.fetchBlockListFromNeuron(ctx, repoID, branch); err != nil {
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

func (tbs *TestBlockListService) populateBlockList(blocktestSource string, blocktestLocators []*blocktestLocator) {

	i := 0
	for _, test := range blocktestLocators {
		//locators must end with delimiter
		if !strings.HasSuffix(test.Locator, delimiter) {
			test.Locator += delimiter
		}
		i = strings.Index(test.Locator, delimiter)
		//TODO: handle duplicate entries and ignore its individual suites or testcases in blocklist if file is blocklisted

		if val, ok := tbs.blockTestEntities[test.Locator[:i]]; ok {
			tbs.blockTestEntities[test.Locator[:i]] = append(val, blocktest{Source: blocktestSource, Locator: test.Locator, Status: test.Status})
		} else {
			tbs.blockTestEntities[test.Locator[:i]] = append([]blocktest{}, blocktest{Source: blocktestSource, Locator: test.Locator, Status: test.Status})
		}
	}
}
