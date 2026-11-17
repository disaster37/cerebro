package domain

import (
	"encoding/json"
	"testing"
)

func TestNodesDataCPUAndLoad(t *testing.T) {
	info := json.RawMessage(`{"nodes":{"n1":{"name":"node-1","version":"9.4.3","attributes":{"zone":"a"},"jvm":{"version":"21"},"os":{"available_processors":8}}}}`)
	stats := json.RawMessage(`{"nodes":{"n1":{
		"name":"node-1","host":"h1","ip":"1.1.1.1","roles":["cluster_manager","data_frozen","search"],
		"os":{"cpu":{"percent":42,"load_average":{"1m":0.5,"5m":0.4,"15m":0.3}}},
		"process":{"cpu":{"percent":7}},
		"jvm":{"uptime_in_millis":123456,"mem":{"heap_used_in_bytes":10,"heap_max_in_bytes":100,"heap_used_percent":10}},
		"fs":{"total":{"total_in_bytes":100,"free_in_bytes":40,"available_in_bytes":30}}
	}}}`)
	master := json.RawMessage(`[{"id":"n1"}]`)

	body, err := NodesData(info, stats, master)
	if err != nil {
		t.Fatalf("NodesData: %v", err)
	}
	var out []NodePageEntry
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("nodes = %d, want 1", len(out))
	}
	n := out[0]
	if n.CPUPercent != 42 || n.ProcessCPUPercent != 7 {
		t.Errorf("cpu = %d process = %d, want 42/7", n.CPUPercent, n.ProcessCPUPercent)
	}
	if n.LoadAverage == nil || *n.LoadAverage != 0.5 {
		t.Errorf("load 1m = %v, want 0.5", n.LoadAverage)
	}
	if n.LoadAverage5m == nil || *n.LoadAverage5m != 0.4 || n.LoadAverage15m == nil || *n.LoadAverage15m != 0.3 {
		t.Errorf("load 5m/15m = %v/%v, want 0.4/0.3", n.LoadAverage5m, n.LoadAverage15m)
	}
	if !n.Master || !n.Data || n.Coordinating {
		t.Errorf("roles = %+v, want master+data (cluster_manager/data_frozen/search)", n)
	}
	if n.UptimeInMillis != 123456 {
		t.Errorf("uptime = %d, want 123456", n.UptimeInMillis)
	}
	if n.Disk.UsedPercent != 70 {
		t.Errorf("disk used = %d, want 70 (available_in_bytes preferred)", n.Disk.UsedPercent)
	}
	if n.Attributes["zone"] != "a" {
		t.Errorf("attributes = %+v, want zone=a", n.Attributes)
	}
}

func TestNodesDataWindowsNilLoadAndCPUProcessFallback(t *testing.T) {
	info := json.RawMessage(`{"nodes":{"n1":{"name":"node-1","version":"9.4.3"}}}`)
	stats := json.RawMessage(`{"nodes":{"n1":{"name":"node-1","roles":["data"],"os":{"cpu":{"percent":0}},"process":{"cpu":{"percent":15}},"jvm":{"mem":{}},"fs":{"total":{"total_in_bytes":0}}}}}`)
	body, err := NodesData(info, stats, json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("NodesData: %v", err)
	}
	var out []NodePageEntry
	_ = json.Unmarshal(body, &out)
	if len(out) != 1 {
		t.Fatalf("nodes = %d, want 1", len(out))
	}
	if out[0].LoadAverage != nil {
		t.Errorf("load = %v, want nil on Windows/unsupported", out[0].LoadAverage)
	}
	if out[0].CPUPercent != 15 {
		t.Errorf("cpu = %d, want process fallback 15", out[0].CPUPercent)
	}
}
