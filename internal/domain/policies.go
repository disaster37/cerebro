package domain

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type IndexLifecyclePayload struct {
	BackendType    string               `json:"backendType"`
	FeatureName    string               `json:"featureName"`
	OperationMode  string               `json:"operationMode"`
	Policies       []IndexPolicySummary `json:"policies"`
	ManagedIndices []ManagedIndexStatus `json:"managedIndices"`
	Unsupported    bool                 `json:"unsupported,omitempty"`
	Warning        string               `json:"warning,omitempty"`
}

type IndexPolicySummary struct {
	ID             string          `json:"id"`
	Version        int64           `json:"version"`
	ModifiedDate   string          `json:"modified_date"`
	PhasesOrStates []string        `json:"phases"`
	ManagedCount   int             `json:"managed_count"`
	FailedCount    int             `json:"failed_count"`
	RawPolicy      json.RawMessage `json:"raw_policy"`
}

type ManagedIndexStatus struct {
	Index           string   `json:"index"`
	PolicyID        string   `json:"policy_id"`
	PhaseOrState    string   `json:"phase"`
	Action          string   `json:"action"`
	Step            string   `json:"step"`
	Failed          bool     `json:"failed"`
	FailedStep      string   `json:"failed_step,omitempty"`
	RetryCount      int      `json:"retry_count"`
	Reason          string   `json:"reason,omitempty"`
	Age             string   `json:"age,omitempty"`
	AvailableStates []string `json:"available_states,omitempty"`
}

type SnapshotPolicySummary struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Repository      string          `json:"repository"`
	Schedule        string          `json:"schedule"`
	Indices         []string        `json:"indices"`
	Enabled         bool            `json:"enabled"`
	CurrentState    string          `json:"current_state"`
	LastSuccessName string          `json:"last_success_name,omitempty"`
	LastSuccessTime *int64          `json:"last_success_time,omitempty"`
	LastFailureName string          `json:"last_failure_name,omitempty"`
	LastFailureTime *int64          `json:"last_failure_time,omitempty"`
	LastFailureMsg  string          `json:"last_failure_message,omitempty"`
	NextExecution   *int64          `json:"next_execution_millis,omitempty"`
	SnapshotsTaken  int64           `json:"snapshots_taken"`
	SnapshotsFailed int64           `json:"snapshots_failed"`
	RawPolicy       json.RawMessage `json:"raw_policy"`
}

// NormalizeIndexLifecycle converts ES ILM or OpenSearch ISM responses into the
// unified index lifecycle payload.
func NormalizeIndexLifecycle(backendType string, policies, explain, status json.RawMessage) (json.RawMessage, error) {
	if backendType == "opensearch" {
		return normalizeISM(policies, explain, status)
	}
	return normalizeILM(policies, explain, status)
}

func normalizeILM(policies, explain, status json.RawMessage) (json.RawMessage, error) {
	var rawPolicies map[string]struct {
		Version      int64  `json:"version"`
		ModifiedDate string `json:"modified_date"`
		Policy       struct {
			Phases map[string]json.RawMessage `json:"phases"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(policies, &rawPolicies); err != nil {
		return nil, err
	}

	var rawExplain struct {
		Indices map[string]struct {
			Index      string `json:"index"`
			Managed    bool   `json:"managed"`
			Policy     string `json:"policy"`
			Phase      string `json:"phase"`
			Action     string `json:"action"`
			Step       string `json:"step"`
			FailedStep string `json:"failed_step"`
			StepInfo   struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"step_info"`
			Age string `json:"age"`
		} `json:"indices"`
	}
	_ = json.Unmarshal(explain, &rawExplain)

	var rawStatus struct {
		OperationMode string `json:"operation_mode"`
	}
	_ = json.Unmarshal(status, &rawStatus)

	managed := make([]ManagedIndexStatus, 0, len(rawExplain.Indices))
	managedByPolicy := map[string]int{}
	failedByPolicy := map[string]int{}
	for _, entry := range rawExplain.Indices {
		if !entry.Managed || entry.Policy == "" {
			continue
		}
		failed := entry.Step == "ERROR" || entry.FailedStep != ""
		reason := entry.StepInfo.Reason
		if reason == "" {
			reason = entry.StepInfo.Type
		}
		managed = append(managed, ManagedIndexStatus{
			Index:        firstNonEmpty(entry.Index, entry.Policy),
			PolicyID:     entry.Policy,
			PhaseOrState: entry.Phase,
			Action:       entry.Action,
			Step:         entry.Step,
			Failed:       failed,
			FailedStep:   entry.FailedStep,
			Reason:       reason,
			Age:          entry.Age,
		})
		managedByPolicy[entry.Policy]++
		if failed {
			failedByPolicy[entry.Policy]++
		}
	}
	sort.Slice(managed, func(i, j int) bool { return managed[i].Index < managed[j].Index })

	summaries := make([]IndexPolicySummary, 0, len(rawPolicies))
	for id, p := range rawPolicies {
		phases := make([]string, 0, len(p.Policy.Phases))
		for phase := range p.Policy.Phases {
			phases = append(phases, phase)
		}
		sort.Strings(phases)
		raw, _ := json.Marshal(p)
		summaries = append(summaries, IndexPolicySummary{
			ID:             id,
			Version:        p.Version,
			ModifiedDate:   p.ModifiedDate,
			PhasesOrStates: phases,
			ManagedCount:   managedByPolicy[id],
			FailedCount:    failedByPolicy[id],
			RawPolicy:      raw,
		})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].ID < summaries[j].ID })

	payload := IndexLifecyclePayload{
		BackendType:    "elasticsearch",
		FeatureName:    "ILM",
		OperationMode:  firstNonEmpty(rawStatus.OperationMode, "RUNNING"),
		Policies:       summaries,
		ManagedIndices: managed,
	}
	return json.Marshal(payload)
}

