package domain

import (
	"encoding/json"
	"sort"
)

// TemplateEntry is a unified composable/legacy index template.
type TemplateEntry struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"` // "composable" | "legacy"
	Patterns   []string        `json:"patterns"`
	Order      *int            `json:"order,omitempty"`
	Version    *int            `json:"version,omitempty"`
	DataStream bool            `json:"data_stream,omitempty"`
	Template   json.RawMessage `json:"template"`
}

// Templates normalizes a legacy GET /_template response.
func Templates(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}

	out := make([]TemplateEntry, 0, len(raw))
	for name, tmpl := range raw {
		var meta struct {
			IndexPatterns   []string `json:"index_patterns"`
			TemplatePattern string   `json:"template"`
			Order           *int     `json:"order"`
			Version         *int     `json:"version"`
		}
		_ = json.Unmarshal(tmpl, &meta)
		patterns := meta.IndexPatterns
		if len(patterns) == 0 && meta.TemplatePattern != "" {
			patterns = []string{meta.TemplatePattern}
		}
		out = append(out, TemplateEntry{
			Name:     name,
			Type:     "legacy",
			Patterns: nonNilStrings(patterns),
			Order:    meta.Order,
			Version:  meta.Version,
			Template: tmpl,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return json.Marshal(out)
}

// ComposableTemplates normalizes a GET /_index_template response.
func ComposableTemplates(in json.RawMessage) (json.RawMessage, error) {
	var raw struct {
		IndexTemplates []struct {
			Name          string          `json:"name"`
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}

	out := make([]TemplateEntry, 0, len(raw.IndexTemplates))
	for _, t := range raw.IndexTemplates {
		var meta struct {
			IndexPatterns []string        `json:"index_patterns"`
			Priority      *int            `json:"priority"`
			Version       *int            `json:"version"`
			DataStream    json.RawMessage `json:"data_stream"`
		}
		_ = json.Unmarshal(t.IndexTemplate, &meta)
		out = append(out, TemplateEntry{
			Name:       t.Name,
			Type:       "composable",
			Patterns:   nonNilStrings(meta.IndexPatterns),
			Order:      meta.Priority,
			Version:    meta.Version,
			DataStream: len(meta.DataStream) > 0 && string(meta.DataStream) != "null",
			Template:   t.IndexTemplate,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return json.Marshal(out)
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
