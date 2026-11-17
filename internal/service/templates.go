package service

import (
	"context"
	"encoding/json"
	"sort"

	"cerebro/internal/domain"
)

type TemplatesService struct {
	templates domain.TemplateAPI
}

func NewTemplatesService(templates domain.TemplateAPI) *TemplatesService {
	return &TemplatesService{templates: templates}
}

func (s *TemplatesService) Templates(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	composable, err := s.templates.GetComposableTemplates(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	legacy, err := s.templates.GetTemplates(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !composable.IsSuccess() && composable.Status != 404 {
		return *composable, nil
	}
	if !legacy.IsSuccess() && legacy.Status != 404 {
		return *legacy, nil
	}

	all := make([]domain.TemplateEntry, 0)
	if composable.IsSuccess() {
		norm, nerr := domain.ComposableTemplates(composable.Body)
		if nerr != nil {
			return domain.ErrorResponse(500, nerr.Error()), nil
		}
		entries, perr := parseTemplateEntries(norm)
		if perr != nil {
			return domain.ErrorResponse(500, perr.Error()), nil
		}
		all = append(all, entries...)
	}
	if legacy.IsSuccess() {
		norm, nerr := domain.Templates(legacy.Body)
		if nerr != nil {
			return domain.ErrorResponse(500, nerr.Error()), nil
		}
		entries, perr := parseTemplateEntries(norm)
		if perr != nil {
			return domain.ErrorResponse(500, perr.Error()), nil
		}
		all = append(all, entries...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Name != all[j].Name {
			return all[i].Name < all[j].Name
		}
		return all[i].Type < all[j].Type
	})
	body, _ := json.Marshal(all)
	return domain.NewResponse(200, body), nil
}

func parseTemplateEntries(body json.RawMessage) ([]domain.TemplateEntry, error) {
	var entries []domain.TemplateEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *TemplatesService) Delete(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	name, err := req.Get("name")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	switch req.GetOpt("type") {
	case "legacy":
		return forward(s.templates.DeleteTemplate(name, ctx, req.Target))
	case "composable":
		return forward(s.templates.DeleteComposableTemplate(name, ctx, req.Target))
	}
	resp, err := s.templates.DeleteComposableTemplate(name, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if resp.Status == 404 {
		return forward(s.templates.DeleteTemplate(name, ctx, req.Target))
	}
	return *resp, nil
}

func (s *TemplatesService) Create(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	name, err := req.Get("name")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	template, err := req.GetObj("template")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	if req.GetOpt("type") == "legacy" {
		return forward(s.templates.CreateTemplate(name, template, ctx, req.Target))
	}
	return forward(s.templates.CreateComposableTemplate(name, template, ctx, req.Target))
}
