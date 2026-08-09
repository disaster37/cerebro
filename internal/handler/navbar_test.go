package handler

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
	"cerebro/internal/service"
)

// fakeElastic satisfies the full ElasticClient interface without 50 stubs:
// the embedded nil interface panics if any unstubbed method is called, which
// is correct for a test double.
type fakeElastic struct {
	domain.ElasticClient
	health *domain.Response
	info   domain.BackendInfo
}

func (f *fakeElastic) ClusterHealth(ctx context.Context, t domain.ElasticServer) (*domain.Response, error) {
	return f.health, nil
}

func (f *fakeElastic) BackendInfo(ctx context.Context, t domain.ElasticServer) (domain.BackendInfo, error) {
	return f.info, nil
}

func newNavbarHandlers(t *testing.T, fake *fakeElastic) *Handlers {
	t.Helper()
	hosts, err := config.NewHosts([]config.HostConfig{
		{Host: "http://es:9200", Name: "test"},
	})
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	return &Handlers{
		Hosts:   hosts,
		Navbar:  service.NewNavbarService(fake),
		Log:     logrus.New(),
		Secret:  "test-secret",
	}
}

func performNavbar(t *testing.T, h *Handlers, body string) *ut.ResponseRecorder {
	t.Helper()
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.POST("/navbar", h.NavbarIndex)
	w := ut.PerformRequest(engine, "POST", "/navbar", &ut.Body{
		Body: strings.NewReader(body),
		Len:  len(body),
	})
	return w
}

func TestNavbarInjectsBackendFields(t *testing.T) {
	healthBody, _ := json.Marshal(map[string]string{
		"cluster_name": "c",
		"status":       "green",
	})
	fake := &fakeElastic{
		health: &domain.Response{Status: 200, Body: healthBody},
		info:   domain.BackendInfo{Type: "opensearch", Version: "3.4.0"},
	}
	h := newNavbarHandlers(t, fake)

	w := performNavbar(t, h, `{"host":"test"}`)
	res := w.Result()
	if res.StatusCode() != 200 {
		t.Errorf("HTTP status = %d, want 200", res.StatusCode())
	}

	var env domain.Response
	if err := json.Unmarshal(res.Body(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\nbody: %s", err, res.Body())
	}
	if env.Status != 200 {
		t.Errorf("envelope status = %d, want 200", env.Status)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(env.Body, &m); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if string(m["cluster_name"]) != `"c"` {
		t.Errorf("cluster_name = %s, want \"c\"", m["cluster_name"])
	}
	if string(m["backendType"]) != `"opensearch"` {
		t.Errorf("backendType = %s, want \"opensearch\"", m["backendType"])
	}
	if string(m["backendVersion"]) != `"3.4.0"` {
		t.Errorf("backendVersion = %s, want \"3.4.0\"", m["backendVersion"])
	}
	if _, ok := m["username"]; ok {
		t.Errorf("username should be absent when no user, got %s", m["username"])
	}
}

func TestNavbarKeepsExistingFields(t *testing.T) {
	healthBody, _ := json.Marshal(map[string]string{
		"cluster_name": "c",
		"status":       "yellow",
	})
	fake := &fakeElastic{
		health: &domain.Response{Status: 200, Body: healthBody},
		info:   domain.BackendInfo{Type: "elasticsearch", Version: "9.4.3"},
	}
	h := newNavbarHandlers(t, fake)

	w := performNavbar(t, h, `{"host":"test"}`)
	res := w.Result()

	var env domain.Response
	if err := json.Unmarshal(res.Body(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(env.Body, &m); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if string(m["cluster_name"]) != `"c"` {
		t.Errorf("cluster_name = %s, want \"c\" (preserved)", m["cluster_name"])
	}
	if string(m["status"]) != `"yellow"` {
		t.Errorf("status = %s, want \"yellow\" (preserved)", m["status"])
	}
	if string(m["backendType"]) != `"elasticsearch"` {
		t.Errorf("backendType = %s, want \"elasticsearch\"", m["backendType"])
	}
}
