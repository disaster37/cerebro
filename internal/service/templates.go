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
	return transform(resp, err, domain.Templates)
}

func (s *TemplatesService) Delete(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "name", s.templates.DeleteTemplate)
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
	return forward(s.templates.CreateTemplate(name, template, ctx, req.Target))
}
