package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

// parserService represents the yml parser object
type parserService struct {
	logger           lumber.Logger
	tasConfigManager core.TASConfigManager
	httpClient       http.Client
	endpoint         string
}

var tierEnumMapping = map[core.Tier]int{
	core.XSmall: 1,
	core.Small:  2,
	core.Medium: 3,
	core.Large:  4,
	core.XLarge: 5,
}

//New returns a new  YMLParserService
func New(tasConfigManager core.TASConfigManager, logger lumber.Logger) core.YMLParserService {
	return &parserService{
		logger:           logger,
		tasConfigManager: tasConfigManager,
		endpoint:         global.NeuronHost + "/ymlparser",
		httpClient: http.Client{
			Timeout: global.DefaultHTTPTimeout,
		},
	}
}

func (p *parserService) ParseAndValidate(ctx context.Context, payload *core.Payload) error {
	targetCommit := payload.BuildTargetCommit
	parserPayloadStatus := &core.ParserStatus{
		TargetCommitID: targetCommit,
		BaseCommitID:   payload.BuildBaseCommit,
		Status:         core.Passed,
	}

	if tasConfig, err := p.tasConfigManager.LoadConfig(ctx, payload.TasFileName, payload.EventType, true); err != nil {
		p.logger.Infof("Parsing failed for commitID: %s, buildID: %s, error: %v", targetCommit, payload.BuildID, err)
		parserPayloadStatus.Status = core.Error
		parserPayloadStatus.Message = err.Error()
	} else {
		parserPayloadStatus.Tier = tasConfig.Tier
		if _, err := isValidLicenseTier(tasConfig.Tier, payload.LicenseTier); err != nil {
			p.logger.Errorf("LicenseTier validation failed for commitID: %s, buildID: %s, error: %v", targetCommit, payload.BuildID, err)
			parserPayloadStatus.Status = core.Error
			parserPayloadStatus.Message = err.Error()
		}
	}
	parserResult := &core.ParserResponse{
		BuildID:     payload.BuildID,
		RepoID:      payload.RepoID,
		OrgID:       payload.OrgID,
		GitProvider: payload.GitProvider,
		RepoSlug:    payload.RepoSlug,
		Status:      parserPayloadStatus,
	}

	if err := p.sendParserResponse(parserResult); err != nil {
		p.logger.Errorf("Parsing API failed to send data, for commitID: %s, buildID: %s, error: %v", targetCommit, payload.BuildID, err)
		return err
	}
	return nil
}

func (p *parserService) sendParserResponse(payload *core.ParserResponse) error {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		p.logger.Errorf("failed to marshal request body %v", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, p.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		p.logger.Errorf("failed to create new request %v", err)
		return err
	}

	resp, err := p.httpClient.Do(req)

	if err != nil {
		p.logger.Errorf("error while sending parser response data %v", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.logger.Errorf("error while sending parser data, status code %d", resp.StatusCode)
		return errors.New("non 200 status")
	}
	return nil
}

func isValidLicenseTier(yamlLicense, currentLicense core.Tier) (bool, error) {
	if tierEnumMapping[yamlLicense] > tierEnumMapping[currentLicense] {
		return false, fmt.Errorf("Tier must not exceed max tier in license %v", currentLicense)
	}
	return true, nil
}
