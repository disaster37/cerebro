package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "application.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	return p
}

func TestNewHostsTypeNormalization(t *testing.T) {
	cfg := []HostConfig{
		{Host: "http://es:9200", Name: "es", Type: "opensearch"},
		{Host: "http://es2:9200", Name: "es2", Type: "OpenSearch"},
		{Host: "http://es3:9200", Name: "es3"}, // no type -> ""
	}
	hosts, err := NewHosts(cfg)
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	if h, ok := hosts.GetHost("es"); !ok || h.Type != "opensearch" {
		t.Errorf("es type = %q, want opensearch", h.Type)
	}
	if h, ok := hosts.GetHost("es2"); !ok || h.Type != "opensearch" {
		t.Errorf("es2 type = %q, want opensearch (normalized)", h.Type)
	}
	if h, ok := hosts.GetHost("es3"); !ok || h.Type != "" {
		t.Errorf("es3 type = %q, want empty", h.Type)
	}
}

func TestNewHostsUnknownTypeErrors(t *testing.T) {
	cfg := []HostConfig{
		{Host: "http://es:9200", Name: "es", Type: "mysql"},
	}
	_, err := NewHosts(cfg)
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
}

func TestNewHostsFallbackNameByURL(t *testing.T) {
	cfg := []HostConfig{
		{Host: "http://es:9200", Auth: &ESAuth{Username: "u", Password: "p"}},
	}
	hosts, err := NewHosts(cfg)
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	h, ok := hosts.GetHost("http://es:9200")
	if !ok {
		t.Fatal("expected host keyed by URL")
	}
	if h.Name != "http://es:9200" {
		t.Errorf("Name = %q, want URL", h.Name)
	}
	if h.Auth == nil || h.Auth.Username != "u" || h.Auth.Password != "p" {
		t.Errorf("Auth = %+v, want u/p", h.Auth)
	}
}

func TestNewConfigLoadsHosts(t *testing.T) {
	yaml := `
secret: "s"
basePath: "/"
hosts:
  - host: "http://es:9200"
    name: "es"
    type: "opensearch"
`
	p := writeYAML(t, yaml)
	cfg, err := NewConfig(p)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	if len(cfg.Hosts) != 1 {
		t.Fatalf("hosts = %d, want 1", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Type != "opensearch" {
		t.Errorf("type = %q, want opensearch", cfg.Hosts[0].Type)
	}
	hosts, err := NewHosts(cfg.Hosts)
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	h, ok := hosts.GetHost("es")
	if !ok {
		t.Fatal("expected host 'es'")
	}
	if h.Type != "opensearch" {
		t.Errorf("Host.Type = %q, want opensearch", h.Type)
	}
}

func TestNewConfigSecretEnvOverride(t *testing.T) {
	yaml := `
secret: "file-secret"
basePath: "/"
`
	p := writeYAML(t, yaml)
	t.Setenv("CEREBRO_SECRET", "env-secret")
	cfg, err := NewConfig(p)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	if cfg.Secret != "env-secret" {
		t.Errorf("Secret = %q, want env-secret (env must override file)", cfg.Secret)
	}
}

func TestNewConfigLoadsHeadersWhitelist(t *testing.T) {
	yaml := `
secret: "s"
hosts:
  - host: "http://es:9200"
    name: "es"
    headers-whitelist: ["x-proxy-user", "x-other"]
`
	p := writeYAML(t, yaml)
	cfg, err := NewConfig(p)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	if len(cfg.Hosts) != 1 {
		t.Fatalf("hosts = %d, want 1", len(cfg.Hosts))
	}
	got := cfg.Hosts[0].HeadersWhitelist
	if len(got) != 2 || got[0] != "x-proxy-user" || got[1] != "x-other" {
		t.Errorf("HeadersWhitelist = %#v, want [x-proxy-user x-other]", got)
	}
}
