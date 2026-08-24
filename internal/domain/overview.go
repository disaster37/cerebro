package domain

import "encoding/json"

func ClusterOverview(state, nodesStats, idxStats, clusterSettings, aliases, health, nodesInfo json.RawMessage) (json.RawMessage, error) {
	var cs struct {
		MasterNode   string          `json:"master_node"`
		RoutingTable json.RawMessage `json:"routing_table"`
		Blocks       json.RawMessage `json:"blocks"`
	}
	_ = json.Unmarshal(state, &cs)

	var idxS struct {
		All struct {
			Primaries struct {
				Docs struct {
					Count int64 `json:"count"`
				} `json:"docs"`
			} `json:"primaries"`
			Total struct {
				Store struct {
					SizeInBytes int64 `json:"size_in_bytes"`
				} `json:"store"`
			} `json:"total"`
		} `json:"_all"`
		Indices json.RawMessage `json:"indices"`
	}
	_ = json.Unmarshal(idxStats, &idxS)

	var set struct {
		Persistent struct {
			Cluster struct {
				Routing struct {
					Allocation struct {
						Enable string `json:"enable"`
					} `json:"allocation"`
				} `json:"routing"`
			} `json:"cluster"`
		} `json:"persistent"`
		Transient struct {
			Cluster struct {
				Routing struct {
					Allocation struct {
						Enable string `json:"enable"`
					} `json:"allocation"`
				} `json:"routing"`
			} `json:"cluster"`
		} `json:"transient"`
	}
	_ = json.Unmarshal(clusterSettings, &set)
	enabled := set.Persistent.Cluster.Routing.Allocation.Enable
	if set.Transient.Cluster.Routing.Allocation.Enable != "" {
		enabled = set.Transient.Cluster.Routing.Allocation.Enable
	}
	shardAllocation := enabled == "all"

	indices := overviewIndices(cs.RoutingTable, cs.Blocks, idxS.Indices, aliases)
	nodes := overviewNodes(cs.MasterNode, nodesInfo, nodesStats)

	out := map[string]any{
		"cluster_name":          extractJSON(health, "cluster_name"),
		"status":                extractJSON(health, "status"),
		"number_of_nodes":       extractJSON(health, "number_of_nodes"),
		"active_primary_shards": extractJSON(health, "active_primary_shards"),
		"active_shards":         extractJSON(health, "active_shards"),
		"relocating_shards":     extractJSON(health, "relocating_shards"),
		"initializing_shards":   extractJSON(health, "initializing_shards"),
		"unassigned_shards":     extractJSON(health, "unassigned_shards"),
		"docs_count":            idxS.All.Primaries.Docs.Count,
		"size_in_bytes":         idxS.All.Total.Store.SizeInBytes,
		"total_indices":         len(indices),
		"closed_indices":        countClosed(indices),
		"special_indices":       countSpecial(indices),
		"indices":               indices,
		"nodes":                 nodes,
		"shard_allocation":      shardAllocation,
	}
	return json.Marshal(out)
}

func extractJSON(raw json.RawMessage, key string) any {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return m[key]
}

func countClosed(indices []any) int {
	n := 0
	for _, idx := range indices {
		if m, ok := idx.(map[string]any); ok {
			if closed, _ := m["closed"].(bool); closed {
				n++
			}
		}
	}
	return n
}

func countSpecial(indices []any) int {
	n := 0
	for _, idx := range indices {
		if m, ok := idx.(map[string]any); ok {
			if special, _ := m["special"].(bool); special {
				n++
			}
		}
	}
	return n
}

func overviewIndices(routingTable, blocks, idxStats, aliases json.RawMessage) []any {
	var rt struct {
		Indices map[string]json.RawMessage `json:"indices"`
	}
	_ = json.Unmarshal(routingTable, &rt)

	var blk struct {
		Indices map[string]json.RawMessage `json:"indices"`
	}
	_ = json.Unmarshal(blocks, &blk)

	var als struct {
		Aliases json.RawMessage `json:"aliases"`
	}
	_ = json.Unmarshal(aliases, &als)

	var result []any
	for name, shards := range rt.Indices {
		var idxStatsEntry json.RawMessage
		var istatsObj struct {
			Indices map[string]json.RawMessage `json:"indices"`
		}
		if json.Unmarshal(idxStats, &istatsObj) == nil {
			idxStatsEntry = istatsObj.Indices[name]
		}

		out := overviewIndexEntry(name, shards, idxStatsEntry, blk.Indices)
		result = append(result, out)
	}
	for name, block := range blk.Indices {
		if _, ok := rt.Indices[name]; ok {
			continue
		}
		var b struct {
			Four any `json:"4"`
		}
		_ = json.Unmarshal(block, &b)
		if b.Four != nil {
			result = append(result, map[string]any{
				"name":    name,
				"closed":  true,
				"special": len(name) > 0 && name[0] == '.',
			})
		}
	}
	return result
}

