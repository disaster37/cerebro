package domain

import (
	"encoding/json"
	"sort"
)

type NodePageEntry struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Host              string            `json:"host"`
	IP                string            `json:"ip"`
	CurrentMaster     bool              `json:"current_master"`
	Master            bool              `json:"master"`
	Data              bool              `json:"data"`
	Coordinating      bool              `json:"coordinating"`
	Ingest            bool              `json:"ingest"`
	ESVersion         string            `json:"es_version"`
	JVMVersion        string            `json:"jvm_version"`
	LoadAverage       *float64          `json:"load_average"`
	LoadAverage5m     *float64          `json:"load_average_5m,omitempty"`
	LoadAverage15m    *float64          `json:"load_average_15m,omitempty"`
	AvailableProc     int               `json:"available_processors"`
	CPUPercent        int               `json:"cpu_percent"`
	ProcessCPUPercent int               `json:"process_cpu_percent"`
	UptimeInMillis    int64             `json:"uptime_in_millis"`
	Attributes        map[string]string `json:"attributes"`
	Heap              heapInfo          `json:"heap"`
	Disk              diskInfo          `json:"disk"`
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

func NodesData(info, stats, master json.RawMessage) (json.RawMessage, error) {
	var infoShape struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	var statsShape struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	var masterShape []struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(info, &infoShape)
	_ = json.Unmarshal(stats, &statsShape)
	_ = json.Unmarshal(master, &masterShape)

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
		out = append(out, buildPageNode(id, masterID, dataRoles, infoShape.Nodes[id], statsShape.Nodes[id]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return json.Marshal(out)
}

func buildPageNode(id, masterID string, dataRoles map[string]bool, infoRaw, statsRaw json.RawMessage) NodePageEntry {
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
				Percent     int `json:"percent"`
				LoadAverage struct {
					OneM     *float64 `json:"1m"`
					FiveM    *float64 `json:"5m"`
					FifteenM *float64 `json:"15m"`
				} `json:"load_average"`
			} `json:"cpu"`
		} `json:"os"`
		Process struct {
			CPU struct {
				Percent int `json:"percent"`
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
	_ = json.Unmarshal(infoRaw, &is)
	_ = json.Unmarshal(statsRaw, &ss)

	roles := resolveRole(dataRoles, ss.Roles)

	diskPct := 0
	if ss.FS.Total.TotalInBytes > 0 {
		available := ss.FS.Total.AvailableInBytes
		if available == 0 {
			available = ss.FS.Total.FreeInBytes
		}
		diskPct = 100 - int(100*float64(available)/float64(ss.FS.Total.TotalInBytes))
	}

	cpuPercent := ss.OS.CPU.Percent
	if cpuPercent == 0 && ss.Process.CPU.Percent > 0 {
		cpuPercent = ss.Process.CPU.Percent
	}

	attributes := map[string]string{}
	for k, v := range is.Attributes {
		if internalNodeAttributes[k] {
			continue
		}
		attributes[k] = v
	}

	return NodePageEntry{
		ID:                id,
		Name:              ss.Name,
		Host:              ss.Host,
		IP:                ss.IP,
		CurrentMaster:     id == masterID,
		Master:            roles.master,
		Data:              roles.data,
		Coordinating:      roles.coordinating,
		Ingest:            roles.ingest,
		ESVersion:         is.Version,
		JVMVersion:        is.JVM.Version,
		LoadAverage:       ss.OS.CPU.LoadAverage.OneM,
		LoadAverage5m:     ss.OS.CPU.LoadAverage.FiveM,
		LoadAverage15m:    ss.OS.CPU.LoadAverage.FifteenM,
		AvailableProc:     is.OS.AvailableProcessors,
		CPUPercent:        cpuPercent,
		ProcessCPUPercent: ss.Process.CPU.Percent,
		UptimeInMillis:    ss.JVM.UptimeInMillis,
		Attributes:        attributes,
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
