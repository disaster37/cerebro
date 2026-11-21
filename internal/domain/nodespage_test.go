package domain

import (
	"encoding/json"
	"math"
	"testing"
)

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }

func TestResolveNodeCPU(t *testing.T) {
	cases := []struct {
		name        string
		resource    *float64
		os          *int
		process     *int
		wantCPU     *float64
		wantSource  string
		wantRes     *float64
		wantOS      *int
		wantProcess *int
	}{
		{"resource float wins over os/process", floatPtr(12.34), intPtr(30), intPtr(7), floatPtr(12.34), CPUSourceResourceTracker, floatPtr(12.34), intPtr(30), intPtr(7)},
		{"resource rounds to two decimals", floatPtr(12.345), nil, nil, floatPtr(12.35), CPUSourceResourceTracker, floatPtr(12.35), nil, nil},
		{"os fallback", nil, intPtr(30), intPtr(7), floatPtr(30), CPUSourceOS, nil, intPtr(30), intPtr(7)},
		{"process fallback", nil, nil, intPtr(7), floatPtr(7), CPUSourceProcess, nil, nil, intPtr(7)},
		{"all negative sentinels", floatPtr(-1), intPtr(-1), intPtr(-1), nil, "", nil, nil, nil},
		{"nil sentinels", nil, nil, nil, nil, "", nil, nil, nil},
		{"zero resource is valid idle", floatPtr(0), intPtr(42), intPtr(7), floatPtr(0), CPUSourceResourceTracker, floatPtr(0), intPtr(42), intPtr(7)},
		{"zero os is valid idle, no process fallback", nil, intPtr(0), intPtr(15), floatPtr(0), CPUSourceOS, nil, intPtr(0), intPtr(15)},
		{"negative os falls back to process", nil, intPtr(-1), intPtr(15), floatPtr(15), CPUSourceProcess, nil, nil, intPtr(15)},
		{"NaN resource falls back to os", floatPtr(math.NaN()), intPtr(21), nil, floatPtr(21), CPUSourceOS, nil, intPtr(21), nil},
		{"Inf resource falls back to process", floatPtr(math.Inf(1)), nil, intPtr(9), floatPtr(9), CPUSourceProcess, nil, nil, intPtr(9)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveNodeCPU(tc.resource, tc.os, tc.process)
			assertFloatPtr(t, "Effective", got.Effective, tc.wantCPU)
			if got.Source != tc.wantSource {
				t.Errorf("Source = %q, want %q", got.Source, tc.wantSource)
			}
			assertFloatPtr(t, "Resource", got.Resource, tc.wantRes)
			assertIntPtr(t, "OS", got.OS, tc.wantOS)
			assertIntPtr(t, "Process", got.Process, tc.wantProcess)
		})
	}
}

