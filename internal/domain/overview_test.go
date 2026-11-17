package domain

import (
	"encoding/json"
	"testing"
)

func TestClusterOverviewPopulatesDocsSizeShardsAndAllocation(t *testing.T) {
	state := json.RawMessage(`{
		"cluster_manager_node": "node1",
		"routing_table": {"indices": {"idx": {"shards": {
			"0": [
				{"index":"idx","shard":0,"primary":true,"state":"STARTED","node":"node1","relocating_node":null},
				{"index":"idx","shard":0,"primary":false,"state":"STARTED","node":"node2","relocating_node":"node3"}
			]
		}}}},
		"blocks": {"indices": {}}
	}`)
	idxStats := json.RawMessage(`{
		"_all": {"primaries":{"docs":{"count":10}},"total":{"store":{"size_in_bytes":200}}},
		"indices": {"idx": {
			"primaries":{"docs":{"count":10,"deleted":2},"store":{"size_in_bytes":100}},
			"total":{"store":{"size_in_bytes":200}}
		}}
	}`)
	settings := json.RawMessage(`{"persistent":{},"transient":{}}`)
	aliases := json.RawMessage(`{"idx":{"aliases":{"a1":{}}}}`)
	health := json.RawMessage(`{"cluster_name":"c1","status":"green","number_of_nodes":3,"active_shards":2,"unassigned_shards":0}`)
	nodesInfo := json.RawMessage(`{"nodes":{"node1":{"version":"9.4.3","attributes":{"zone":"a","ml.machine_memory":"1","xpack.installed":"true"},"os":{"available_processors":4},"jvm":{"version":"21"}}}}`)
	nodesStats := json.RawMessage(`{"nodes":{"node1":{"name":"n1","host":"h1","ip":"1.1.1.1","roles":["cluster_manager","data"],"os":{"cpu":{"percent":12,"load_average":{"1m":0.5}}},"jvm":{"mem":{"heap_used_in_bytes":10,"heap_max_in_bytes":100,"heap_used_percent":10}},"fs":{"total":{"total_in_bytes":100,"free_in_bytes":40}}}}}`)

	body, err := ClusterOverview(state, nodesStats, idxStats, settings, aliases, health, nodesInfo)
	if err != nil {
		t.Fatalf("ClusterOverview: %v", err)
	}

	var out struct {
		DocsCount       int64            `json:"docs_count"`
		SizeInBytes     int64            `json:"size_in_bytes"`
		ShardAllocation bool             `json:"shard_allocation"`
		Indices         []OverviewIndex  `json:"indices"`
		Nodes           []map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.DocsCount != 10 || out.SizeInBytes != 200 {
		t.Errorf("docs/size = %d/%d, want 10/200", out.DocsCount, out.SizeInBytes)
	}
	if !out.ShardAllocation {
		t.Error("shard_allocation = false, want true when unset (default all)")
	}
	if len(out.Indices) != 1 {
		t.Fatalf("indices = %d, want 1", len(out.Indices))
	}
	idx := out.Indices[0]
	if idx.DocCount != 10 || idx.DeletedDocs != 2 || idx.SizeInBytes != 100 || idx.TotalSizeInBytes != 200 {
		t.Errorf("index stats = %+v, want docs 10 deleted 2 size 100 total 200", idx)
	}
	if idx.NumShards != 1 || idx.NumReplicas != 1 {
		t.Errorf("shards/replicas = %d/%d, want 1/1", idx.NumShards, idx.NumReplicas)
	}
	if len(idx.Shards["node1"]) != 1 || len(idx.Shards["node2"]) != 1 {
		t.Errorf("shard grouping = %+v, want node1 and node2", idx.Shards)
	}
	if len(idx.Shards["node3"]) != 1 || idx.Shards["node3"][0].State != "INITIALIZING" {
		t.Errorf("relocating synthetic shard missing: %+v", idx.Shards["node3"])
	}
	if len(idx.Aliases) != 1 || idx.Aliases[0] != "a1" {
		t.Errorf("aliases = %v, want [a1]", idx.Aliases)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1", len(out.Nodes))
	}
	node := out.Nodes[0]
	if node["master"] != true || node["current_master"] != true {
		t.Errorf("cluster_manager role not resolved: %+v", node)
	}
	attrs := node["attributes"].(map[string]any)
	if _, ok := attrs["zone"]; !ok {
		t.Errorf("custom attribute zone missing: %+v", attrs)
	}
	if _, ok := attrs["ml.machine_memory"]; ok {
		t.Errorf("internal attribute not stripped: %+v", attrs)
	}
}

func TestClusterOverviewClosedIndexAndEmptySlices(t *testing.T) {
	state := json.RawMessage(`{"master_node":"n1","routing_table":{"indices":{}},"blocks":{"indices":{"closed-idx":{"4":{"read_only":true}}}}}`)
	idxStats := json.RawMessage(`{"_all":{"primaries":{"docs":{"count":0}},"total":{"store":{"size_in_bytes":0}}},"indices":{}}`)
	body, err := ClusterOverview(state, json.RawMessage(`{}`), idxStats, json.RawMessage(`{}`), json.RawMessage(`{}`), json.RawMessage(`{}`), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("ClusterOverview: %v", err)
	}
	var out struct {
		Indices []OverviewIndex  `json:"indices"`
		Nodes   []map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Indices == nil || out.Nodes == nil {
		t.Fatal("indices/nodes must be non-nil empty slices")
	}
	if len(out.Indices) != 1 || !out.Indices[0].Closed {
		t.Errorf("closed index not reported: %+v", out.Indices)
	}
}

func TestClusterOverviewTransientAllocationWins(t *testing.T) {
	state := json.RawMessage(`{"master_node":"n1","routing_table":{"indices":{}},"blocks":{}}`)
	settings := json.RawMessage(`{"persistent":{"cluster":{"routing":{"allocation":{"enable":"all"}}}},"transient":{"cluster":{"routing":{"allocation":{"enable":"none"}}}}}`)
	body, err := ClusterOverview(state, json.RawMessage(`{}`), json.RawMessage(`{}`), settings, json.RawMessage(`{}`), json.RawMessage(`{}`), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("ClusterOverview: %v", err)
	}
	var out struct {
		ShardAllocation bool `json:"shard_allocation"`
	}
	_ = json.Unmarshal(body, &out)
	if out.ShardAllocation {
		t.Error("shard_allocation = true, want false when transient disables allocation")
	}
}
