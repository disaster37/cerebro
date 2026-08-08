package models

import (
	"encoding/json"
)

func Repositories(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}

	type entry struct {
		Name     string          `json:"name"`
		Type     json.RawMessage `json:"type"`
		Settings json.RawMessage `json:"settings"`
	}
	var out []entry
	for name, v := range raw {
		var info struct {
			Type     json.RawMessage `json:"type"`
			Settings json.RawMessage `json:"settings"`
		}
		if err := json.Unmarshal(v, &info); err != nil {
			return nil, err
		}
		settings := json.RawMessage("{}")
		if len(info.Settings) > 0 {
			settings = info.Settings
		}
		out = append(out, entry{Name: name, Type: info.Type, Settings: settings})
	}
	return json.Marshal(out)
}

func SnapshotRepoList(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return json.Marshal([]any{})
	}
	var out []string
	for name := range raw {
		out = append(out, name)
	}
	return json.Marshal(out)
}

func SnapshotIndices(in json.RawMessage) (json.RawMessage, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(in, &arr); err != nil {
		return nil, err
	}
	type entry struct {
		Name    string `json:"name"`
		Special bool   `json:"special"`
	}
	var out []entry
	for _, m := range arr {
		var name string
		if nm, ok := m["index"]; ok {
			_ = json.Unmarshal(nm, &name)
		}
		out = append(out, entry{Name: name, Special: len(name) > 0 && name[0] == '.'})
	}
	return json.Marshal(out)
}
