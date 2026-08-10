package service

import (
	"context"

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
	return forward(s.cluster.SaveClusterSettings(settings, ctx, req.Target))
}
