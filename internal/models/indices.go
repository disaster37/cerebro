package models

import "encoding/json"

func CommonsIndices(in json.RawMessage) (json.RawMessage, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(in, &arr); err != nil {
		return nil, err
	}
	var out []string
	for _, m := range arr {
		if idx, ok := m["index"]; ok {
			var s string
			if json.Unmarshal(idx, &s) == nil {
				out = append(out, s)
			}
		}
	}
	return json.Marshal(out)
}
