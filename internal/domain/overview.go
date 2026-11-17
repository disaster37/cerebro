package domain

import (
	"encoding/json"
	"sort"
)

// UnassignedShardInfo explains why a shard could not be allocated.
type UnassignedShardInfo struct {
	Reason           string `json:"reason"`
	At               string `json:"at,omitempty"`
	Details          string `json:"details,omitempty"`
	AllocationStatus string `json:"last_allocation_status,omitempty"`
}

// OverviewShard is one shard routing entry in the overview shard-allocation grid.
type OverviewShard struct {
	Index          string               `json:"index"`
	Shard          int                  `json:"shard"`
	Primary        bool                 `json:"primary"`
	State          string               `json:"state"`
	Node           *string              `json:"node"`
	RelocatingNode *string              `json:"relocating_node,omitempty"`
	UnassignedInfo *UnassignedShardInfo `json:"unassigned_info,omitempty"`
}

// OverviewIndex is one index row in the overview.
type OverviewIndex struct {
	Name                   string                     `json:"name"`
	Closed                 bool                       `json:"closed"`
	Special                bool                       `json:"special"`
	BackingIndex           bool                       `json:"backing_index"`
	DataStream             string                     `json:"data_stream,omitempty"`
	IsWriteIndex           bool                       `json:"is_write_index,omitempty"`
	Status                 string                     `json:"status"` // "green" | "yellow" | "red" | "closed"
	Unhealthy              bool                       `json:"unhealthy"`
	Unreplicated           bool                       `json:"unreplicated"`
	UnassignedShardCount   int                        `json:"unassigned_shard_count"`
	InitializingShardCount int                        `json:"initializing_shard_count"`
	RelocatingShardCount   int                        `json:"relocating_shard_count"`
	DocCount               int64                      `json:"doc_count"`
	DeletedDocs            int64                      `json:"deleted_docs"`
	SizeInBytes            int64                      `json:"size_in_bytes"`
	TotalSizeInBytes       int64                      `json:"total_size_in_bytes"`
	Aliases                []string                   `json:"aliases"`
	NumShards              int                        `json:"num_shards"`
	NumReplicas            int                        `json:"num_replicas"`
	Shards                 map[string][]OverviewShard `json:"shards"`
}

type aliasMap map[string]struct {
	Aliases map[string]json.RawMessage `json:"aliases"`
}

type indexStatPayload struct {
	Primaries struct {
		Docs struct {
			Count   int64 `json:"count"`
			Deleted int64 `json:"deleted"`
		} `json:"docs"`
		Store struct {
			SizeInBytes int64 `json:"size_in_bytes"`
		} `json:"store"`
	} `json:"primaries"`
	Total struct {
		Store struct {
			SizeInBytes int64 `json:"size_in_bytes"`
		} `json:"store"`
	} `json:"total"`
}

func ClusterOverview(state, nodesStats, idxStats, clusterSettings, aliases, health, nodesInfo, dataStreamsRaw json.RawMessage) (json.RawMessage, error) {
	var cs struct {
		MasterNode         string          `json:"master_node"`
		ClusterManagerNode string          `json:"cluster_manager_node"`
		RoutingTable       json.RawMessage `json:"routing_table"`
		Blocks             json.RawMessage `json:"blocks"`
	}
	_ = json.Unmarshal(state, &cs)
	masterID := firstNonEmpty(cs.ClusterManagerNode, cs.MasterNode)

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
	if enabled == "" {
		enabled = "all"
	}
	shardAllocation := enabled == "all"

	dataStreams, backingMeta, dsErr := ParseDataStreams(dataStreamsRaw)
	if dsErr != nil {
		dataStreams = []DataStreamSummary{}
		backingMeta = nil
	}
	indices := overviewIndices(cs.RoutingTable, cs.Blocks, idxS.Indices, aliases, backingMeta)
	nodes := overviewNodes(masterID, nodesInfo, nodesStats)
	aggregateDataStreamStats(dataStreams, indices)

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
		"data_streams":          dataStreams,
		"nodes":                 nodes,
		"shard_allocation":      shardAllocation,
	}
	return json.Marshal(out)
}

