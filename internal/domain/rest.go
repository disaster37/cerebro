package domain

import (
	"encoding/json"
	"sort"
)

// AutocompletionIndices merges index names, aliases and (optionally) data
// stream names into the DevTools autocomplete index list.
func AutocompletionIndices(aliases json.RawMessage, dataStreams ...json.RawMessage) (json.RawMessage, error) {
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
	if len(dataStreams) > 0 {
		streams, _, err := ParseDataStreams(dataStreams[0])
		if err == nil {
			for _, ds := range streams {
				if !seen[ds.Name] {
					seen[ds.Name] = true
					out = append(out, ds.Name)
				}
			}
		}
	}
	sort.Strings(out)
	return json.Marshal(out)
}

// ExtractMappingFields walks a GET /_mapping response and returns the
// deduplicated, sorted field paths per index plus the union of all fields.
// When backing-index metadata is supplied, fields from `.ds-*` backing indices
// are also aggregated under their parent data stream name.
func ExtractMappingFields(mappingJSON json.RawMessage, backingMeta ...map[string]DataStreamBackingMeta) (map[string][]string, []string, error) {
	var raw map[string]struct {
		Mappings json.RawMessage `json:"mappings"`
	}
	if err := json.Unmarshal(mappingJSON, &raw); err != nil {
		return nil, nil, err
	}

	var meta map[string]DataStreamBackingMeta
	if len(backingMeta) > 0 {
		meta = backingMeta[0]
	}

	fieldsByIndex := make(map[string][]string, len(raw))
	allSet := map[string]bool{}
	streamFields := map[string]map[string]bool{}
	for index, entry := range raw {
		var mappings any
		_ = json.Unmarshal(entry.Mappings, &mappings)
		fields := collectMappingFields(mappings, "")
		sort.Strings(fields)
		fieldsByIndex[index] = fields
		for _, f := range fields {
			allSet[f] = true
		}
		if ds, ok := meta[index]; ok && ds.DataStream != "" {
			if streamFields[ds.DataStream] == nil {
				streamFields[ds.DataStream] = map[string]bool{}
			}
			for _, f := range fields {
				streamFields[ds.DataStream][f] = true
			}
		}
	}
	for stream, set := range streamFields {
		fields := make([]string, 0, len(set))
		for f := range set {
			fields = append(fields, f)
		}
		sort.Strings(fields)
		fieldsByIndex[stream] = fields
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
