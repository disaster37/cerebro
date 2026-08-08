package models

import "encoding/json"

type NodePageEntry struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	IP            string   `json:"ip"`
	CurrentMaster bool     `json:"current_master"`
	Master        bool     `json:"master"`
	Data          bool     `json:"data"`
	Coordinating  bool     `json:"coordinating"`
	Ingest        bool     `json:"ingest"`
	ESVersion     string   `json:"es_version"`
	JVMVersion    string   `json:"jvm_version"`
	LoadAverage   float64  `json:"load_average"`
	AvailableProc int      `json:"available_processors"`
	CPUPercent    int      `json:"cpu_percent"`
	Heap          heapInfo `json:"heap"`
	Disk          diskInfo `json:"disk"`
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
		case "master":
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
		"data_warm": true, "data_cold": true,
	}

	var out []NodePageEntry
	for id := range infoShape.Nodes {
		out = append(out, buildPageNode(id, masterID, dataRoles, infoShape.Nodes[id], statsShape.Nodes[id]))
	}
	return json.Marshal(out)
}

func buildPageNode(id, masterID string, dataRoles map[string]bool, infoRaw, statsRaw json.RawMessage) NodePageEntry {
	var is struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		JVM     struct {
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
				Percent int `json:"percent"`
			} `json:"cpu"`
		} `json:"os"`
		Process struct {
			CPU struct {
				Percent int `json:"percent"`
			} `json:"cpu"`
		} `json:"process"`
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
		LoadAverage []float64 `json:"os.cpu.load_average"`
	}
	_ = json.Unmarshal(infoRaw, &is)
	_ = json.Unmarshal(statsRaw, &ss)

	roles := resolveRole(dataRoles, ss.Roles)

	diskPct := 0
	if ss.FS.Total.TotalInBytes > 0 {
		diskPct = 100 - int(100*float64(ss.FS.Total.FreeInBytes)/float64(ss.FS.Total.TotalInBytes))
	}

	load := 0.0
	if len(ss.LoadAverage) > 0 {
		load = ss.LoadAverage[0]
	}

	return NodePageEntry{
		ID:            id,
		Name:          ss.Name,
		Host:          ss.Host,
		IP:            ss.IP,
		CurrentMaster: id == masterID,
		Master:        roles.master,
		Data:          roles.data,
		Coordinating:  roles.coordinating,
		Ingest:        roles.ingest,
		ESVersion:     is.Version,
		JVMVersion:    is.JVM.Version,
		LoadAverage:   load,
		AvailableProc: is.OS.AvailableProcessors,
		CPUPercent:    ss.Process.CPU.Percent,
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
