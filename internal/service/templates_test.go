package service

import (
	"context"
	"encoding/json"
	"testing"

	"cerebro/internal/domain"
)

type fakeTemplateAPI struct {
	domain.TemplateAPI
	composableResp *domain.Response
	legacyResp     *domain.Response
	createdName    string
	createdBody    json.RawMessage
	createResp     *domain.Response
}

func (f *fakeTemplateAPI) GetComposableTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.composableResp, nil
}

func (f *fakeTemplateAPI) GetTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.legacyResp, nil
}

func (f *fakeTemplateAPI) CreateComposableTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	f.createdName = name
	f.createdBody = template
	return f.createResp, nil
}

func TestTemplatesServiceNormalizesBothShapes(t *testing.T) {
	fake := &fakeTemplateAPI{
		composableResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"index_templates":[
			{"name":"logs-ds","index_template":{"index_patterns":["logs-*"],"priority":200,"data_stream":{}}}
		]}`)},
		legacyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{
			"legacy":{"template":"metrics-*","order":1}
		}`)},
	}
	svc := NewTemplatesService(fake)
	resp, err := svc.Templates(context.Background(), domain.CerebroRequest{})
	if err != nil {
		t.Fatalf("Templates: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
	}
	var entries []domain.TemplateEntry
	if err := json.Unmarshal(resp.Body, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	// Deterministic (Name, Type) ordering.
	if entries[0].Name != "legacy" || entries[1].Name != "logs-ds" {
		t.Errorf("order = %s, %s", entries[0].Name, entries[1].Name)
	}
	if !entries[1].DataStream {
		t.Errorf("composable data stream flag = false, want true")
	}
	if len(entries[0].Patterns) != 1 || entries[0].Patterns[0] != "metrics-*" {
		t.Errorf("legacy patterns = %v, want [metrics-*]", entries[0].Patterns)
	}
}

func TestTemplatesServiceTolerates404(t *testing.T) {
	fake := &fakeTemplateAPI{
		composableResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":"not found"}`)},
		legacyResp:     &domain.Response{Status: 200, Body: json.RawMessage(`{"legacy":{"index_patterns":["a-*"]}}`)},
	}
	svc := NewTemplatesService(fake)
	resp, err := svc.Templates(context.Background(), domain.CerebroRequest{})
	if err != nil {
		t.Fatalf("Templates: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	var entries []domain.TemplateEntry
	_ = json.Unmarshal(resp.Body, &entries)
	if len(entries) != 1 || entries[0].Name != "legacy" {
		t.Errorf("entries = %+v, want only legacy", entries)
	}
}
