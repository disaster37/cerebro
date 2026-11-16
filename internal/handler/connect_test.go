package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
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

// connectFake records the target seen by the connect service.
type connectFake struct {
	got    domain.ElasticServer
	called bool
}

func (f *connectFake) ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, t domain.ElasticServer) (*domain.Response, error) {
	f.called = true
	f.got = t
	return &domain.Response{Status: 200, Body: json.RawMessage(`{"status":"green"}`)}, nil
}

func newConnectHandlers(t *testing.T, fake *connectFake, logger *logrus.Logger) *Handlers {
	t.Helper()
	hosts, err := config.NewHosts([]config.HostConfig{
		{Host: "http://es:9200", Name: "test"},
	})
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	return &Handlers{
		Hosts:   hosts,
		Connect: service.NewConnectService(fake),
		Log:     logger,
		Secret:  "test-secret",
	}
}

func performConnect(t *testing.T, h *Handlers, body string) *ut.ResponseRecorder {
	t.Helper()
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.POST("/connect", h.ConnectConnect)
	return ut.PerformRequest(engine, "POST", "/connect", &ut.Body{
		Body: strings.NewReader(body),
		Len:  len(body),
	})
}

func TestExtractBool(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"true", `{"trustCertificate":true}`, true},
		{"false", `{"trustCertificate":false}`, false},
		{"absent", `{}`, false},
		{"null", `{"trustCertificate":null}`, false},
		{"string", `{"trustCertificate":"true"}`, false},
		{"number", `{"trustCertificate":1}`, false},
		{"object", `{"trustCertificate":{}}`, false},
		{"array", `{"trustCertificate":[]}`, false},
		{"empty body", ``, false},
		{"malformed", `{"trustCertificate":`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractBool(json.RawMessage(tc.body), "trustCertificate"); got != tc.want {
				t.Errorf("extractBool(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestConnectPropagatesTrustCertificateAndLogsWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.TextFormatter{DisableColors: true})

	fake := &connectFake{}
	h := newConnectHandlers(t, fake, logger)

	w := performConnect(t, h, `{"host":"test","trustCertificate":true}`)
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if !fake.got.TrustCertificate {
		t.Error("target.TrustCertificate = false, want true")
	}
	logged := buf.String()
	if !strings.Contains(logged, "level=warning") {
		t.Errorf("log = %q, want level=warning", logged)
	}
	if !strings.Contains(logged, "trustCertificate=true") {
		t.Errorf("log = %q, want trustCertificate=true", logged)
	}

	buf.Reset()
	w = performConnect(t, h, `{"host":"test"}`)
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if fake.got.TrustCertificate {
		t.Error("target.TrustCertificate = true, want false when omitted")
	}
	if buf.Len() != 0 {
		t.Errorf("log = %q, want empty when trustCertificate is omitted", buf.String())
	}

	w = performConnect(t, h, `{"host":"test","trustCertificate":"true"}`)
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if fake.got.TrustCertificate {
		t.Error("target.TrustCertificate = true, want false for string \"true\"")
	}
}

func TestConnectMissingHostReturns400(t *testing.T) {
	fake := &connectFake{}
	h := newConnectHandlers(t, fake, logrus.New())

	w := performConnect(t, h, `{"trustCertificate":true}`)
	res := w.Result()
	if res.StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", res.StatusCode())
	}
	var env domain.Response
	if err := json.Unmarshal(res.Body(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Status != 400 {
		t.Errorf("envelope status = %d, want 400", env.Status)
	}
	var body map[string]string
	if err := json.Unmarshal(env.Body, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "missing host" {
		t.Errorf("error = %q, want missing host", body["error"])
	}
	if fake.called {
		t.Error("connect service called for missing host")
	}
}

func TestTrustCertificateDoesNotMutateHostsRegistry(t *testing.T) {
	fake := &connectFake{}
	h := newConnectHandlers(t, fake, logrus.New())

	before, ok := h.Hosts.GetHost("test")
	if !ok {
		t.Fatal("host 'test' missing")
	}

	w := performConnect(t, h, `{"host":"test","trustCertificate":true}`)
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if !fake.got.TrustCertificate {
		t.Error("first request target.TrustCertificate = false, want true")
	}

	w = performConnect(t, h, `{"host":"test"}`)
	if w.Result().StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", w.Result().StatusCode())
	}
	if fake.got.TrustCertificate {
		t.Error("second request target.TrustCertificate = true, want false")
	}

	after, ok := h.Hosts.GetHost("test")
	if !ok {
		t.Fatal("host 'test' missing after requests")
	}
	if !reflect.DeepEqual(before, after) {
		t.Errorf("registry host = %+v, want unchanged %+v", after, before)
	}
}
