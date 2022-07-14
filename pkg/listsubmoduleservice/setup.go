package listsubmoduleservice

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

type subModuleListService struct {
	logger                lumber.Logger
	requests              core.Requests
	subModuleListEndpoint string
}

func New(request core.Requests, logger lumber.Logger) core.ListSubModuleService {
	return &subModuleListService{
		logger:                logger,
		requests:              request,
		subModuleListEndpoint: global.NeuronHost + "/submodule-list",
	}
}

func (s *subModuleListService) Send(ctx context.Context, buildID string, totalSubmodule int) error {
	subModuleList := core.SubModuleList{
		BuildID:        buildID,
		TotalSubModule: totalSubmodule,
	}
	reqBody, err := json.Marshal(&subModuleList)
	if err != nil {
		s.logger.Errorf("error while json marshal %v", err)
		return err
	}
	query, headers := utils.GetDefaultQueryAndHeaders()
	if _, statusCode, err := s.requests.MakeAPIRequest(ctx, http.MethodPost, s.subModuleListEndpoint,
		reqBody, query, headers); err != nil || statusCode != 200 {
		s.logger.Errorf("error while making submodule-list api call status code %d, err %v", statusCode, err)
		return err
	}
	return nil
}
