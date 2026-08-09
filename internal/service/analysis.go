package service

import (
	"context"

	"cerebro/internal/domain"
)

type AnalysisService struct {
	indices  domain.IndexAPI
	analysis domain.AnalysisAPI
}

func NewAnalysisService(indices domain.IndexAPI, analysis domain.AnalysisAPI) *AnalysisService {
	return &AnalysisService{indices: indices, analysis: analysis}
}

func (s *AnalysisService) GetIndices(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.indices.GetIndices(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.OpenIndices(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *AnalysisService) GetIndexAnalyzers(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexSettings(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.IndexAnalyzers(index, resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *AnalysisService) GetIndexFields(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexMapping(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.IndexFields(index, resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *AnalysisService) AnalyzeByField(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	field, err := req.Get("field")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	text, err := req.Get("text")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.analysis.AnalyzeTextByField(index, field, text, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.Tokens(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *AnalysisService) AnalyzeByAnalyzer(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	analyzer, err := req.Get("analyzer")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	text, err := req.Get("text")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.analysis.AnalyzeTextByAnalyzer(index, analyzer, text, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.Tokens(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}
