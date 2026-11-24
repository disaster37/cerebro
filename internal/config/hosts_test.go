package config

import (
	"strings"
	"testing"
)

func TestNewHostsValidation(t *testing.T) {
	t.Run("valid preserves order and normalizes types", func(t *testing.T) {
		hosts, err := NewHosts([]HostConfig{
			{Host: "http://es1:9200", Name: "one"},
			{Host: "http://es2:9200", Name: "two", Type: "OpenSearch", Auth: &ESAuth{Username: "u", Password: "p"}, HeadersWhitelist: []string{"x-a"}},
			{Host: "https://es3:9200", Type: "elasticsearch"},
		})
		if err != nil {
			t.Fatalf("NewHosts: %v", err)
		}
		got := hosts.GetHostNames()
		want := []string{"one", "two", "https://es3:9200"}
		if len(got) != len(want) {
			t.Fatalf("names = %#v, want %#v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("names = %#v, want %#v", got, want)
			}
		}
		if h, _ := hosts.GetHost("two"); h.Type != "opensearch" {
			t.Errorf("type = %q, want opensearch", h.Type)
		}
	})

	cases := []struct {
		name    string
		cfg     []HostConfig
		wantErr string
	}{
		{"empty host", []HostConfig{{Host: ""}}, "must not be empty"},
		{"blank host", []HostConfig{{Host: "   "}}, "must not be empty"},
		{"whitespace name", []HostConfig{{Host: "http://es:9200", Name: "   "}}, "whitespace-only"},
		{"tab name", []HostConfig{{Host: "http://es:9200", Name: "\t"}}, "whitespace-only"},
		{"ftp scheme", []HostConfig{{Host: "ftp://es:9200"}}, "invalid host URL"},
		{"empty authority", []HostConfig{{Host: "http://"}}, "invalid host URL"},
		{"unknown type", []HostConfig{{Host: "http://es:9200", Type: "mysql"}}, "unknown type"},
		{"duplicate explicit", []HostConfig{{Host: "http://es1:9200", Name: "prod"}, {Host: "http://es2:9200", Name: "prod"}}, "duplicate host name"},
		{"duplicate fallback", []HostConfig{{Host: "http://es:9200", Name: "http://es:9200"}, {Host: "http://es:9200"}}, "duplicate host name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewHosts(tc.cfg)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestNewHostsEmpty(t *testing.T) {
	hosts, err := NewHosts(nil)
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	names := hosts.GetHostNames()
	if names == nil || len(names) != 0 {
		t.Fatalf("GetHostNames() = %#v, want empty non-nil slice", names)
	}
}

func TestNewHostsRedactsCredentialsInErrors(t *testing.T) {
	_, err := NewHosts([]HostConfig{{Host: "http://user:secret@es:9200", Type: "mysql"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "user:") {
		t.Fatalf("error leaked credentials: %q", err)
	}
	if !strings.Contains(err.Error(), "http://es:9200") {
		t.Fatalf("error = %q, want redacted URL", err)
	}
}

func TestHostsImmutability(t *testing.T) {
	hosts, err := NewHosts([]HostConfig{{
		Host:             "http://es:9200",
		Name:             "es",
		Auth:             &ESAuth{Username: "u", Password: "p"},
		HeadersWhitelist: []string{"x-a", "x-b"},
	}})
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}

	first, _ := hosts.GetHost("es")
	first.Auth.Username = "mutated"
	first.HeadersWhitelist[0] = "mutated"
	names := hosts.GetHostNames()
	names[0] = "mutated"

	second, _ := hosts.GetHost("es")
	if second.Auth.Username != "u" {
		t.Errorf("Auth.Username = %q, want u", second.Auth.Username)
	}
	if second.HeadersWhitelist[0] != "x-a" {
		t.Errorf("HeadersWhitelist[0] = %q, want x-a", second.HeadersWhitelist[0])
	}
	if got := hosts.GetHostNames()[0]; got != "es" {
		t.Errorf("GetHostNames()[0] = %q, want es", got)
	}
}

func TestRedactHostURL(t *testing.T) {
	cases := map[string]string{
		"http://es:9200":            "http://es:9200",
		"http://user:pass@es:9200":  "http://es:9200",
		"user:pass@es:9200":         "http://es:9200",
		"https://user@es:9200/path": "https://es:9200/path",
		"":                          "",
		"not a url with spaces":     "not a url with spaces",
		"http://bad host@es:9200":   "<redacted-url>",
	}
	for in, want := range cases {
		if got := RedactHostURL(in); got != want {
			t.Errorf("RedactHostURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDiffHosts(t *testing.T) {
	prev, err := NewHosts([]HostConfig{
		{Host: "http://es1:9200", Name: "keep"},
		{Host: "http://es2:9200", Name: "change", Type: "elasticsearch"},
		{Host: "http://user:secret@es3:9200", Name: "remove"},
	})
	if err != nil {
		t.Fatalf("prev: %v", err)
	}
	next, err := NewHosts([]HostConfig{
		{Host: "http://es1:9200", Name: "keep"},
		{Host: "http://es2-new:9200", Name: "change", Type: "opensearch"},
		{Host: "http://es4:9200", Name: "add"},
	})
	if err != nil {
		t.Fatalf("next: %v", err)
	}

	diff := DiffHosts(prev, next)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	if len(diff.Added) != 1 || !strings.Contains(diff.Added[0], "add") {
		t.Errorf("Added = %#v", diff.Added)
	}
	if len(diff.Removed) != 1 || !strings.Contains(diff.Removed[0], "remove") {
		t.Errorf("Removed = %#v", diff.Removed)
	}
	if len(diff.Changed) != 1 {
		t.Fatalf("Changed = %#v", diff.Changed)
	}
	for _, field := range []string{"url", "type"} {
		if !strings.Contains(diff.Changed[0], field) {
			t.Errorf("Changed[0] = %q, want field %q", diff.Changed[0], field)
		}
	}
	joined := strings.Join(append(append(diff.Added, diff.Removed...), diff.Changed...), " ")
	if strings.Contains(joined, "secret") || strings.Contains(joined, "user:") {
		t.Fatalf("diff leaked credentials: %q", joined)
	}

	if d := DiffHosts(prev, prev); !d.IsEmpty() {
		t.Errorf("self diff should be empty: %#v", d)
	}
}