func aggregateDataStreamStats(dataStreams []DataStreamSummary, indices []OverviewIndex) {
	byName := make(map[string]*DataStreamSummary, len(dataStreams))
	for i := range dataStreams {
		byName[dataStreams[i].Name] = &dataStreams[i]
	}
	for _, idx := range indices {
		if idx.DataStream == "" {
			continue
		}
		ds, ok := byName[idx.DataStream]
		if !ok {
			continue
		}
		ds.DocCount += idx.DocCount
		ds.SizeInBytes += idx.SizeInBytes
		ds.TotalSizeInBytes += idx.TotalSizeInBytes
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func extractJSON(raw json.RawMessage, key string) any {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return m[key]
}

func countClosed(indices []OverviewIndex) int {
	n := 0
	for _, idx := range indices {
		if idx.Closed {
			n++
		}
	}
	return n
}

func countSpecial(indices []OverviewIndex) int {
	n := 0
	for _, idx := range indices {
		if idx.Special {
			n++
		}
	}
	return n
}

func overviewIndices(routingTable, blocks, idxStats, aliases json.RawMessage, backingMeta map[string]DataStreamBackingMeta) []OverviewIndex {
	var rt struct {
		Indices map[string]json.RawMessage `json:"indices"`
	}
	_ = json.Unmarshal(routingTable, &rt)

	var blk struct {
		Indices map[string]json.RawMessage `json:"indices"`
	}
	_ = json.Unmarshal(blocks, &blk)

	stats := map[string]indexStatPayload{}
	_ = json.Unmarshal(idxStats, &stats)

	aliasMap := aliasMap{}
	_ = json.Unmarshal(aliases, &aliasMap)

	result := make([]OverviewIndex, 0, len(rt.Indices))
	for name, shards := range rt.Indices {
		result = append(result, overviewIndexEntry(name, shards, stats[name], blk.Indices, aliasMap, backingMeta))
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
			entry := OverviewIndex{
				Name:    name,
				Closed:  true,
				Special: isSpecialIndex(name),
				Status:  "closed",
				Aliases: aliasNames(aliasMap, name),
				Shards:  map[string][]OverviewShard{},
			}
			applyDataStreamMeta(&entry, backingMeta)
			result = append(result, entry)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func overviewIndexEntry(name string, shardsRaw json.RawMessage, stats indexStatPayload, blocks map[string]json.RawMessage, aliases aliasMap, backingMeta map[string]DataStreamBackingMeta) OverviewIndex {
	closed := false
	if block, ok := blocks[name]; ok {
		var b struct {
			Four any `json:"4"`
		}
		_ = json.Unmarshal(block, &b)
		closed = b.Four != nil
	}

	var routing struct {
		Shards map[string][]OverviewShard `json:"shards"`
	}
	_ = json.Unmarshal(shardsRaw, &routing)

	grouped := map[string][]OverviewShard{}
	unhealthy := false
	unassigned := 0
	initializing := 0
	relocating := 0
	primaryDown := false
	maxShard := -1
	for _, shardList := range routing.Shards {
		for _, shard := range shardList {
			if shard.Shard > maxShard {
				maxShard = shard.Shard
			}
			switch shard.State {
			case "STARTED":
			case "UNASSIGNED":
				unassigned++
			case "INITIALIZING":
				initializing++
			case "RELOCATING":
				relocating++
			}
			if shard.State != "STARTED" {
				unhealthy = true
				if shard.Primary {
					primaryDown = true
				}
			}
			nodeKey := "unassigned"
			if shard.Node != nil && *shard.Node != "" {
				nodeKey = *shard.Node
			}
			grouped[nodeKey] = append(grouped[nodeKey], shard)
			if shard.RelocatingNode != nil && *shard.RelocatingNode != "" {
				relocatingShard := OverviewShard{
					Index:   name,
					Shard:   shard.Shard,
					Primary: false,
					State:   "INITIALIZING",
					Node:    shard.RelocatingNode,
				}
				grouped[*shard.RelocatingNode] = append(grouped[*shard.RelocatingNode], relocatingShard)
			}
		}
	}
	for nodeKey := range grouped {
		sort.SliceStable(grouped[nodeKey], func(i, j int) bool {
			if grouped[nodeKey][i].Shard != grouped[nodeKey][j].Shard {
				return grouped[nodeKey][i].Shard < grouped[nodeKey][j].Shard
			}
			return grouped[nodeKey][i].Primary && !grouped[nodeKey][j].Primary
		})
	}

	numShards := 0
	if maxShard >= 0 {
		numShards = maxShard + 1
	}
	numReplicas := 0
	if primaryShards, ok := routing.Shards["0"]; ok && len(primaryShards)-1 > 0 {
		numReplicas = len(primaryShards) - 1
	}

	status := "green"
	switch {
	case closed:
		status = "closed"
	case primaryDown || numShards == 0:
		status = "red"
	case unhealthy:
		status = "yellow"
	}

	entry := OverviewIndex{
		Name:                   name,
		Closed:                 closed,
		Special:                isSpecialIndex(name),
		Status:                 status,
		Unhealthy:              !closed && unhealthy,
		Unreplicated:           !closed && numShards > 0 && numReplicas == 0,
		UnassignedShardCount:   unassigned,
		InitializingShardCount: initializing,
		RelocatingShardCount:   relocating,
		DocCount:               stats.Primaries.Docs.Count,
		DeletedDocs:            stats.Primaries.Docs.Deleted,
		SizeInBytes:            stats.Primaries.Store.SizeInBytes,
		TotalSizeInBytes:       stats.Total.Store.SizeInBytes,
		Aliases:                aliasNames(aliases, name),
		NumShards:              numShards,
		NumReplicas:            numReplicas,
		Shards:                 grouped,
	}
	applyDataStreamMeta(&entry, backingMeta)
	return entry
}

// applyDataStreamMeta marks an index as a data stream backing index and
// decouples user data stream backing indices from the `.special` system-index
// filter. When GET /_data_stream is unavailable it falls back to the `.ds-*`
// naming heuristic.
func applyDataStreamMeta(entry *OverviewIndex, backingMeta map[string]DataStreamBackingMeta) {
	if meta, ok := backingMeta[entry.Name]; ok {
		entry.BackingIndex = true
		entry.DataStream = meta.DataStream
		entry.IsWriteIndex = meta.IsWriteIndex
		if !meta.HiddenOrSystem {
			entry.Special = false
		}
		return
	}
	if name, ok := DataStreamNameFromBackingIndex(entry.Name); ok {
		entry.BackingIndex = true
		entry.DataStream = name
		entry.Special = false
	}
}

func isSpecialIndex(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

func aliasNames(aliases aliasMap, index string) []string {
	entry, ok := aliases[index]
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(entry.Aliases))
	for alias := range entry.Aliases {
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func overviewNodes(masterID string, nodesInfo, nodesStats json.RawMessage) []map[string]any {
	var ni struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	var ns struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	_ = json.Unmarshal(nodesInfo, &ni)
	_ = json.Unmarshal(nodesStats, &ns)

	result := make([]map[string]any, 0, len(ns.Nodes))
	for id, nsData := range ns.Nodes {
		niData := ni.Nodes[id]
		result = append(result, buildOverviewNode(id, masterID, niData, nsData))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i]["name"].(string) < result[j]["name"].(string)
	})
	return result
}

var internalNodeAttributes = map[string]bool{
	"ml.machine_memory": true,
	"xpack.installed":   true,
	"transform.node":    true,
	"ml.max_open_jobs":  true,
	"ml.config_version": true,
}

func buildOverviewNode(id, masterID string, info, stats json.RawMessage) map[string]any {
	var is struct {
		Version    string            `json:"version"`
		Attributes map[string]string `json:"attributes"`
		OS         struct {
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
					OneM *float64 `json:"1m"`
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
				TotalInBytes     int64 `json:"total_in_bytes"`
				FreeInBytes      int64 `json:"free_in_bytes"`
				AvailableInBytes int64 `json:"available_in_bytes"`
			} `json:"total"`
		} `json:"fs"`
	}
	_ = json.Unmarshal(info, &is)
	_ = json.Unmarshal(stats, &ss)

	roles := resolveRole(map[string]bool{
		"data": true, "data_content": true, "data_hot": true,
		"data_warm": true, "data_cold": true, "data_frozen": true, "search": true,
	}, ss.Roles)

	diskPct := 0
	if ss.FS.Total.TotalInBytes > 0 {
		available := ss.FS.Total.AvailableInBytes
		if available == 0 {
			available = ss.FS.Total.FreeInBytes
		}
		diskPct = 100 - int(100*float64(available)/float64(ss.FS.Total.TotalInBytes))
	}

	attributes := map[string]string{}
	for k, v := range is.Attributes {
		if internalNodeAttributes[k] {
			continue
		}
		attributes[k] = v
	}

	var load any
	if ss.OS.CPU.LoadAverage.OneM != nil {
		load = *ss.OS.CPU.LoadAverage.OneM
	}

	return map[string]any{
		"id":                   id,
		"current_master":       id == masterID,
		"name":                 ss.Name,
		"host":                 ss.Host,
		"ip":                   ss.IP,
		"es_version":           is.Version,
		"jvm_version":          is.JVM.Version,
		"load_average":         load,
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
		"attributes": attributes,
	}
}
