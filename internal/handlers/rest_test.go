package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/sirupsen/logrus"

	"cerebro/internal/config"
	"cerebro/internal/domain"
)

// execFake records the ExecuteRequest arguments; unstubbed methods panic.
type execFake struct {
	domain.ElasticClient
	method string
	path   string
	data   json.RawMessage
}

func (f *execFake) ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, t domain.ElasticServer) (*domain.Response, error) {
	f.method = method
	f.path = path
	f.data = data
	return &domain.Response{Status: 200, Body: json.RawMessage(`{"ok":true}`)}, nil
}

// TestRestRequestUnwrapsStringData pins that the JSON-string `data` field sent
// by the REST console reaches ExecuteRequest as the RAW payload (Scala
// executeRequest unwrapped JsString bodies): the cluster must receive the
// text itself, not a quoted JSON string literal.
func TestRestRequestUnwrapsStringData(t *testing.T) {
	ndjson := `{"index":{"_index":"idx","_id":"1"}}` + "\n" + `{"a":1}` + "\n"

	fake := &execFake{}
	h := newRestHandlers(t, fake)

	reqBody, _ := json.Marshal(map[string]string{
		"host":   "test",
		"method": "POST",
		"path":   "_bulk",
		"data":   ndjson, // serialized as a JSON string by the frontend
	})
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.POST("/rest/request", h.RestRequest)
	w := ut.PerformRequest(engine, "POST", "/rest/request", &ut.Body{
		Body: strings.NewReader(string(reqBody)),
		Len:  len(reqBody),
	})
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}

	if fake.method != "POST" || fake.path != "_bulk" {
		t.Errorf("ExecuteRequest(%q, %q), want (POST, _bulk)", fake.method, fake.path)
	}
	if string(fake.data) != ndjson {
		t.Errorf("data = %q, want raw ndjson %q (unwrapped, no quotes/escapes)", fake.data, ndjson)
	}
}

// TestRestRequestKeepsNonStringData pins that non-string data (a real JSON
// object) passes through unchanged.
func TestRestRequestKeepsNonStringData(t *testing.T) {
	fake := &execFake{}
	h := newRestHandlers(t, fake)

	reqBody := `{"host":"test","method":"GET","path":"/_search","data":{"query":{"match_all":{}}}}`
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.POST("/rest/request", h.RestRequest)
	w := ut.PerformRequest(engine, "POST", "/rest/request", &ut.Body{
		Body: strings.NewReader(reqBody),
		Len:  len(reqBody),
	})
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if string(fake.data) != `{"query":{"match_all":{}}}` {
		t.Errorf("data = %s, want the JSON object unchanged", fake.data)
	}
}

func newRestHandlers(t *testing.T, fake *execFake) *Handlers {
	t.Helper()
	hosts, err := config.NewHosts([]config.HostConfig{
		{Host: "http://es:9200", Name: "test"},
	})
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	return &Handlers{
		Hosts:   hosts,
		Elastic: fake,
		Log:     logrus.New(),
		Secret:  "test-secret",
	}
}

// TestRestHistoryWithoutDAO pins that /rest/history degrades to an empty list
// (not a panic) when data.path is unset and History is nil.
func TestRestHistoryWithoutDAO(t *testing.T) {
	fake := &execFake{}
	h := newRestHandlers(t, fake) // History left nil

	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.POST("/rest/history", h.RestHistory)
	w := ut.PerformRequest(engine, "POST", "/rest/history", &ut.Body{
		Body: strings.NewReader(`{"host":"test"}`),
		Len:  len(`{"host":"test"}`),
	})
	res := w.Result()
	if res.StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", res.StatusCode())
	}
	var env domain.Response
	if err := json.Unmarshal(res.Body(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Status != 200 || string(env.Body) != "[]" {
		t.Errorf("envelope = {%d, %s}, want {200, []}", env.Status, env.Body)
	}
}

// captureFake records the target seen by ClusterHealth; unstubbed methods panic.
type captureFake struct {
	domain.ElasticClient
	got domain.ElasticServer
}

func (f *captureFake) ClusterHealth(ctx context.Context, t domain.ElasticServer) (*domain.Response, error) {
	f.got = t
	return &domain.Response{Status: 200, Body: json.RawMessage(`{"status":"green"}`)}, nil
}

func (f *captureFake) BackendInfo(ctx context.Context, t domain.ElasticServer) (domain.BackendInfo, error) {
	return domain.BackendInfo{Type: "elasticsearch"}, nil
}

// TestPerRequestCredentialsDoNotMutateRegistry pins that per-request
// username/password overrides do NOT mutate the shared Hosts registry entry
// (they used to alias targetHost.Auth), which leaked credentials across
// requests and raced under concurrency.
func TestPerRequestCredentialsDoNotMutateRegistry(t *testing.T) {
	hosts, err := config.NewHosts([]config.HostConfig{
		{Host: "http://es:9200", Name: "test", Auth: &config.ESAuth{Username: "cfg_u", Password: "cfg_p"}},
	})
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	fake := &captureFake{}
	h := &Handlers{
		Hosts:   hosts,
		Elastic: fake,
		Log:     logrus.New(),
		Secret:  "test-secret",
	}

	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.POST("/navbar", h.NavbarIndex)

	// Request 1: per-request credentials override the configured ones.
	w := ut.PerformRequest(engine, "POST", "/navbar", &ut.Body{
		Body: strings.NewReader(`{"host":"test","username":"req_u","password":"req_p"}`),
		Len:  len(`{"host":"test","username":"req_u","password":"req_p"}`),
	})
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if fake.got.Host.Auth == nil || fake.got.Host.Auth.Username != "req_u" || fake.got.Host.Auth.Password != "req_p" {
		t.Errorf("request auth = %+v, want req_u/req_p", fake.got.Host.Auth)
	}

	// The registry entry must still hold the configured credentials.
	reg, ok := hosts.GetHost("test")
	if !ok {
		t.Fatal("host 'test' missing")
	}
	if reg.Auth == nil || reg.Auth.Username != "cfg_u" || reg.Auth.Password != "cfg_p" {
		t.Errorf("registry auth = %+v, want cfg_u/cfg_p (must not be mutated)", reg.Auth)
	}

	// Request 2: without per-request credentials the configured ones apply.
	w = ut.PerformRequest(engine, "POST", "/navbar", &ut.Body{
		Body: strings.NewReader(`{"host":"test"}`),
		Len:  len(`{"host":"test"}`),
	})
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if fake.got.Host.Auth == nil || fake.got.Host.Auth.Username != "cfg_u" || fake.got.Host.Auth.Password != "cfg_p" {
		t.Errorf("second request auth = %+v, want cfg_u/cfg_p (no leak from request 1)", fake.got.Host.Auth)
	}
}
