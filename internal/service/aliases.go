package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

type AliasesService struct {
	aliases domain.AliasAPI
}

func NewAliasesService(aliases domain.AliasAPI) *AliasesService {
	return &AliasesService{aliases: aliases}
}

func (s *AliasesService) GetAliases(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.aliases.GetAliases(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	aliases, err := domain.Aliases(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, aliases), nil
}

func (s *AliasesService) UpdateAliases(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(req.Body, &m)
	changes := json.RawMessage("[]")
	if cg, ok := m["changes"]; ok {
		changes = cg
	}
	resp, err := s.aliases.UpdateAliases(changes, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
