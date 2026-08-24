package domain

import "encoding/json"

func Aliases(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}

	type entry struct {
		Alias         string          `json:"alias"`
		Index         string          `json:"index"`
		Filter        json.RawMessage `json:"filter"`
		SearchRouting json.RawMessage `json:"search_routing"`
		IndexRouting  json.RawMessage `json:"index_routing"`
	}

	var out []entry
	for index, v := range raw {
		var info struct {
			Aliases map[string]json.RawMessage `json:"aliases"`
		}
		_ = json.Unmarshal(v, &info)
		if len(info.Aliases) == 0 {
			continue
		}
		for alias, props := range info.Aliases {
			var propsObj struct {
				Filter        json.RawMessage `json:"filter"`
				SearchRouting json.RawMessage `json:"search_routing"`
				IndexRouting  json.RawMessage `json:"index_routing"`
			}
			_ = json.Unmarshal(props, &propsObj)
			e := entry{
				Alias:         alias,
				Index:         index,
				Filter:        marshalNull(propsObj.Filter),
				SearchRouting: marshalNull(propsObj.SearchRouting),
				IndexRouting:  marshalNull(propsObj.IndexRouting),
			}
			if len(e.Filter) == 0 || string(e.Filter) == "null" {
				e.Filter = json.RawMessage("null")
			}
			if len(e.SearchRouting) == 0 || string(e.SearchRouting) == "null" {
				e.SearchRouting = json.RawMessage("null")
			}
			if len(e.IndexRouting) == 0 || string(e.IndexRouting) == "null" {
				e.IndexRouting = json.RawMessage("null")
			}
			out = append(out, e)
		}
	}
	return json.Marshal(out)
}

func marshalNull(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage("null")
	}
	return v
}
