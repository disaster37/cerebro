package domain

import (
	"encoding/json"
	"testing"
)

func TestParseDataStreamsElasticsearch(t *testing.T) {
	raw := json.RawMessage(`{"data_streams":[{
		"name":"logs-app",
		"timestamp_field":{"name":"@timestamp"},
		"indices":[
			{"index_name":".ds-logs-app-000001","index_uuid":"a"},
			{"index_name":".ds-logs-app-000002","index_uuid":"b"}
		],
		"generation":2,
		"status":"GREEN",
		"template":"logs-app-template"
	}]}`)
	streams, backing, err := ParseDataStreams(raw)
	if err != nil {
		t.Fatalf("ParseDataStreams: %v", err)
	}
	if len(streams) != 1 {
		t.Fatalf("streams = %d, want 1", len(streams))
	}
	ds := streams[0]
	if ds.Name != "logs-app" || ds.Status != "green" || ds.Generation != 2 || ds.Template != "logs-app-template" {
		t.Errorf("stream = %+v", ds)
	}
	if ds.TimestampField != "@timestamp" || ds.WriteIndex != ".ds-logs-app-000002" {
		t.Errorf("timestamp/write index = %q/%q", ds.TimestampField, ds.WriteIndex)
	}
	if len(ds.BackingIndices) != 2 {
		t.Errorf("backing indices = %v", ds.BackingIndices)
	}
	if meta := backing[".ds-logs-app-000001"]; meta.DataStream != "logs-app" || meta.IsWriteIndex || meta.HiddenOrSystem {
		t.Errorf("first backing meta = %+v", meta)
	}
	if meta := backing[".ds-logs-app-000002"]; !meta.IsWriteIndex {
		t.Errorf("last backing meta = %+v, want write index", meta)
	}
}

func TestParseDataStreamsOpenSearchHidden(t *testing.T) {
	raw := json.RawMessage(`{"data_streams":[{
		"name":".hidden-stream",
		"timestamp_field":{"name":"@timestamp"},
		"indices":[{"index_name":".ds-.hidden-stream-000001"}],
		"generation":1,
		"status":"YELLOW",
		"template":"hidden-template",
		"hidden":true
	}]}`)
	streams, backing, err := ParseDataStreams(raw)
	if err != nil {
		t.Fatalf("ParseDataStreams: %v", err)
	}
	if len(streams) != 1 || streams[0].Status != "yellow" || !streams[0].Hidden {
		t.Errorf("streams = %+v", streams)
	}
	if meta := backing[".ds-.hidden-stream-000001"]; !meta.HiddenOrSystem {
		t.Errorf("hidden backing meta = %+v, want HiddenOrSystem", meta)
	}
}

func TestParseDataStreamsEmptyAndFallback(t *testing.T) {
	streams, backing, err := ParseDataStreams(nil)
	if err != nil {
		t.Fatalf("ParseDataStreams(nil): %v", err)
	}
	if streams == nil || len(streams) != 0 || backing == nil {
		t.Errorf("empty parse = %v / %v, want non-nil empty", streams, backing)
	}

	name, ok := DataStreamNameFromBackingIndex(".ds-logs-app-000012")
	if !ok || name != "logs-app" {
		t.Errorf("fallback = %q/%v, want logs-app/true", name, ok)
	}
	if _, ok := DataStreamNameFromBackingIndex("logs-app"); ok {
		t.Error("non-backing index should not resolve")
	}
}

func TestValidateDataStreamName(t *testing.T) {
	valid := []string{"logs-app", "metrics-2024", "a"}
	for _, name := range valid {
		if err := ValidateDataStreamName(name); err != nil {
			t.Errorf("ValidateDataStreamName(%q) = %v, want nil", name, err)
		}
	}
	invalid := []string{"", "Logs-App", "-logs", "_logs", "+logs", ".ds-logs", "logs app", "logs*", "logs/app", "logs,app"}
	for _, name := range invalid {
		if err := ValidateDataStreamName(name); err == nil {
			t.Errorf("ValidateDataStreamName(%q) = nil, want error", name)
		}
	}
}
