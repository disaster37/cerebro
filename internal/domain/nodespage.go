package domain

import (
	"encoding/json"
	"math"
	"sort"
)

// CPU source identifiers exposed to the UI so it can explain where the
// effective CPU percentage came from.
const (
	CPUSourceResourceTracker = "resource_tracker"
	CPUSourceOS              = "os"
	CPUSourceProcess         = "process"
)

// ResolvedNodeCPU is the per-node CPU resolution result. Effective is the value
// the UI displays, Source says where it came from, and the remaining fields are
// the raw sanitized per-source readings.
type ResolvedNodeCPU struct {
	Effective *float64 // Effective CPU % (rounded to 2 decimals) or nil
	Source    string   // "resource_tracker" | "os" | "process" | ""
	Resource  *float64 // OpenSearch resource_usage_stats.cpu_utilization_percent (>= 0) or nil
	OS        *int     // os.cpu.percent (>= 0) or nil
	Process   *int     // process.cpu.percent (>= 0) or nil
}

// ResolveNodeCPU applies the per-node CPU fallback chain
// resource_usage_stats -> os -> process -> nil, sanitizing negative sentinels
// (-1) and non-finite values. Elasticsearch has no resource_usage_stats, so its
// effective chain is os -> process -> nil.
func ResolveNodeCPU(resourceRaw *float64, osRaw, procRaw *int) ResolvedNodeCPU {
	resolved := ResolvedNodeCPU{}

	if resourceRaw != nil && !math.IsNaN(*resourceRaw) && !math.IsInf(*resourceRaw, 0) && *resourceRaw >= 0 {
		v := math.Round(*resourceRaw*100) / 100
		resolved.Resource = &v
	}
	if osRaw != nil && *osRaw >= 0 {
		v := *osRaw
		resolved.OS = &v
	}
	if procRaw != nil && *procRaw >= 0 {
		v := *procRaw
		resolved.Process = &v
	}

	switch {
	case resolved.Resource != nil:
		v := *resolved.Resource
		resolved.Effective = &v
		resolved.Source = CPUSourceResourceTracker
	case resolved.OS != nil:
		v := float64(*resolved.OS)
		resolved.Effective = &v
		resolved.Source = CPUSourceOS
	case resolved.Process != nil:
		v := float64(*resolved.Process)
		resolved.Effective = &v
		resolved.Source = CPUSourceProcess
	}
	return resolved
}

// sanitizeLoad drops negative and NaN load averages (unsupported on Windows and
// some container environments).
func sanitizeLoad(load *float64) *float64 {
	if load == nil || math.IsNaN(*load) || *load < 0 {
		return nil
	}
	return load
}

type NodePageEntry struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Host               string            `json:"host"`
	IP                 string            `json:"ip"`
	CurrentMaster      bool              `json:"current_master"`
	Master             bool              `json:"master"`
	Data               bool              `json:"data"`
	Coordinating       bool              `json:"coordinating"`
	Ingest             bool              `json:"ingest"`
	ESVersion          string            `json:"es_version"`
	JVMVersion         string            `json:"jvm_version"`
	LoadAverage        *float64          `json:"load_average"`
	LoadAverage5m      *float64          `json:"load_average_5m,omitempty"`
	LoadAverage15m     *float64          `json:"load_average_15m,omitempty"`
	AvailableProc      int               `json:"available_processors"`
	CPUPercent         *float64          `json:"cpu_percent"`
	CPUSource          string            `json:"cpu_source,omitempty"`
	ResourceCPUPercent *float64          `json:"resource_cpu_percent"`
	OSCPUPercent       *int              `json:"os_cpu_percent"`
	ProcessCPUPercent  *int              `json:"process_cpu_percent"`
	UptimeInMillis     int64             `json:"uptime_in_millis"`
	Attributes         map[string]string `json:"attributes"`
	Heap               heapInfo          `json:"heap"`
	Disk               diskInfo          `json:"disk"`
}

type heapInfo struct {
	Used        int64   `json:"used"`
	Committed   int64   `json:"committed"`
	UsedPercent float64 `json:"used_percent"`
	Max         int64   `json:"max"`
}

type diskInfo struct {
	Total       int64 `json:"total"`
	Free        int64 `json:"free"`
	UsedPercent int   `json:"used_percent"`
}

type roleFlags struct{ master, data, coordinating, ingest bool }

func resolveRole(dataRoles map[string]bool, roles []string) roleFlags {
	r := roleFlags{}
	for _, role := range roles {
		switch role {
		case "master", "cluster_manager":
			r.master = true
		case "ingest":
			r.ingest = true
		}
		if dataRoles[role] {
			r.data = true
		}
	}
	r.coordinating = !r.master && !r.data && !r.ingest
	return r
}

