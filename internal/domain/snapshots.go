package domain

import "encoding/json"

// SnapshotShardStats is the shard success/failure summary of a snapshot.
type SnapshotShardStats struct {
	Total      int `json:"total"`
	Failed     int `json:"failed"`
	Successful int `json:"successful"`
}

// SnapshotFailure is one shard failure recorded in a snapshot.
type SnapshotFailure struct {
	Index   string `json:"index"`
	ShardID int    `json:"shard_id"`
	NodeID  string `json:"node_id"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
}

// SnapshotSummary preserves the snapshot metadata the UI needs (state,
// timing, shards, failures, global state and data streams).
type SnapshotSummary struct {
	Snapshot           string             `json:"snapshot"`
	UUID               string             `json:"uuid"`
	Version            string             `json:"version"`
	Indices            []string           `json:"indices"`
	DataStreams        []string           `json:"data_streams"`
	IncludeGlobalState bool               `json:"include_global_state"`
	State              string             `json:"state"`
	StartTimeInMillis  int64              `json:"start_time_in_millis"`
	EndTimeInMillis    int64              `json:"end_time_in_millis"`
	DurationInMillis   int64              `json:"duration_in_millis"`
	Failures           []SnapshotFailure  `json:"failures"`
	Shards             SnapshotShardStats `json:"shards"`
}

func Snapshots(in json.RawMessage) (json.RawMessage, error) {
	var raw struct {
		Snapshots []SnapshotSummary `json:"snapshots"`
	}
	if err := json.Unmarshal(in, &raw); err != nil {
		return json.RawMessage("[]"), nil
	}
	if raw.Snapshots == nil {
		return json.RawMessage("[]"), nil
	}
	for i := range raw.Snapshots {
		if raw.Snapshots[i].Indices == nil {
			raw.Snapshots[i].Indices = []string{}
		}
		if raw.Snapshots[i].DataStreams == nil {
			raw.Snapshots[i].DataStreams = []string{}
		}
		if raw.Snapshots[i].Failures == nil {
			raw.Snapshots[i].Failures = []SnapshotFailure{}
		}
	}
	return json.Marshal(raw.Snapshots)
}
