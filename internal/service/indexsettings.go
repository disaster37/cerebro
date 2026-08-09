package service

import (
	"context"

	"cerebro/internal/domain"
)

type IndexSettingsService struct {
	indices domain.IndexAPI
}

func NewIndexSettingsService(indices domain.IndexAPI) *IndexSettingsService {
	return &IndexSettingsService{indices: indices}
}

func (s *IndexSettingsService) Get(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexSettingsFlat(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *IndexSettingsService) Update(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	settings, err := req.GetObj("settings")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.UpdateIndexSettings(index, settings, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
