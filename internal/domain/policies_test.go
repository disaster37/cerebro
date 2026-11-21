package domain

import (
	"encoding/json"
	"testing"
)

func TestNormalizeILM(t *testing.T) {
	policies := json.RawMessage(`{
		"policy1": {
			"version": 3,
			"modified_date": "2024-01-01T00:00:00.000Z",
			"policy": {"phases": {"hot": {}, "warm": {}, "delete": {}}}
		}
	}`)
	explain := json.RawMessage(`{
		"indices": {
			"idx1": {"index":"idx1","managed":true,"policy":"policy1","phase":"hot","action":"rollover","step":"ERROR","failed_step":"check-rollover-ready","step_info":{"type":"illegal_state_exception","reason":"boom"}},
			"idx2": {"index":"idx2","managed":true,"policy":"policy1","phase":"warm","action":"forcemerge","step":"complete"}
		}
	}`)
	status := json.RawMessage(`{"operation_mode":"RUNNING"}`)

	body, err := NormalizeIndexLifecycle("elasticsearch", policies, explain, status)
	if err != nil {
		t.Fatalf("NormalizeIndexLifecycle: %v", err)
	}
	var out IndexLifecyclePayload
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.FeatureName != "ILM" || out.OperationMode != "RUNNING" {
		t.Errorf("feature/mode = %s/%s, want ILM/RUNNING", out.FeatureName, out.OperationMode)
	}
	if len(out.Policies) != 1 || out.Policies[0].ManagedCount != 2 || out.Policies[0].FailedCount != 1 {
		t.Errorf("policies = %+v, want 1 policy with 2 managed / 1 failed", out.Policies)
	}
	if len(out.Policies[0].PhasesOrStates) != 3 {
		t.Errorf("phases = %v, want 3", out.Policies[0].PhasesOrStates)
	}
	var failed *ManagedIndexStatus
	for i := range out.ManagedIndices {
		if out.ManagedIndices[i].Index == "idx1" {
			failed = &out.ManagedIndices[i]
		}
	}
	if failed == nil || !failed.Failed || failed.Reason != "boom" || failed.FailedStep != "check-rollover-ready" {
		t.Errorf("failed index = %+v, want failed with reason boom", failed)
	}
}

func TestNormalizeSLM(t *testing.T) {
	policies := json.RawMessage(`{
		"nightly": {
			"version": 1,
			"policy": {"name":"nightly","schedule":"0 30 1 * * ?","repository":"repo","config":{"indices":["*"]}},
			"stats": {"snapshots_taken": 5, "snapshots_failed": 1},
			"last_success": {"snapshot_name":"snap-1","time": 1000},
			"last_failure": {"snapshot_name":"snap-2","time": 2000,"details":"s3 timeout"},
			"next_execution": "2024-01-02T01:30:00.000Z"
		}
	}`)
	body, err := NormalizeSnapshotLifecycle("elasticsearch", policies, json.RawMessage(`{"operation_mode":"RUNNING"}`))
	if err != nil {
		t.Fatalf("NormalizeSnapshotLifecycle: %v", err)
	}
	var out []SnapshotPolicySummary
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("policies = %d, want 1", len(out))
	}
	p := out[0]
	if p.ID != "nightly" || p.Repository != "repo" || p.Schedule != "0 30 1 * * ?" {
		t.Errorf("policy = %+v", p)
	}
	if p.LastFailureMsg != "s3 timeout" || p.LastFailureTime == nil || *p.LastFailureTime != 2000 {
		t.Errorf("last failure = %+v", p)
	}
	if p.SnapshotsTaken != 5 || p.SnapshotsFailed != 1 {
		t.Errorf("stats = %d/%d, want 5/1", p.SnapshotsTaken, p.SnapshotsFailed)
	}
	if p.NextExecution == nil {
		t.Error("next_execution_millis missing")
	}
}

func TestNormalizeISM(t *testing.T) {
	policies := json.RawMessage(`{
		"policies": [{
			"_id": "ism1",
			"policy": {"policy_id":"ism1","last_updated_time":1700000000000,"states":[{"name":"hot"},{"name":"delete"}]}
		}]
	}`)
	explain := json.RawMessage(`{
		"indices": {
			"idx1": {
				"index":"idx1","policy_id":"ism1",
				"state":{"name":"hot"},
				"action":{"name":"rollover","failed":true},
				"step":{"name":"check","step_status":"failed"},
				"retry_info":{"failed":true,"consumed_retries":2},
				"info":{"message":"failed","cause":"disk full"}
			},
			"unmanaged": {"index":"unmanaged","policy_id":""}
		}
	}`)
	status := json.RawMessage(`{"persistent":{"plugins.index_state_management.enabled":true}}`)

	body, err := NormalizeIndexLifecycle("opensearch", policies, explain, status)
	if err != nil {
		t.Fatalf("NormalizeIndexLifecycle: %v", err)
	}
	var out IndexLifecyclePayload
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.FeatureName != "ISM" || out.OperationMode != "RUNNING" {
		t.Errorf("feature/mode = %s/%s, want ISM/RUNNING", out.FeatureName, out.OperationMode)
	}
	if len(out.ManagedIndices) != 1 {
		t.Fatalf("managed = %d, want 1 (unmanaged ignored)", len(out.ManagedIndices))
	}
	m := out.ManagedIndices[0]
	if !m.Failed || m.Reason != "failed: disk full" || m.RetryCount != 2 {
		t.Errorf("managed = %+v", m)
	}
	if len(m.AvailableStates) != 2 {
		t.Errorf("available states = %v, want 2", m.AvailableStates)
	}
	if len(out.Policies) != 1 || out.Policies[0].FailedCount != 1 {
		t.Errorf("policies = %+v", out.Policies)
	}
}

