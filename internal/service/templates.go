package service

import (
	"context"

	"cerebro/internal/domain"
)

type TemplatesService struct {
	templates domain.TemplateAPI
}

func NewTemplatesService(templates domain.TemplateAPI) *TemplatesService {
	return &TemplatesService{templates: templates}
}

func (s *TemplatesService) Templates(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.templates.GetTemplates(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	body, err := domain.Templates(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, body), nil
}

func (s *TemplatesService) Delete(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	name, err := req.Get("name")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.templates.DeleteTemplate(name, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
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
	resp, err := s.templates.CreateTemplate(name, template, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
