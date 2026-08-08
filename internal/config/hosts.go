package config

import (
	"fmt"
	"strings"

	"cerebro/internal/models"
)

// Hosts resolves the configured host list into a name->Host lookup.
type Hosts struct {
	byName map[string]models.Host
}

// NewHosts builds the name->Host lookup from the configured host list. It
// errors on an unknown host type (anything other than "", "elasticsearch",
// "opensearch", case-insensitive). Types are normalized to lowercase.
func NewHosts(cfg []HostConfig) (*Hosts, error) {
	byName := make(map[string]models.Host, len(cfg))
	for _, h := range cfg {
		flavor := strings.ToLower(h.Type)
		switch flavor {
		case "", "elasticsearch", "opensearch":
		default:
			return nil, fmt.Errorf("host %q: unknown type %q (want \"\", \"elasticsearch\" or \"opensearch\")", h.Host, h.Type)
		}
		name := h.Name
		if name == "" {
			name = h.Host
		}
		var auth *models.ESAuth
		if h.Auth != nil {
			auth = &models.ESAuth{Username: h.Auth.Username, Password: h.Auth.Password}
		}
		byName[name] = models.Host{
			Name:             h.Host,
			Type:             flavor,
			Auth:             auth,
			HeadersWhitelist: h.HeadersWhitelist,
		}
	}
	return &Hosts{byName: byName}, nil
}

// GetHostNames returns the configured host names.
func (h *Hosts) GetHostNames() []string {
	names := make([]string, 0, len(h.byName))
	for name := range h.byName {
		names = append(names, name)
	}
	return names
}

// GetHost returns the host for the given name.
func (h *Hosts) GetHost(name string) (models.Host, bool) {
	host, ok := h.byName[name]
	return host, ok
}
