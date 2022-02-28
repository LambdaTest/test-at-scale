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
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

// PayloadManager represents the payload for nucleus
type payloadManager struct {
	logger      lumber.Logger
	httpClient  http.Client
	azureClient core.AzureClient
	cfg         *config.NucleusConfig
}

// NewPayloadManger creates and returns a new PayloadManager instance
func NewPayloadManger(azureClient core.AzureClient,
	logger lumber.Logger, cfg *config.NucleusConfig) core.PayloadManager {
	pm := payloadManager{
		azureClient: azureClient,
		logger:      logger,
		httpClient: http.Client{
			Timeout: 30 * time.Second,
		},
		cfg: cfg,
	}

	return &pm
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sasURL, nil)
	if err != nil {
		return nil, err
	}

	r, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	var p core.Payload
	err = json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil

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
	if !(pm.cfg.CoverageMode || pm.cfg.ParseMode) {
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
