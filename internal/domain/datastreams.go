package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DataStreamSummary is the normalized representation of one data stream,
// shared by Elasticsearch 8/9 and OpenSearch 2/3.
type DataStreamSummary struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"` // lowercased: "green" | "yellow" | "red"
	Generation       int      `json:"generation"`
	Template         string   `json:"template"`
	TimestampField   string   `json:"timestamp_field"`
	WriteIndex       string   `json:"write_index"`
	BackingIndices   []string `json:"backing_indices"`
	DocCount         int64    `json:"doc_count"`
	SizeInBytes      int64    `json:"size_in_bytes"`
	TotalSizeInBytes int64    `json:"total_size_in_bytes"`
	Hidden           bool     `json:"hidden,omitempty"`
	System           bool     `json:"system,omitempty"`
}

// DataStreamBackingMeta maps a backing index to its parent data stream.
type DataStreamBackingMeta struct {
	DataStream     string `json:"data_stream"`
	IsWriteIndex   bool   `json:"is_write_index"`
	HiddenOrSystem bool   `json:"hidden_or_system"`
}

var backingIndexGeneration = regexp.MustCompile(`-\d{6}$`)

// ParseDataStreams normalizes a GET /_data_stream response from Elasticsearch
// 8/9 or OpenSearch 2/3. It returns the streams plus a lookup from every
// backing index name to its parent stream metadata.
func ParseDataStreams(raw json.RawMessage) ([]DataStreamSummary, map[string]DataStreamBackingMeta, error) {
	backing := map[string]DataStreamBackingMeta{}
	if len(raw) == 0 || string(raw) == "null" {
		return []DataStreamSummary{}, backing, nil
	}

	var payload struct {
		DataStreams []struct {
			Name           string `json:"name"`
			Generation     int    `json:"generation"`
			Status         string `json:"status"`
			Template       string `json:"template"`
			Hidden         bool   `json:"hidden"`
			System         bool   `json:"system"`
			TimestampField struct {
				Name string `json:"name"`
			} `json:"timestamp_field"`
			Indices []struct {
				IndexName string `json:"index_name"`
			} `json:"indices"`
		} `json:"data_streams"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil, err
	}

	out := make([]DataStreamSummary, 0, len(payload.DataStreams))
	for _, ds := range payload.DataStreams {
		names := make([]string, 0, len(ds.Indices))
		for _, idx := range ds.Indices {
			if idx.IndexName != "" {
				names = append(names, idx.IndexName)
			}
		}
		writeIndex := ""
		if len(names) > 0 {
			writeIndex = names[len(names)-1]
		}
		hiddenOrSystem := ds.Hidden || ds.System || strings.HasPrefix(ds.Name, ".")
		for i, name := range names {
			backing[name] = DataStreamBackingMeta{
				DataStream:     ds.Name,
				IsWriteIndex:   i == len(names)-1,
				HiddenOrSystem: hiddenOrSystem,
			}
		}
		out = append(out, DataStreamSummary{
			Name:           ds.Name,
			Status:         strings.ToLower(ds.Status),
			Generation:     ds.Generation,
			Template:       ds.Template,
			TimestampField: ds.TimestampField.Name,
			WriteIndex:     writeIndex,
			BackingIndices: names,
			Hidden:         ds.Hidden,
			System:         ds.System,
		})
	}
	return out, backing, nil
}

// DataStreamNameFromBackingIndex derives the parent data stream name from a
// `.ds-<stream>-<generation>` backing index. It is the fallback heuristic used
// when GET /_data_stream is unavailable (e.g. restricted permissions).
func DataStreamNameFromBackingIndex(index string) (string, bool) {
	if !strings.HasPrefix(index, ".ds-") {
		return "", false
	}
	name := strings.TrimPrefix(index, ".ds-")
	name = backingIndexGeneration.ReplaceAllString(name, "")
	if name == "" {
		return "", false
	}
	return name, true
}

// ValidateDataStreamName enforces the data stream naming rules shared by
// Elasticsearch 8/9 and OpenSearch 2/3.
func ValidateDataStreamName(name string) error {
	if name == "" {
		return fmt.Errorf("data stream name is required")
	}
	if name != strings.ToLower(name) {
		return fmt.Errorf("data stream name %q must be lowercase", name)
	}
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") || strings.HasPrefix(name, "+") {
		return fmt.Errorf("data stream name %q cannot start with '-', '_' or '+'", name)
	}
	if strings.HasPrefix(name, ".ds-") {
		return fmt.Errorf("data stream name %q cannot start with '.ds-'", name)
	}
	if strings.ContainsAny(name, " \t\r\n*?/\\,<>|\"#:") {
		return fmt.Errorf("data stream name %q cannot contain spaces, wildcards, slashes or reserved characters", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("data stream name %q is not allowed", name)
	}
	return nil
}
