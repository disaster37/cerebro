package models

import "encoding/json"

func IndexMetadata(in json.RawMessage) (json.RawMessage, error) {
	mappings := extractFirst(in, "mappings")
	settings := extractFirst(in, "settings")
	return json.Marshal(map[string]json.RawMessage{
		"mappings": mappings,
		"settings": settings,
	})
}

func extractFirst(in json.RawMessage, key string) json.RawMessage {
	var d any
	_ = json.Unmarshal(in, &d)
	return deepExtract(d, key)
}

func deepExtract(v any, key string) json.RawMessage {
	switch m := v.(type) {
	case map[string]any:
		if sub, ok := m[key]; ok {
			b, _ := json.Marshal(sub)
			return b
		}
		for _, sv := range m {
			if r := deepExtract(sv, key); len(r) > 0 {
				return r
			}
		}
	case []any:
		for _, sv := range m {
			if r := deepExtract(sv, key); len(r) > 0 {
				return r
			}
		}
	}
	return nil
}
