package service

import (
	"context"
	"encoding/json"

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
	return transform(resp, err, domain.OpenIndices)
}

func (s *AnalysisService) GetIndexAnalyzers(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexSettings(index, ctx, req.Target)
	return transform(resp, err, func(body json.RawMessage) (json.RawMessage, error) {
		return domain.IndexAnalyzers(index, body)
	})
}

func (s *AnalysisService) GetIndexFields(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.GetIndexMapping(index, ctx, req.Target)
	return transform(resp, err, func(body json.RawMessage) (json.RawMessage, error) {
		return domain.IndexFields(index, body)
	})
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
	return transform(resp, err, domain.Tokens)
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
	return transform(resp, err, domain.Tokens)
}
