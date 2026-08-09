package service

import (
	"context"

	"cerebro/internal/domain"
)

type CommonsService struct {
	indices domain.IndexAPI
	nodes   domain.NodeAPI
}

func NewCommonsService(indices domain.IndexAPI, nodes domain.NodeAPI) *CommonsService {
	return &CommonsService{indices: indices, nodes: nodes}
}

func (s *CommonsService) Indices(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.indices.GetIndices(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.CommonsIndices(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *CommonsService) Nodes(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.nodes.GetNodes(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.CommonsNodes(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *CommonsService) GetIndexSettings(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexSettings(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *CommonsService) GetIndexMappings(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexMapping(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *CommonsService) GetNodeStats(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	node, err := req.Get("node")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.nodes.NodeStats(node, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *CommonsService) GetIndexStats(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.IndexStats(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