func overviewIndexEntry(name string, shards, idxStats json.RawMessage, blocks map[string]json.RawMessage) map[string]any {
	closed := false
	if block, ok := blocks[name]; ok {
		var b struct {
			Four any `json:"4"`
		}
		_ = json.Unmarshal(block, &b)
		closed = b.Four != nil
	}
	return map[string]any{
		"name":    name,
		"closed":  closed,
		"special": len(name) > 0 && name[0] == '.',
	}
}

func overviewNodes(masterID string, nodesInfo, nodesStats json.RawMessage) []any {
	var ni struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	var ns struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	_ = json.Unmarshal(nodesInfo, &ni)
	_ = json.Unmarshal(nodesStats, &ns)

	var result []any
	for id, nsData := range ns.Nodes {
		niData := ni.Nodes[id]
		result = append(result, buildOverviewNode(id, masterID, niData, nsData))
	}
	return result
}

func buildOverviewNode(id, masterID string, info, stats json.RawMessage) map[string]any {
	var is struct {
		Version string `json:"version"`
		OS      struct {
			AvailableProcessors int `json:"available_processors"`
		} `json:"os"`
		JVM struct {
			Version string `json:"version"`
		} `json:"jvm"`
	}
	var ss struct {
		Name  string   `json:"name"`
		Host  string   `json:"host"`
		IP    string   `json:"ip"`
		Roles []string `json:"roles"`
		OS    struct {
			CPU struct {
				Percent     int `json:"percent"`
				LoadAverage struct {
					OneM float64 `json:"1m"`
				} `json:"load_average"`
			} `json:"cpu"`
		} `json:"os"`
		JVM struct {
			Mem struct {
				HeapUsedInBytes      int64   `json:"heap_used_in_bytes"`
				HeapCommittedInBytes int64   `json:"heap_committed_in_bytes"`
				HeapUsedPercent      float64 `json:"heap_used_percent"`
				HeapMaxInBytes       int64   `json:"heap_max_in_bytes"`
			} `json:"mem"`
		} `json:"jvm"`
		FS struct {
			Total struct {
				TotalInBytes int64 `json:"total_in_bytes"`
				FreeInBytes  int64 `json:"free_in_bytes"`
			} `json:"total"`
		} `json:"fs"`
	}
	_ = json.Unmarshal(info, &is)
	_ = json.Unmarshal(stats, &ss)

	roles := resolveRole(map[string]bool{
		"data": true, "data_content": true, "data_hot": true,
		"data_warm": true, "data_cold": true,
	}, ss.Roles)

	diskPct := 0
	if ss.FS.Total.TotalInBytes > 0 {
		diskPct = 100 - int(100*float64(ss.FS.Total.FreeInBytes)/float64(ss.FS.Total.TotalInBytes))
	}

	return map[string]any{
		"id":                   id,
		"current_master":       id == masterID,
		"name":                 ss.Name,
		"host":                 ss.Host,
		"ip":                   ss.IP,
		"es_version":           is.Version,
		"jvm_version":          is.JVM.Version,
		"load_average":         ss.OS.CPU.LoadAverage.OneM,
		"available_processors": is.OS.AvailableProcessors,
		"cpu_percent":          ss.OS.CPU.Percent,
		"master":               roles.master,
		"data":                 roles.data,
		"coordinating":         roles.coordinating,
		"ingest":               roles.ingest,
		"heap": map[string]any{
			"used":         ss.JVM.Mem.HeapUsedInBytes,
			"committed":    ss.JVM.Mem.HeapCommittedInBytes,
			"used_percent": ss.JVM.Mem.HeapUsedPercent,
			"max":          ss.JVM.Mem.HeapMaxInBytes,
		},
		"disk": map[string]any{
			"total":        ss.FS.Total.TotalInBytes,
			"free":         ss.FS.Total.FreeInBytes,
			"used_percent": diskPct,
		},
		"attributes": map[string]any{},
	}
}
