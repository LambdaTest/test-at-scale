package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

// Parser represents the code parser object
type Parser struct {
	ctx              context.Context
	logger           lumber.Logger
	TASConfigManager core.TASConfigManager
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

//New returns a new instance of Parser
func New(ctx context.Context, TASConfigManager core.TASConfigManager,
	logger lumber.Logger) (*Parser, error) {
	return &Parser{
		logger:           logger,
		ctx:              ctx,
		TASConfigManager: TASConfigManager,
		endpoint:         global.NeuronHost + "/ymlparser",
		httpClient: http.Client{
			Timeout: 30 * time.Second,
		}}, nil

}

// PerformParsing parses the YML file and returns error if there are any
func (p *Parser) PerformParsing(payload *core.Payload) error {
	targetCommit := payload.BuildTargetCommit
	parserPayloadStatus := &core.ParserStatus{
		TargetCommitID: targetCommit,
		BaseCommitID:   payload.BuildBaseCommit,
		Status:         core.Passed,
	}

	if tasConfig, err := p.TASConfigManager.LoadConfig(p.ctx,
		targetCommit+payload.TasFileName, payload.EventType, true); err != nil {
		p.logger.Infof("Parsing failed for commitID: %s, buildID: %s, error: %v", targetCommit, payload.BuildID, err)
		parserPayloadStatus.Status = core.Error
		parserPayloadStatus.Message = err.Error()
	} else {
		parserPayloadStatus.Tier = tasConfig.Tier
		parserPayloadStatus.ContainerImage = tasConfig.ContainerImage
		if _, err := isValidLicenseTier(tasConfig.Tier, payload.LicenseTier); err != nil {
			p.logger.Errorf("LicenseTier validation failed error:%v", err)
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
		p.logger.Errorf("Parsing API failed to send data, error: %v", err)
		return err
	}
	return nil
}

func (p *Parser) sendParserResponse(payload *core.ParserResponse) error {

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
		errorMsg := fmt.Sprintf("Tier must not exceed max tier in license %v", currentLicense)
		return false, errors.New(errorMsg)
	}
	return true, nil
}
