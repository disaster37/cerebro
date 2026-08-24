package domain

import "encoding/json"

func AutocompletionIndices(aliases json.RawMessage) (json.RawMessage, error) {
	var raw map[string]struct {
		Aliases map[string]json.RawMessage `json:"aliases"`
	}
	if err := json.Unmarshal(aliases, &raw); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
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