func NodesData(info, stats, master, resourceStats json.RawMessage) (json.RawMessage, error) {
	var infoShape struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	var statsShape struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	var masterShape []struct {
		ID string `json:"id"`
	}
	var resourceShape struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	_ = json.Unmarshal(info, &infoShape)
	_ = json.Unmarshal(stats, &statsShape)
	_ = json.Unmarshal(master, &masterShape)
	_ = json.Unmarshal(resourceStats, &resourceShape)

	masterID := ""
	if len(masterShape) > 0 {
		masterID = masterShape[0].ID
	}

	dataRoles := map[string]bool{
		"data": true, "data_content": true, "data_hot": true,
		"data_warm": true, "data_cold": true, "data_frozen": true, "search": true,
	}

	out := make([]NodePageEntry, 0, len(infoShape.Nodes))
	for id := range infoShape.Nodes {
		out = append(out, buildPageNode(id, masterID, dataRoles, infoShape.Nodes[id], statsShape.Nodes[id], resourceShape.Nodes[id]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return json.Marshal(out)
}

func buildPageNode(id, masterID string, dataRoles map[string]bool, infoRaw, statsRaw, resourceRaw json.RawMessage) NodePageEntry {
	var is struct {
		Name       string            `json:"name"`
		Version    string            `json:"version"`
		Attributes map[string]string `json:"attributes"`
		JVM        struct {
			Version string `json:"version"`
		} `json:"jvm"`
		OS struct {
			AvailableProcessors int `json:"available_processors"`
		} `json:"os"`
	}
	var ss struct {
		Name  string   `json:"name"`
		Host  string   `json:"host"`
		IP    string   `json:"ip"`
		Roles []string `json:"roles"`
		OS    struct {
			CPU struct {
				Percent     *int `json:"percent"`
				LoadAverage struct {
					OneM     *float64 `json:"1m"`
					FiveM    *float64 `json:"5m"`
					FifteenM *float64 `json:"15m"`
				} `json:"load_average"`
			} `json:"cpu"`
		} `json:"os"`
		Process struct {
			CPU struct {
				Percent *int `json:"percent"`
			} `json:"cpu"`
		} `json:"process"`
		JVM struct {
			UptimeInMillis int64 `json:"uptime_in_millis"`
			Mem            struct {
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
	var rs struct {
		ResourceUsageStats struct {
			CPUUtilizationPercent *float64 `json:"cpu_utilization_percent"`
		} `json:"resource_usage_stats"`
	}
	_ = json.Unmarshal(infoRaw, &is)
	_ = json.Unmarshal(statsRaw, &ss)
	_ = json.Unmarshal(resourceRaw, &rs)

	roles := resolveRole(dataRoles, ss.Roles)

	diskPct := 0
	if ss.FS.Total.TotalInBytes > 0 {
		available := ss.FS.Total.AvailableInBytes
		if available == 0 {
			available = ss.FS.Total.FreeInBytes
		}
		diskPct = 100 - int(100*float64(available)/float64(ss.FS.Total.TotalInBytes))
	}

	cpu := ResolveNodeCPU(rs.ResourceUsageStats.CPUUtilizationPercent, ss.OS.CPU.Percent, ss.Process.CPU.Percent)

	attributes := map[string]string{}
	for k, v := range is.Attributes {
		if internalNodeAttributes[k] {
			continue
		}
		attributes[k] = v
	}

	return NodePageEntry{
		ID:                 id,
		Name:               ss.Name,
		Host:               ss.Host,
		IP:                 ss.IP,
		CurrentMaster:      id == masterID,
		Master:             roles.master,
		Data:               roles.data,
		Coordinating:       roles.coordinating,
		Ingest:             roles.ingest,
		ESVersion:          is.Version,
		JVMVersion:         is.JVM.Version,
		LoadAverage:        sanitizeLoad(ss.OS.CPU.LoadAverage.OneM),
		LoadAverage5m:      sanitizeLoad(ss.OS.CPU.LoadAverage.FiveM),
		LoadAverage15m:     sanitizeLoad(ss.OS.CPU.LoadAverage.FifteenM),
		AvailableProc:      is.OS.AvailableProcessors,
		CPUPercent:         cpu.Effective,
		CPUSource:          cpu.Source,
		ResourceCPUPercent: cpu.Resource,
		OSCPUPercent:       cpu.OS,
		ProcessCPUPercent:  cpu.Process,
		UptimeInMillis:     ss.JVM.UptimeInMillis,
		Attributes:         attributes,
		Heap: heapInfo{
			Used:        ss.JVM.Mem.HeapUsedInBytes,
			Committed:   ss.JVM.Mem.HeapCommittedInBytes,
			UsedPercent: ss.JVM.Mem.HeapUsedPercent,
			Max:         ss.JVM.Mem.HeapMaxInBytes,
		},
		Disk: diskInfo{
			Total:       ss.FS.Total.TotalInBytes,
			Free:        ss.FS.Total.FreeInBytes,
			UsedPercent: diskPct,
		},
	}
}
