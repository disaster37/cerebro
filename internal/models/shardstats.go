package models

import (
	"encoding/json"
	"fmt"
)

func ShardStats(index, node string, shard int, stats, recovery json.RawMessage) (json.RawMessage, error) {
	var ss struct {
		Indices map[string]struct {
			Shards map[string]json.RawMessage `json:"shards"`
		} `json:"indices"`
	}
	_ = json.Unmarshal(stats, &ss)

	idxData, ok := ss.Indices[index]
	if ok {
		shardKey := fmt.Sprintf("%d", shard)
		if shards, has := idxData.Shards[shardKey]; has {
			var shardList []map[string]any
			_ = json.Unmarshal(shards, &shardList)
			for _, s := range shardList {
				if routing, ok := s["routing"].(map[string]any); ok {
					if n, _ := routing["node"].(string); n == node {
						b, _ := json.Marshal(s)
						return b, nil
					}
				}
			}
		}
	}

	var rec struct {
		Recovery map[string]struct {
			Shards json.RawMessage `json:"shards"`
		} `json:"recovery"`
	}
	_ = json.Unmarshal(recovery, &rec)
	if idxRec, ok := rec.Recovery[index]; ok {
		var recList []map[string]any
		_ = json.Unmarshal(idxRec.Shards, &recList)
		for _, r := range recList {
			if target, ok := r["target"].(map[string]any); ok {
				if id, _ := target["id"].(string); id == node {
					if sid, ok := asFloat(r["id"]); ok && int(sid) == shard {
						b, _ := json.Marshal(r)
						return b, nil
					}
				}
			}
		}
	}

	return json.RawMessage("null"), nil
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}