func normalizeISM(policies, explain, status json.RawMessage) (json.RawMessage, error) {
	var rawPolicies struct {
		Policies []struct {
			ID     string `json:"_id"`
			Policy struct {
				PolicyID        string `json:"policy_id"`
				LastUpdatedTime int64  `json:"last_updated_time"`
				States          []struct {
					Name string `json:"name"`
				} `json:"states"`
			} `json:"policy"`
		} `json:"policies"`
	}
	if err := json.Unmarshal(policies, &rawPolicies); err != nil {
		return nil, err
	}

	statesByPolicy := map[string][]string{}
	summaries := make([]IndexPolicySummary, 0, len(rawPolicies.Policies))
	for _, p := range rawPolicies.Policies {
		id := firstNonEmpty(p.Policy.PolicyID, p.ID)
		states := make([]string, 0, len(p.Policy.States))
		for _, s := range p.Policy.States {
			states = append(states, s.Name)
		}
		statesByPolicy[id] = states
		raw, _ := json.Marshal(p)
		summaries = append(summaries, IndexPolicySummary{
			ID:             id,
			Version:        1,
			ModifiedDate:   millisToRFC3339(p.Policy.LastUpdatedTime),
			PhasesOrStates: states,
			RawPolicy:      raw,
		})
	}

	entries := ismExplainEntries(explain)
	managed := make([]ManagedIndexStatus, 0, len(entries))
	managedByPolicy := map[string]int{}
	failedByPolicy := map[string]int{}
	for _, entry := range entries {
		if entry.PolicyID == "" {
			continue
		}
		failed := entry.Action.Failed || entry.RetryInfo.Failed ||
			entry.Step.StepStatus == "failed" || entry.Step.StepStatus == "time_limit_exceeded"
		managed = append(managed, ManagedIndexStatus{
			Index:           entry.Index,
			PolicyID:        entry.PolicyID,
			PhaseOrState:    entry.State.Name,
			Action:          entry.Action.Name,
			Step:            entry.Step.Name,
			Failed:          failed,
			RetryCount:      entry.RetryInfo.ConsumedRetries,
			Reason:          joinNonEmpty(entry.Info.Message, entry.Info.Cause),
			AvailableStates: statesByPolicy[entry.PolicyID],
		})
		managedByPolicy[entry.PolicyID]++
		if failed {
			failedByPolicy[entry.PolicyID]++
		}
	}
	sort.Slice(managed, func(i, j int) bool { return managed[i].Index < managed[j].Index })

	for i := range summaries {
		summaries[i].ManagedCount = managedByPolicy[summaries[i].ID]
		summaries[i].FailedCount = failedByPolicy[summaries[i].ID]
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].ID < summaries[j].ID })

	payload := IndexLifecyclePayload{
		BackendType:    "opensearch",
		FeatureName:    "ISM",
		OperationMode:  ismOperationMode(status),
		Policies:       summaries,
		ManagedIndices: managed,
	}
	return json.Marshal(payload)
}

