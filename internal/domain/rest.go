package domain

import (
	"encoding/json"
	"sort"
)

func AutocompletionIndices(aliases json.RawMessage) (json.RawMessage, error) {
	var raw map[string]struct {
		Aliases map[string]json.RawMessage `json:"aliases"`
	}
	if err := json.Unmarshal(aliases, &raw); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for idx, data := range raw {
		if !seen[idx] {
			seen[idx] = true
			out = append(out, idx)
		}
		for alias := range data.Aliases {
			if !seen[alias] {
				seen[alias] = true
				out = append(out, alias)
			}
		}
	}
	return json.Marshal(out)
}

// ExtractMappingFields walks a GET /_mapping response and returns the
// deduplicated, sorted field paths per index plus the union of all fields.
func ExtractMappingFields(mappingJSON json.RawMessage) (map[string][]string, []string, error) {
	var raw map[string]struct {
		Mappings json.RawMessage `json:"mappings"`
	}
	if err := json.Unmarshal(mappingJSON, &raw); err != nil {
		return nil, nil, err
	}

	fieldsByIndex := make(map[string][]string, len(raw))
	allSet := map[string]bool{}
	for index, entry := range raw {
		var mappings any
		_ = json.Unmarshal(entry.Mappings, &mappings)
		fields := collectMappingFields(mappings, "")
		sort.Strings(fields)
		fieldsByIndex[index] = fields
		for _, f := range fields {
			allSet[f] = true
		}
	}

	allFields := make([]string, 0, len(allSet))
	for f := range allSet {
		allFields = append(allFields, f)
	}
	sort.Strings(allFields)
	return fieldsByIndex, allFields, nil
}

func collectMappingFields(v any, prefix string) []string {
	m, ok := v.(map[string]any)
	if !ok {
		return []string{}
	}
	props, ok := m["properties"].(map[string]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0)
	for name, raw := range props {
		field := name
		if prefix != "" {
			field = prefix + "." + name
		}
		sub, _ := raw.(map[string]any)
		if sub == nil {
			out = append(out, field)
			continue
		}
		if nested, ok := sub["properties"]; ok {
			out = append(out, collectMappingFields(map[string]any{"properties": nested}, field)...)
		} else {
			out = append(out, field)
		}
		if multi, ok := sub["fields"].(map[string]any); ok {
			for multiName := range multi {
				out = append(out, field+"."+multiName)
			}
		}
	}
	return out
}
