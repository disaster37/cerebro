package domain

import (
	"encoding/json"
)

func Repositories(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}

	type entry struct {
		Name     string          `json:"name"`
		Type     json.RawMessage `json:"type"`
		Settings json.RawMessage `json:"settings"`
	}
	out := make([]entry, 0, len(raw))
	for name, v := range raw {
		var info struct {
			Type     json.RawMessage `json:"type"`
			Settings json.RawMessage `json:"settings"`
		}
		if err := json.Unmarshal(v, &info); err != nil {
			return nil, err
		}
		settings := json.RawMessage("{}")
		if len(info.Settings) > 0 {
			settings = info.Settings
		}
		out = append(out, entry{Name: name, Type: info.Type, Settings: settings})
	}
	return json.Marshal(out)
}

// ParseRepositoryVerification extracts the verified nodes from a
// POST /_snapshot/{repo}/_verify response.
func ParseRepositoryVerification(in json.RawMessage) ([]VerifiedRepoNode, error) {
	var raw struct {
		Nodes map[string]struct {
			Name string `json:"name"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}
	out := make([]VerifiedRepoNode, 0, len(raw.Nodes))
	for id, node := range raw.Nodes {
		out = append(out, VerifiedRepoNode{ID: id, Name: node.Name})
	}
	return out, nil
}

// RepositorySizeStats is the aggregated size of a repository's snapshots.
type RepositorySizeStats struct {
	SizeInBytes        int64
	LogicalSizeInBytes int64
	HasStats           bool
}

type byteCounter struct {
	SizeInBytes *int64 `json:"size_in_bytes"`
}

// snapshotStatsWire covers both the nested _status shape
// (incremental/processed/total) and the flat SnapshotInfo shape
// (processed_size_in_bytes/total_size_in_bytes).
type snapshotStatsWire struct {
	Incremental          *byteCounter `json:"incremental"`
	Processed            *byteCounter `json:"processed"`
	Total                *byteCounter `json:"total"`
	ProcessedSizeInBytes *int64       `json:"processed_size_in_bytes"`
	TotalSizeInBytes     *int64       `json:"total_size_in_bytes"`
}

// ParseRepositorySizeStats sums incremental snapshot sizes and takes the max
// total (logical) size from a GET /_snapshot/{repo}/_all or _all/_status body.
// HasStats is only true when real byte statistics are present.
func ParseRepositorySizeStats(in json.RawMessage) (RepositorySizeStats, error) {
	var raw struct {
		Snapshots []struct {
			Stats *snapshotStatsWire `json:"stats"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(in, &raw); err != nil {
		return RepositorySizeStats{}, err
	}
	out := RepositorySizeStats{}
	var processedSum, maxTotal int64
	for _, snap := range raw.Snapshots {
		if snap.Stats == nil {
			continue
		}
		stats := snap.Stats
		hasBytes := false

		var incremental int64
		if stats.Incremental != nil && stats.Incremental.SizeInBytes != nil {
			incremental = *stats.Incremental.SizeInBytes
			hasBytes = true
		} else if stats.ProcessedSizeInBytes != nil {
			incremental = *stats.ProcessedSizeInBytes
			hasBytes = true
		}
		if stats.Processed != nil && stats.Processed.SizeInBytes != nil {
			processedSum += *stats.Processed.SizeInBytes
			hasBytes = true
		}

		var total int64
		if stats.Total != nil && stats.Total.SizeInBytes != nil {
			total = *stats.Total.SizeInBytes
			hasBytes = true
		}
		if stats.TotalSizeInBytes != nil {
			if *stats.TotalSizeInBytes > total {
				total = *stats.TotalSizeInBytes
			}
			hasBytes = true
		}
		if total > maxTotal {
			maxTotal = total
		}
		if hasBytes {
			out.HasStats = true
		}
		out.SizeInBytes += incremental
	}
	if out.SizeInBytes == 0 && maxTotal > 0 {
		// A deleted base snapshot can leave only zero-incremental snapshots
		// behind; fall back to the processed/total bytes so a non-empty
		// repository never reports 0 B.
		if processedSum > 0 {
			out.SizeInBytes = processedSum
		} else {
			out.SizeInBytes = maxTotal
		}
	}
	out.LogicalSizeInBytes = maxTotal
	return out, nil
}

func SnapshotRepoList(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return json.Marshal([]string{})
	}
	out := make([]string, 0, len(raw))
	for name := range raw {
		out = append(out, name)
	}
	return json.Marshal(out)
}

func SnapshotIndices(in json.RawMessage) (json.RawMessage, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(in, &arr); err != nil {
		return nil, err
	}
	type entry struct {
		Name    string `json:"name"`
		Special bool   `json:"special"`
	}
	out := make([]entry, 0, len(arr))
	for _, m := range arr {
		var name string
		if nm, ok := m["index"]; ok {
			_ = json.Unmarshal(nm, &name)
		}
		out = append(out, entry{Name: name, Special: len(name) > 0 && name[0] == '.'})
	}
	return json.Marshal(out)
}
