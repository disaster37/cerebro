package domain

import "encoding/json"

func CommonsNodes(in json.RawMessage) (json.RawMessage, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(in, &arr); err != nil {
		return nil, err
	}
	var out []string
	for _, m := range arr {
		if name, ok := m["name"]; ok {
			var s string
			if json.Unmarshal(name, &s) == nil {
				out = append(out, s)
			}
		}
	}
	return json.Marshal(out)
}
