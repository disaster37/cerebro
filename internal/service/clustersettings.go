package service

import (
	"context"
	"strings"

	"cerebro/internal/domain"
)

type ClusterSettingsService struct {
	cluster domain.ClusterAPI
}

func NewClusterSettingsService(cluster domain.ClusterAPI) *ClusterSettingsService {
	return &ClusterSettingsService{cluster: cluster}
}

func (s *ClusterSettingsService) Get(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forward(s.cluster.GetClusterSettingsFlat(ctx, req.Target))
}

func (s *ClusterSettingsService) Save(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	settings, err := req.GetObj("settings")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}

	current, err := s.cluster.GetClusterSettingsFlat(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if current.IsSuccess() {
		unknown, verr := domain.ValidateClusterSettingsPayload(settings, current.Body)
		if verr != nil {
			return domain.ErrorResponse(400, verr.Error()), nil
		}
		if len(unknown) > 0 {
			return domain.ErrorResponse(400, "Unknown cluster setting parameter(s): "+strings.Join(unknown, ", ")), nil
		}
	}

	return forward(s.cluster.SaveClusterSettings(settings, ctx, req.Target))
}