func TestNormalizeSM(t *testing.T) {
	policies := json.RawMessage(`{
		"policies": [{
			"_id": "sm1",
			"policy": {
				"policy_id":"sm1","description":"nightly","enabled":true,
				"creation":{"schedule":{"cron":{"expression":"0 30 1 * * ?"}}},
				"snapshot_config":{"repository":"repo","indices":"idx1,idx2"}
			}
		}]
	}`)
	explain := json.RawMessage(`{
		"policies": [{
			"_id":"sm1",
			"creation":{"latest_execution":{"status":"FAILED","snapshot_name":"snap-1","timestamp":1234,"info":{"message":"failed","cause":"timeout"}}}
		}]
	}`)
	body, err := NormalizeSnapshotLifecycle("opensearch", policies, explain)
	if err != nil {
		t.Fatalf("NormalizeSnapshotLifecycle: %v", err)
	}
	var out []SnapshotPolicySummary
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("policies = %d, want 1", len(out))
	}
	p := out[0]
	if p.ID != "sm1" || p.Repository != "repo" || p.Schedule != "0 30 1 * * ?" {
		t.Errorf("policy = %+v", p)
	}
	if len(p.Indices) != 2 {
		t.Errorf("indices = %v, want 2", p.Indices)
	}
	if p.CurrentState != "FAILED" || p.LastFailureMsg != "failed: timeout" {
		t.Errorf("failure = %+v", p)
	}
}

func TestISMOperationModeBooleanAndString(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "boolean true",
			status: `{"persistent":{"plugins.index_state_management.enabled":true}}`,
			want:   "RUNNING",
		},
		{
			name:   "boolean false",
			status: `{"persistent":{"plugins.index_state_management.enabled":false}}`,
			want:   "STOPPED",
		},
		{
			name:   "string false",
			status: `{"persistent":{"plugins.index_state_management.enabled":"false"}}`,
			want:   "STOPPED",
		},
		{
			name:   "string true",
			status: `{"persistent":{"plugins.index_state_management.enabled":"true"}}`,
			want:   "RUNNING",
		},
		{
			name:   "persistent string false wins over defaults bool true",
			status: `{"persistent":{"plugins.index_state_management.enabled":"false"},"defaults":{"plugins.index_state_management.enabled":true}}`,
			want:   "STOPPED",
		},
		{
			name:   "string FALSE with surrounding whitespace",
			status: `{"persistent":{"plugins.index_state_management.enabled":" FALSE "}}`,
			want:   "STOPPED",
		},
		{
			name:   "string True mixed case",
			status: `{"persistent":{"plugins.index_state_management.enabled":"True"}}`,
			want:   "RUNNING",
		},
		{
			name:   "null persistent unmarshals to bool false",
			status: `{"persistent":{"plugins.index_state_management.enabled":null}}`,
			want:   "STOPPED",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ismOperationMode(json.RawMessage(tc.status)); got != tc.want {
				t.Errorf("ismOperationMode(%s) = %s, want %s", tc.status, got, tc.want)
			}
		})
	}
}

func TestISMExplainEntriesDirectWithTotalManagedIndices(t *testing.T) {
	explain := json.RawMessage(`{
		"pw-test": {"index.plugins.index_state_management.policy_id":"pw-ism","index":"pw-test","policy_id":"pw-ism","enabled":true},
		"total_managed_indices": 1
	}`)
	entries := ismExplainEntries(explain)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1 (total_managed_indices skipped)", len(entries))
	}
	if entries[0].Index != "pw-test" || entries[0].PolicyID != "pw-ism" {
		t.Errorf("entry = %+v, want index pw-test / policy pw-ism", entries[0])
	}
}

func TestISMExplainEntriesWrapped(t *testing.T) {
	explain := json.RawMessage(`{"indices":{"idx1":{"policy_id":"ism1"}}}`)
	entries := ismExplainEntries(explain)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Index != "idx1" || entries[0].PolicyID != "ism1" {
		t.Errorf("entry = %+v, want index idx1 / policy ism1", entries[0])
	}
}

func TestISMExplainEntriesAllNumbers(t *testing.T) {
	explain := json.RawMessage(`{"count":1,"total":2}`)
	entries := ismExplainEntries(explain)
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(entries))
	}
}