type ismExplainEntry struct {
	Index    string `json:"index"`
	PolicyID string `json:"policy_id"`
	State    struct {
		Name string `json:"name"`
	} `json:"state"`
	Action struct {
		Name   string `json:"name"`
		Failed bool   `json:"failed"`
	} `json:"action"`
	Step struct {
		Name       string `json:"name"`
		StepStatus string `json:"step_status"`
	} `json:"step"`
	RetryInfo struct {
		Failed          bool `json:"failed"`
		ConsumedRetries int  `json:"consumed_retries"`
	} `json:"retry_info"`
	Info struct {
		Message string `json:"message"`
		Cause   string `json:"cause"`
	} `json:"info"`
}

func ismExplainEntries(explain json.RawMessage) []ismExplainEntry {
	var wrapped struct {
		Indices map[string]json.RawMessage `json:"indices"`
	}
	if err := json.Unmarshal(explain, &wrapped); err == nil && wrapped.Indices != nil {
		return ismEntriesFromRaw(wrapped.Indices)
	}
	var direct map[string]json.RawMessage
	if err := json.Unmarshal(explain, &direct); err == nil {
		return ismEntriesFromRaw(direct)
	}
	return []ismExplainEntry{}
}

// ismEntriesFromRaw decodes each raw value individually, skipping non-object
// entries such as the numeric "total_managed_indices" field OpenSearch
// includes alongside the index entries.
func ismEntriesFromRaw(raw map[string]json.RawMessage) []ismExplainEntry {
	out := make([]ismExplainEntry, 0, len(raw))
	for name, value := range raw {
		var entry ismExplainEntry
		if err := json.Unmarshal(value, &entry); err != nil {
			continue
		}
		if entry.Index == "" {
			entry.Index = name
		}
		out = append(out, entry)
	}
	return out
}

func ismOperationMode(status json.RawMessage) string {
	var raw struct {
		Persistent map[string]json.RawMessage `json:"persistent"`
		Defaults   map[string]json.RawMessage `json:"defaults"`
	}
	_ = json.Unmarshal(status, &raw)
	for _, m := range []map[string]json.RawMessage{raw.Persistent, raw.Defaults} {
		if v, ok := m["plugins.index_state_management.enabled"]; ok {
			if enabled, ok := ismBool(v); ok {
				if enabled {
					return "RUNNING"
				}
				return "STOPPED"
			}
		}
	}
	return "RUNNING"
}

