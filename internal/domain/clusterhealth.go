package domain

import "encoding/json"

// NOTE: currently unused — handlers use the ElasticClient.ClusterHealth method
// instead of this transform. Kept for parity; do not delete in this refactor.
func ClusterHealth(in json.RawMessage) (json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(in, &raw); err != nil {
		return nil, err
	}
	fields := []string{
		"cluster_name", "status", "number_of_nodes",
		"active_primary_shards", "active_shards", "relocating_shards",
		"initializing_shards", "unassigned_shards", "delayed_unassigned_shards",
		"number_of_pending_tasks", "number_of_in_flight_fetch",
		"task_max_waiting_in_queue_millis", "active_shards_percent_as_number",
	}
	out := make(map[string]json.RawMessage)
	for _, f := range fields {
		if v, ok := raw[f]; ok {
			out[f] = v
		} else {
			out[f] = json.RawMessage(`null`)
		}
	}
	return json.Marshal(out)
}