func assertFloatPtr(t *testing.T, name string, got, want *float64) {
	t.Helper()
	if got == nil || want == nil {
		if got != nil || want != nil {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
		return
	}
	if *got != *want {
		t.Errorf("%s = %v, want %v", name, *got, *want)
	}
}

func assertIntPtr(t *testing.T, name string, got, want *int) {
	t.Helper()
	if got == nil || want == nil {
		if got != nil || want != nil {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
		return
	}
	if *got != *want {
		t.Errorf("%s = %v, want %v", name, *got, *want)
	}
}

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

	body, err := NodesData(info, stats, master, nil)
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
	assertFloatPtr(t, "CPUPercent", n.CPUPercent, floatPtr(42))
	if n.CPUSource != CPUSourceOS {
		t.Errorf("CPUSource = %q, want %q", n.CPUSource, CPUSourceOS)
	}
	assertIntPtr(t, "OSCPUPercent", n.OSCPUPercent, intPtr(42))
	assertIntPtr(t, "ProcessCPUPercent", n.ProcessCPUPercent, intPtr(7))
	if n.ResourceCPUPercent != nil {
		t.Errorf("ResourceCPUPercent = %v, want nil", *n.ResourceCPUPercent)
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

func TestNodesDataResourceUsageStatsPriorityAndPartialCoverage(t *testing.T) {
	info := json.RawMessage(`{"nodes":{
		"n1":{"name":"node-1","version":"3.4.0"},
		"n2":{"name":"node-2","version":"3.4.0"},
		"n3":{"name":"node-3","version":"3.4.0"}
	}}`)
	stats := json.RawMessage(`{"nodes":{
		"n1":{"name":"node-1","roles":["data"],"os":{"cpu":{"percent":-1,"load_average":{"1m":-1}}},"process":{"cpu":{"percent":-1}},"jvm":{"mem":{}},"fs":{"total":{"total_in_bytes":0}}},
		"n2":{"name":"node-2","roles":["data"],"os":{"cpu":{"percent":30}},"process":{"cpu":{"percent":12}},"jvm":{"mem":{}},"fs":{"total":{"total_in_bytes":0}}},
		"n3":{"name":"node-3","roles":["data"],"os":{"cpu":{"percent":-1}},"process":{"cpu":{"percent":-1}},"jvm":{"mem":{}},"fs":{"total":{"total_in_bytes":0}}}
	}}`)
	resource := json.RawMessage(`{"nodes":{
		"n1":{"resource_usage_stats":{"timestamp":1,"cpu_utilization_percent":14.2}},
		"n2":{"resource_usage_stats":{}},
		"n3":{"resource_usage_stats":{"cpu_utilization_percent":-1}}
	}}`)

	body, err := NodesData(info, stats, json.RawMessage(`[]`), resource)
	if err != nil {
		t.Fatalf("NodesData: %v", err)
	}
	var out []NodePageEntry
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("nodes = %d, want 3", len(out))
	}

	n1 := out[0]
	assertFloatPtr(t, "node1 CPUPercent", n1.CPUPercent, floatPtr(14.2))
	if n1.CPUSource != CPUSourceResourceTracker {
		t.Errorf("node1 CPUSource = %q, want %q", n1.CPUSource, CPUSourceResourceTracker)
	}
	assertFloatPtr(t, "node1 ResourceCPUPercent", n1.ResourceCPUPercent, floatPtr(14.2))
	if n1.OSCPUPercent != nil || n1.ProcessCPUPercent != nil {
		t.Errorf("node1 os/process sentinels not sanitized: %v/%v", n1.OSCPUPercent, n1.ProcessCPUPercent)
	}
	if n1.LoadAverage != nil {
		t.Errorf("node1 negative load not sanitized: %v", *n1.LoadAverage)
	}

	n2 := out[1]
	assertFloatPtr(t, "node2 CPUPercent", n2.CPUPercent, floatPtr(30))
	if n2.CPUSource != CPUSourceOS {
		t.Errorf("node2 CPUSource = %q, want %q", n2.CPUSource, CPUSourceOS)
	}
	if n2.ResourceCPUPercent != nil {
		t.Errorf("node2 ResourceCPUPercent = %v, want nil (missing)", n2.ResourceCPUPercent)
	}
	assertIntPtr(t, "node2 ProcessCPUPercent", n2.ProcessCPUPercent, intPtr(12))

	n3 := out[2]
	if n3.CPUPercent != nil || n3.CPUSource != "" {
		t.Errorf("node3 CPU = %v/%q, want nil/empty", n3.CPUPercent, n3.CPUSource)
	}
	if n3.OSCPUPercent != nil || n3.ProcessCPUPercent != nil || n3.ResourceCPUPercent != nil {
		t.Errorf("node3 sources = %v/%v/%v, want all nil", n3.ResourceCPUPercent, n3.OSCPUPercent, n3.ProcessCPUPercent)
	}
}

func TestNodesDataWindowsNilLoadAndCPUProcessFallback(t *testing.T) {
	info := json.RawMessage(`{"nodes":{"n1":{"name":"node-1","version":"9.4.3"}}}`)
	stats := json.RawMessage(`{"nodes":{"n1":{"name":"node-1","roles":["data"],"os":{"cpu":{"percent":0}},"process":{"cpu":{"percent":15}},"jvm":{"mem":{}},"fs":{"total":{"total_in_bytes":0}}}}}`)
	body, err := NodesData(info, stats, json.RawMessage(`[]`), nil)
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
	// 0% is a valid idle OS reading and must not be replaced by the process
	// fallback.
	assertFloatPtr(t, "CPUPercent", out[0].CPUPercent, floatPtr(0))
	if out[0].CPUSource != CPUSourceOS {
		t.Errorf("CPUSource = %q, want %q", out[0].CPUSource, CPUSourceOS)
	}
	assertIntPtr(t, "ProcessCPUPercent", out[0].ProcessCPUPercent, intPtr(15))
}