// ismBool accepts both the JSON boolean form and the flat_settings string
// form ("true"/"false", case-insensitive) that OpenSearch returns.
func ismBool(value json.RawMessage) (bool, bool) {
	var enabled bool
	if err := json.Unmarshal(value, &enabled); err == nil {
		return enabled, true
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

// NormalizeSnapshotLifecycle converts ES SLM or OpenSearch SM responses into
// the unified snapshot policy list.
func NormalizeSnapshotLifecycle(backendType string, policies, explain json.RawMessage) (json.RawMessage, error) {
	if backendType == "opensearch" {
		return normalizeSM(policies, explain)
	}
	return normalizeSLM(policies, explain)
}

func normalizeSLM(policies, explain json.RawMessage) (json.RawMessage, error) {
	var rawPolicies map[string]struct {
		Version      int64  `json:"version"`
		ModifiedDate string `json:"modified_date"`
		Policy       struct {
			Name       string `json:"name"`
			Schedule   string `json:"schedule"`
			Repository string `json:"repository"`
			Config     struct {
				Indices []string `json:"indices"`
			} `json:"config"`
		} `json:"policy"`
		Stats struct {
			SnapshotsTaken  int64 `json:"snapshots_taken"`
			SnapshotsFailed int64 `json:"snapshots_failed"`
		} `json:"stats"`
		LastSuccess *struct {
			SnapshotName string `json:"snapshot_name"`
			Time         int64  `json:"time"`
			Details      string `json:"details"`
		} `json:"last_success"`
		LastFailure *struct {
			SnapshotName string `json:"snapshot_name"`
			Time         int64  `json:"time"`
			Details      string `json:"details"`
		} `json:"last_failure"`
		NextExecution string `json:"next_execution"`
	}
	if err := json.Unmarshal(policies, &rawPolicies); err != nil {
		return nil, err
	}

	var rawStatus struct {
		OperationMode string `json:"operation_mode"`
	}
	_ = json.Unmarshal(explain, &rawStatus)
	state := firstNonEmpty(rawStatus.OperationMode, "RUNNING")

	out := make([]SnapshotPolicySummary, 0, len(rawPolicies))
	for id, p := range rawPolicies {
		raw, _ := json.Marshal(p)
		summary := SnapshotPolicySummary{
			ID:              id,
			Name:            firstNonEmpty(p.Policy.Name, id),
			Repository:      p.Policy.Repository,
			Schedule:        p.Policy.Schedule,
			Indices:         nonNilStrings(p.Policy.Config.Indices),
			Enabled:         true,
			CurrentState:    state,
			SnapshotsTaken:  p.Stats.SnapshotsTaken,
			SnapshotsFailed: p.Stats.SnapshotsFailed,
			RawPolicy:       raw,
		}
		if p.LastSuccess != nil {
			summary.LastSuccessName = p.LastSuccess.SnapshotName
			summary.LastSuccessTime = int64Ptr(p.LastSuccess.Time)
		}
		if p.LastFailure != nil && p.LastFailure.Time > 0 {
			summary.LastFailureName = p.LastFailure.SnapshotName
			summary.LastFailureTime = int64Ptr(p.LastFailure.Time)
			summary.LastFailureMsg = p.LastFailure.Details
		}
		if p.NextExecution != "" {
			if t, err := time.Parse(time.RFC3339, p.NextExecution); err == nil {
				summary.NextExecution = int64Ptr(t.UnixMilli())
			}
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return json.Marshal(out)
}

func normalizeSM(policies, explain json.RawMessage) (json.RawMessage, error) {
	var rawPolicies struct {
		Policies []struct {
			ID     string `json:"_id"`
			Policy struct {
				PolicyID       string            `json:"policy_id"`
				Description    string            `json:"description"`
				Enabled        bool              `json:"enabled"`
				Creation       smExecutionConfig `json:"creation"`
				SnapshotConfig struct {
					Repository string `json:"repository"`
					Indices    string `json:"indices"`
				} `json:"snapshot_config"`
			} `json:"policy"`
		} `json:"policies"`
	}
	if err := json.Unmarshal(policies, &rawPolicies); err != nil {
		return nil, err
	}

	explainByID := map[string]smExplain{}
	var rawExplain struct {
		Policies []smExplain `json:"policies"`
	}
	if err := json.Unmarshal(explain, &rawExplain); err == nil {
		for _, e := range rawExplain.Policies {
			explainByID[e.ID] = e
		}
	}

	out := make([]SnapshotPolicySummary, 0, len(rawPolicies.Policies))
	for _, p := range rawPolicies.Policies {
		id := firstNonEmpty(p.Policy.PolicyID, p.ID)
		raw, _ := json.Marshal(p)
		summary := SnapshotPolicySummary{
			ID:           id,
			Name:         firstNonEmpty(p.Policy.Description, id),
			Repository:   p.Policy.SnapshotConfig.Repository,
			Schedule:     p.Policy.Creation.Schedule.Cron.Expression,
			Indices:      splitIndices(p.Policy.SnapshotConfig.Indices),
			Enabled:      p.Policy.Enabled,
			CurrentState: "IDLE",
			RawPolicy:    raw,
		}
		if e, ok := explainByID[id]; ok {
			summary.CurrentState = firstNonEmpty(e.Creation.LatestExecution.Status, "IDLE")
			exec := e.Creation.LatestExecution
			switch exec.Status {
			case "SUCCESS":
				summary.LastSuccessName = exec.SnapshotName
				summary.LastSuccessTime = int64Ptr(exec.Timestamp)
			case "FAILED", "TIME_LIMIT_EXCEEDED":
				summary.LastFailureName = exec.SnapshotName
				summary.LastFailureTime = int64Ptr(exec.Timestamp)
				summary.LastFailureMsg = joinNonEmpty(exec.Info.Message, exec.Info.Cause)
			}
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return json.Marshal(out)
}

type smExecutionConfig struct {
	Schedule struct {
		Cron struct {
			Expression string `json:"expression"`
		} `json:"cron"`
	} `json:"schedule"`
}

type smExplain struct {
	ID       string `json:"_id"`
	Creation struct {
		LatestExecution struct {
			Status       string `json:"status"`
			SnapshotName string `json:"snapshot_name"`
			Timestamp    int64  `json:"timestamp"`
			Info         struct {
				Message string `json:"message"`
				Cause   string `json:"cause"`
			} `json:"info"`
		} `json:"latest_execution"`
	} `json:"creation"`
}

func splitIndices(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func joinNonEmpty(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ": ")
}

func millisToRFC3339(millis int64) string {
	if millis <= 0 {
		return ""
	}
	return time.UnixMilli(millis).UTC().Format(time.RFC3339)
}

func int64Ptr(v int64) *int64 { return &v }
