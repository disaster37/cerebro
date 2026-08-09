package domain

import "encoding/json"

func OpenIndices(in json.RawMessage) (json.RawMessage, error) {
	var arr []struct {
		Status string `json:"status"`
		Index  string `json:"index"`
	}
	if err := json.Unmarshal(in, &arr); err != nil {
		return nil, err
	}
	var out []string
	for _, e := range arr {
		if e.Status == "open" {
			out = append(out, e.Index)
		}
	}
	return json.Marshal(out)
}

func IndexAnalyzers(index string, settings json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(settings, &raw); err != nil {
		return json.RawMessage("[]"), nil
	}
	idxData, ok := raw[index]
	if !ok {
		return json.RawMessage("[]"), nil
	}
	var idx struct {
		Settings struct {
			Index struct {
				Analysis struct {
					Analyzer json.RawMessage `json:"analyzer"`
				} `json:"analysis"`
			} `json:"index"`
		} `json:"settings"`
	}
	_ = json.Unmarshal(idxData, &idx)
	if len(idx.Settings.Index.Analysis.Analyzer) == 0 {
		return json.RawMessage("[]"), nil
	}
	var analyzers map[string]any
	_ = json.Unmarshal(idx.Settings.Index.Analysis.Analyzer, &analyzers)
	var out []string
	for k := range analyzers {
		out = append(out, k)
	}
	return json.Marshal(out)
}

func IndexFields(index string, mapping json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(mapping, &raw); err != nil {
		return json.RawMessage("[]"), nil
	}
	idxData, ok := raw[index]
	if !ok {
		return json.RawMessage("[]"), nil
	}
	var idx struct {
		Mappings json.RawMessage `json:"mappings"`
	}
	_ = json.Unmarshal(idxData, &idx)
	fields := extractFieldNames(idx.Mappings)
	var out []any
	for _, f := range fields {
		out = append(out, f)
	}
	return json.Marshal(out)
}

func extractFieldNames(mappings json.RawMessage) []string {
	var props any
	_ = json.Unmarshal(mappings, &props)
	return walkProperties(props, "")
}

var analyzableTypes = map[string]bool{"string": true, "text": true}

func walkProperties(v any, prefix string) []string {
	var result []string
	switch m := v.(type) {
	case map[string]any:
		if props, ok := m["properties"]; ok {
			return walkProperties(props, prefix)
		}
		for k, sv := range m {
			if k == "properties" {
				result = append(result, walkProperties(sv, prefix)...)
				continue
			}
			sub, _ := sv.(map[string]any)
			if sub == nil {
				continue
			}
			// check for nested properties
			if _, hasProps := sub["properties"]; hasProps {
				result = append(result, walkProperties(sub, prefix+k+".")...)
				continue
			}
			// check for fields
			if fields, ok := sub["fields"].(map[string]any); ok {
				for fk, fv := range fields {
					fvm, _ := fv.(map[string]any)
					if fvm == nil {
						continue
					}
					ft, _ := fvm["type"].(string)
					if analyzableTypes[ft] {
						result = append(result, prefix+k+"."+fk)
					}
				}
				if ft, _ := sub["type"].(string); analyzableTypes[ft] {
					result = append(result, prefix+k)
				}
				continue
			}
			// check type
			if ft, _ := sub["type"].(string); analyzableTypes[ft] {
				result = append(result, prefix+k)
			}
			// recurse for unknown structure
			result = append(result, walkProperties(sub, prefix+k+".")...)
		}
	}
	return result
}

func Tokens(in json.RawMessage) (json.RawMessage, error) {
	var raw struct {
		Tokens json.RawMessage `json:"tokens"`
	}
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}
	return raw.Tokens, nil
}
