package models

import "encoding/json"

func Templates(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}

	type entry struct {
		Name     string          `json:"name"`
		Template json.RawMessage `json:"template"`
	}
	var out []entry
	for name, tmpl := range raw {
		out = append(out, entry{Name: name, Template: tmpl})
	}
	return json.Marshal(out)
}
