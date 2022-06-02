// Package payloadmanager is used for fetching and validating the nucleus execution payload
package payloadmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

// PayloadManager represents the payload for nucleus
type payloadManager struct {
	logger      lumber.Logger
	azureClient core.AzureClient
	cfg         *config.NucleusConfig
	requests    core.Requests
}

// NewPayloadManger creates and returns a new PayloadManager instance
func NewPayloadManger(azureClient core.AzureClient,
	logger lumber.Logger, cfg *config.NucleusConfig, requests core.Requests) core.PayloadManager {
	return &payloadManager{
		azureClient: azureClient,
		logger:      logger,
		cfg:         cfg,
		requests:    requests,
	}
}

func (pm *payloadManager) FetchPayload(ctx context.Context, payloadAddress string) (*core.Payload, error) {
	if payloadAddress == "" {
		return nil, errors.New("invalid payload address")
	}

	u, err := url.Parse(payloadAddress)
	if err != nil {
		return nil, err
	}
	// string the container name to get blob path
	blobPath := strings.Replace(u.Path, fmt.Sprintf("/%s/", core.PayloadContainer), "", -1)

	sasURL, err := pm.azureClient.GetSASURL(ctx, blobPath, core.PayloadContainer)
	if err != nil {
		return nil, err
	}

	rawBytes, _, err := pm.requests.MakeAPIRequest(ctx, http.MethodGet, sasURL, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	p := new(core.Payload)
	if err := json.Unmarshal(rawBytes, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (pm *payloadManager) ValidatePayload(ctx context.Context, payload *core.Payload) error {
	if payload.RepoLink == "" {
		return errs.ErrInvalidPayload("Missing repo link")
	}

	if payload.RepoSlug == "" {
		return errs.ErrInvalidPayload("Missing repo slug")
	}

	if payload.GitProvider == "" {
		return errs.ErrInvalidPayload("Missing git provider")
	}

	if payload.BuildID == "" {
		return errs.ErrInvalidPayload("Missing BuildID")
	}
	if payload.RepoID == "" {
		return errs.ErrInvalidPayload("Missing RepoID")
	}

	if payload.BranchName == "" {
		return errs.ErrInvalidPayload("Missing Branch Name")
	}

	if payload.OrgID == "" {
		return errs.ErrInvalidPayload("Missing OrgID")
	}

	if payload.TasFileName == "" {
		return errs.ErrInvalidPayload("Missing tas yml filename")
	}

	if pm.cfg.Locators != "" {
		payload.Locators = pm.cfg.Locators
	}

	if pm.cfg.LocatorAddress != "" {
		payload.LocatorAddress = pm.cfg.LocatorAddress
	}
	if payload.BuildTargetCommit == "" {
		return errs.ErrInvalidPayload("Missing build target commit")
	}
	// some checks are removed in case of coverage mode or parsing mode
	if !pm.cfg.CoverageMode {
		if pm.cfg.TaskID == "" {
			return errs.ErrInvalidPayload("Missing taskID in config")
		}
		payload.TaskID = pm.cfg.TaskID
	}

	if payload.EventType != core.EventPush && payload.EventType != core.EventPullRequest {
		return errs.ErrInvalidPayload("Invalid event type")
	}

	if payload.EventType == core.EventPush && len(payload.Commits) == 0 {
		return errs.ErrInvalidPayload("Missing commits error")
	}

	return nil
}
