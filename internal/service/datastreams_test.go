package service

import (
	"context"
	"encoding/json"
	"testing"

	"cerebro/internal/domain"
)

type fakeDataStreamAPI struct {
	domain.DataStreamAPI
	listResp     *domain.Response
	createResp   *domain.Response
	deleteResp   *domain.Response
	rolloverResp *domain.Response
	createdName  string
	deletedName  string
	rolledName   string
}

func (f *fakeDataStreamAPI) GetDataStreams(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.listResp, nil
}

func (f *fakeDataStreamAPI) CreateDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	f.createdName = name
	return f.createResp, nil
}

func (f *fakeDataStreamAPI) DeleteDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	f.deletedName = name
	return f.deleteResp, nil
}

func (f *fakeDataStreamAPI) RolloverDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	f.rolledName = name
	return f.rolloverResp, nil
}

func TestDataStreamsServiceList(t *testing.T) {
	fake := &fakeDataStreamAPI{listResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"data_streams":[
		{"name":"logs-app","timestamp_field":{"name":"@timestamp"},"indices":[{"index_name":".ds-logs-app-000001"}],"generation":1,"status":"GREEN","template":"t"}
	]}`)}}
	svc := NewDataStreamsService(fake, nil)
	resp, err := svc.List(context.Background(), domain.CerebroRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	var streams []domain.DataStreamSummary
	if err := json.Unmarshal(resp.Body, &streams); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(streams) != 1 || streams[0].Name != "logs-app" || streams[0].WriteIndex != ".ds-logs-app-000001" {
		t.Errorf("streams = %+v", streams)
	}
}

func TestDataStreamsServiceCreateWithTemplate(t *testing.T) {
	ds := &fakeDataStreamAPI{createResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"acknowledged":true}`)}}
	tmpl := &fakeTemplateAPI{createResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"acknowledged":true}`)}}
	svc := NewDataStreamsService(ds, tmpl)

	req := domain.CerebroRequest{Body: json.RawMessage(`{"name":"logs-app","createTemplate":true}`)}
	resp, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	if tmpl.createdName != "logs-app-ds-template" {
		t.Errorf("template name = %q, want logs-app-ds-template", tmpl.createdName)
	}
	var body map[string]any
	if err := json.Unmarshal(tmpl.createdBody, &body); err != nil {
		t.Fatalf("unmarshal template: %v", err)
	}
	if _, ok := body["data_stream"]; !ok {
		t.Errorf("template body missing data_stream: %s", tmpl.createdBody)
	}
	if body["priority"] != float64(200) {
		t.Errorf("priority = %v, want 200", body["priority"])
	}
	if ds.createdName != "logs-app" {
		t.Errorf("created data stream = %q, want logs-app", ds.createdName)
	}
}

func TestDataStreamsServiceCreateValidation(t *testing.T) {
	ds := &fakeDataStreamAPI{}
	svc := NewDataStreamsService(ds, nil)
	resp, err := svc.Create(context.Background(), domain.CerebroRequest{Body: json.RawMessage(`{"name":"Logs-App"}`)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.Status != 400 {
		t.Errorf("status = %d, want 400", resp.Status)
	}
	if ds.createdName != "" {
		t.Errorf("CreateDataStream called for invalid name")
	}
}

func TestDataStreamsServiceCreateTemplateFailurePassthrough(t *testing.T) {
	ds := &fakeDataStreamAPI{}
	tmpl := &fakeTemplateAPI{createResp: &domain.Response{Status: 400, Body: json.RawMessage(`{"error":"template conflict"}`)}}
	svc := NewDataStreamsService(ds, tmpl)
	resp, err := svc.Create(context.Background(), domain.CerebroRequest{Body: json.RawMessage(`{"name":"logs-app","createTemplate":true}`)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.Status != 400 {
		t.Errorf("status = %d, want 400 passthrough", resp.Status)
	}
	if ds.createdName != "" {
		t.Errorf("CreateDataStream called after template failure")
	}
}

func TestDataStreamsServiceDeleteAndRollover(t *testing.T) {
	ds := &fakeDataStreamAPI{
		deleteResp:   &domain.Response{Status: 200, Body: json.RawMessage(`{"acknowledged":true}`)},
		rolloverResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"acknowledged":true}`)},
	}
	svc := NewDataStreamsService(ds, nil)
	if resp, _ := svc.Delete(context.Background(), domain.CerebroRequest{Body: json.RawMessage(`{"name":"logs-app"}`)}); resp.Status != 200 {
		t.Errorf("delete status = %d", resp.Status)
	}
	if resp, _ := svc.Rollover(context.Background(), domain.CerebroRequest{Body: json.RawMessage(`{"name":"logs-app"}`)}); resp.Status != 200 {
		t.Errorf("rollover status = %d", resp.Status)
	}
	if ds.deletedName != "logs-app" || ds.rolledName != "logs-app" {
		t.Errorf("delete/rollover names = %q/%q", ds.deletedName, ds.rolledName)
	}
}
