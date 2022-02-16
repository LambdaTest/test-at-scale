// Package testblocklistservice is used for creating the blocklist file
package testblocklistservice

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

//blocklist represents the blocklisted test suites and test cases.
type blocklist struct {
	Source  string `json:"source"`
	Locator string `json:"locator"`
}

// fetch blocklisted test cases from neuron API
type blocklistResponse struct {
	Name        string `json:"name"`
	Repo        string `json:"repo"`
	TestLocator string `json:"test_locator"`
}

// TestBlockListService represents an instance of ConfManager instance
type TestBlockListService struct {
	cfg                 *config.NucleusConfig
	logger              lumber.Logger
	httpClient          http.Client
	endpoint            string
	blocklistedEntities map[string][]blocklist
	once                sync.Once
	errChan             chan error
}

// NewTestBlockListService creates and returns a new TestBlockListService instance
func NewTestBlockListService(cfg *config.NucleusConfig, logger lumber.Logger) (*TestBlockListService, error) {

	return &TestBlockListService{
		cfg:                 cfg,
		logger:              logger,
		endpoint:            global.NeuronHost + "/blocklist",
		blocklistedEntities: make(map[string][]blocklist),
		errChan:             make(chan error, 1),
		httpClient: http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		}}, nil
}

//fetchBlockListFromNeuron
func (tbs *TestBlockListService) fetchBlockListFromNeuron(ctx context.Context, repoID string) error {

	var inp []blocklistResponse

	u, err := url.Parse(tbs.endpoint)
	if err != nil {
		tbs.logger.Errorf("error while parsing endpoint %s, %v", tbs.endpoint, err)
		return err
	}
	q := u.Query()
	q.Set("repoID", repoID)
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

	locators := make([]string, 0, len(inp))
	for i := range inp {
		locators = append(locators, inp[i].TestLocator)
	}
	tbs.populateBlockList("api", locators)
	return nil
}

// GetBlockListedTests provides list of blocklisted test cases
func (tbs *TestBlockListService) GetBlockListedTests(ctx context.Context, tasConfig *core.TASConfig, repoID string) error {

	tbs.once.Do(func() {
		tbs.populateBlockList("yml", tasConfig.Blocklist)

		if err := tbs.fetchBlockListFromNeuron(ctx, repoID); err != nil {
			tbs.logger.Errorf("Unable to fetch remote blocklist: %v. Ignoring remote response", err)
			tbs.errChan <- err
			return
		}
		tbs.logger.Infof("Blocklisted tests: %+v", tbs.blocklistedEntities)

		// write blocklistest tests on disk
		marshalledBlocklist, err := json.Marshal(tbs.blocklistedEntities)
		if err != nil {
			tbs.logger.Errorf("Unable to json marshal blocklist: %+v", err)
			tbs.errChan <- err
			return
		}

		if err = ioutil.WriteFile(global.BlocklistedFileLocation, marshalledBlocklist, 0644); err != nil {
			tbs.logger.Errorf("Unable to write blocklist file: %+v", err)
			tbs.errChan <- err
			return
		}
		tbs.blocklistedEntities = nil
	})
	select {
	case err := <-tbs.errChan:
		return err
	default:
		return nil
	}
}

func (tbs *TestBlockListService) populateBlockList(blocklistSource string, blocklistLocators []string) {

	i := 0
	for _, locator := range blocklistLocators {

		//locators must end with delimiter
		if !strings.HasSuffix(locator, delimiter) {
			locator += delimiter
		}
		i = strings.Index(locator, delimiter)
		//TODO: handle duplicate entries and ignore its individual suites or testcases in blocklist if file is blocklisted

		if val, ok := tbs.blocklistedEntities[locator[:i]]; ok {
			tbs.blocklistedEntities[locator[:i]] = append(val, blocklist{Source: blocklistSource, Locator: locator})
		} else {
			tbs.blocklistedEntities[locator[:i]] = append([]blocklist{}, blocklist{Source: blocklistSource, Locator: locator})
		}
	}
}
