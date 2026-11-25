package config

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"cerebro/internal/domain"
)

// Hosts resolves the configured host list into a name->Host lookup while
// preserving the YAML declaration order for deterministic iteration.
type Hosts struct {
	byName map[string]domain.Host
	names  []string
}

// HostsDiff describes host changes between two snapshots. Entries contain only
// names, redacted URLs, and changed attribute labels, never credentials.
type HostsDiff struct {
	Added   []string
	Removed []string
	Changed []string
}

// IsEmpty reports whether the diff contains no changes.
func (d HostsDiff) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Changed) == 0
}

// NewHosts builds the name->Host lookup from the configured host list. It
// validates host URLs, normalizes types to lowercase, rejects duplicates, and
// preserves declaration order.
func NewHosts(cfg []HostConfig) (*Hosts, error) {
	byName := make(map[string]domain.Host, len(cfg))
	names := make([]string, 0, len(cfg))

	for i, h := range cfg {
		if h.Name != "" && strings.TrimSpace(h.Name) == "" {
			return nil, fmt.Errorf("hosts[%d]: host name must not be whitespace-only", i)
		}

		rawURL := strings.TrimSpace(h.Host)
		if rawURL == "" {
			return nil, fmt.Errorf("hosts[%d]: host URL must not be empty", i)
		}

		if err := validateHostURL(rawURL); err != nil {
			return nil, fmt.Errorf("hosts[%d]: %w", i, err)
		}

		flavor := strings.ToLower(strings.TrimSpace(h.Type))
		switch flavor {
		case "", "elasticsearch", "opensearch":
		default:
			return nil, fmt.Errorf("hosts[%d]: host %q: unknown type %q (want \"\", \"elasticsearch\" or \"opensearch\")", i, RedactHostURL(rawURL), h.Type)
		}

		name := strings.TrimSpace(h.Name)
		if name == "" {
			name = rawURL
		}
		if _, ok := byName[name]; ok {
			// name may be the fallback URL (with embedded userinfo); redact it
			// so credentials never leak into an error (CWE-532).
			display := name
			if name == rawURL {
				display = RedactHostURL(name)
			}
			return nil, fmt.Errorf("hosts[%d]: duplicate host name %q", i, display)
		}

		var auth *domain.ESAuth
		if h.Auth != nil {
			auth = &domain.ESAuth{Username: h.Auth.Username, Password: h.Auth.Password}
		}
		byName[name] = domain.Host{
			Name:             rawURL,
			Type:             flavor,
			Auth:             auth,
			HeadersWhitelist: slices.Clone(h.HeadersWhitelist),
		}
		names = append(names, name)
	}

	return &Hosts{byName: byName, names: names}, nil
}

func validateHostURL(rawURL string) error {
	if strings.Contains(rawURL, "://") {
		u, err := url.Parse(rawURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("invalid host URL %q: must be an http(s) URL", RedactHostURL(rawURL))
		}
		return nil
	}

	u, err := url.Parse("http://" + rawURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid host URL %q", RedactHostURL(rawURL))
	}
	return nil
}

// RedactHostURL removes userinfo credentials from a host URL before it is
// logged or embedded in an error. It delegates to domain.RedactHostURL so the
// handler and repository packages can redact URLs without importing config.
func RedactHostURL(raw string) string {
	return domain.RedactHostURL(raw)
}

// GetHostNames returns the configured host names in declaration order.
func (h *Hosts) GetHostNames() []string {
	if h == nil || len(h.names) == 0 {
		return []string{}
	}
	return slices.Clone(h.names)
}

// GetHost returns a defensive copy of the host for the given name.
func (h *Hosts) GetHost(name string) (domain.Host, bool) {
	if h == nil {
		return domain.Host{}, false
	}
	host, ok := h.byName[name]
	if !ok {
		return domain.Host{}, false
	}
	if host.Auth != nil {
		auth := *host.Auth
		host.Auth = &auth
	}
	host.HeadersWhitelist = slices.Clone(host.HeadersWhitelist)
	return host, true
}

// DiffHosts compares two snapshots deterministically in declaration order.
func DiffHosts(prev, next *Hosts) HostsDiff {
	var d HostsDiff
	if next != nil {
		for _, name := range next.names {
			nextHost := next.byName[name]
			if prev == nil {
				d.Added = append(d.Added, hostLabel(name, nextHost))
				continue
			}
			prevHost, ok := prev.byName[name]
			if !ok {
				d.Added = append(d.Added, hostLabel(name, nextHost))
				continue
			}
			changed := make([]string, 0, 4)
			if prevHost.Name != nextHost.Name {
				changed = append(changed, "url")
			}
			if prevHost.Type != nextHost.Type {
				changed = append(changed, "type")
			}
			if !equalESAuth(prevHost.Auth, nextHost.Auth) {
				changed = append(changed, "auth")
			}
			if !slices.Equal(prevHost.HeadersWhitelist, nextHost.HeadersWhitelist) {
				changed = append(changed, "headers-whitelist")
			}
			if len(changed) > 0 {
				d.Changed = append(d.Changed, fmt.Sprintf("%s (fields: %s)", displayHostName(name, nextHost), strings.Join(changed, ",")))
			}
		}
	}

	if prev != nil {
		for _, name := range prev.names {
			if next == nil {
				d.Removed = append(d.Removed, hostLabel(name, prev.byName[name]))
				continue
			}
			if _, ok := next.byName[name]; !ok {
				d.Removed = append(d.Removed, hostLabel(name, prev.byName[name]))
			}
		}
	}

	return d
}

func hostLabel(name string, host domain.Host) string {
	return fmt.Sprintf("%s (%s)", displayHostName(name, host), RedactHostURL(host.Name))
}

func displayHostName(name string, host domain.Host) string {
	if name == host.Name {
		return RedactHostURL(name)
	}
	return name
}

func equalESAuth(a, b *domain.ESAuth) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Username == b.Username && a.Password == b.Password
}
