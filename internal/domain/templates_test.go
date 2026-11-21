package domain

import (
	"encoding/json"
	"testing"
)

func TestComposableTemplatesDataStreamFlag(t *testing.T) {
	raw := json.RawMessage(`{"index_templates":[
		{"name":"logs-ds","index_template":{"index_patterns":["logs-*"],"priority":200,"data_stream":{}}},
		{"name":"plain","index_template":{"index_patterns":["plain-*"],"priority":100}}
	]}`)
	body, err := ComposableTemplates(raw)
	if err != nil {
		t.Fatalf("ComposableTemplates: %v", err)
	}
	var entries []TemplateEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	byName := map[string]TemplateEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if !byName["logs-ds"].DataStream {
		t.Errorf("logs-ds data_stream = false, want true")
	}
	if byName["plain"].DataStream {
		t.Errorf("plain data_stream = true, want false")
	}
	if byName["logs-ds"].Type != "composable" || byName["logs-ds"].Order == nil || *byName["logs-ds"].Order != 200 {
		t.Errorf("logs-ds entry = %+v", byName["logs-ds"])
	}
}

func TestLegacyTemplatesTemplateStringFallback(t *testing.T) {
	raw := json.RawMessage(`{
		"legacy-pattern":{"template":"logs-*","order":1},
		"legacy-patterns":{"index_patterns":["metrics-*"],"order":2}
	}`)
	body, err := Templates(raw)
	if err != nil {
		t.Fatalf("Templates: %v", err)
	}
	var entries []TemplateEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	byName := map[string]TemplateEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if got := byName["legacy-pattern"].Patterns; len(got) != 1 || got[0] != "logs-*" {
		t.Errorf("legacy template string fallback = %v, want [logs-*]", got)
	}
	if got := byName["legacy-patterns"].Patterns; len(got) != 1 || got[0] != "metrics-*" {
		t.Errorf("legacy index_patterns = %v, want [metrics-*]", got)
	}
}
