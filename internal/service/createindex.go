package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

type CreateIndexService struct {
	indices domain.IndexAPI
}

func NewCreateIndexService(indices domain.IndexAPI) *CreateIndexService {
	return &CreateIndexService{indices: indices}
}

func (s *CreateIndexService) GetIndexMetadata(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexMetadata(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.IndexMetadata(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *CreateIndexService) Execute(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	metadata := req.GetObjOpt("metadata")
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}
	resp, err := s.indices.CreateIndex(index, metadata, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
