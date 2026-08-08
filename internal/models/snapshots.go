package models

import "encoding/json"

func Snapshots(in json.RawMessage) (json.RawMessage, error) {
	var raw struct {
		Snapshots json.RawMessage `json:"snapshots"`
	}
	if err := json.Unmarshal(in, &raw); err != nil {
		return json.RawMessage("[]"), nil
	}
	if len(raw.Snapshots) == 0 {
		return json.RawMessage("[]"), nil
	}
	return raw.Snapshots, nil
}
