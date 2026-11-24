package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// DefaultSecret is the sample secret shipped in conf/application.yaml (also
// shipped by the original Scala cerebro). Deployments with auth enabled must
// replace it: it is public, and the session cookie HMAC is only as strong as
// this value.
const DefaultSecret = "ki:s:[[@=Ag?QI`W2jMwkY:eqvrJ]JqoJyi2axj3ZvOvOv^/KavOT4ViJSv?6YY4[N"

// MaxConfigBytes caps configuration files at 1 MiB (matching Kubernetes' 1 MiB
// ConfigMap limit) to prevent unbounded memory allocation on malformed inputs.
const MaxConfigBytes = 1 << 20

// Config holds the application configuration loaded from conf/application.yaml
// (or an explicit --config path) with environment variable overrides.
type Config struct {
	Secret          string
	BasePath        string
	DataPath        string
	RestHistorySize int
	ESGzip          bool
	Auth            AuthConfig
	Hosts           []HostConfig
}

// AuthConfig holds the authentication type and settings.
type AuthConfig struct {
	Type     string
	Settings AuthSettings
}

// AuthSettings holds basic and LDAP authentication settings.
type AuthSettings struct {
	Username, Password                string
	URL, BaseDN, Method, UserTemplate string
	BindDN, BindPW                    string
	GroupSearch                       GroupSearchConfig
}

// GroupSearchConfig holds LDAP group membership search settings.
type GroupSearchConfig struct {
	BaseDN, UserAttr, UserAttrTemplate, Group string
}

// HostConfig is a single configured ES host entry.
type HostConfig struct {
	Host string
	Name string
	Type string  `yaml:"type"` // "", "elasticsearch" or "opensearch"; optional, auto-detected when empty
	Auth *ESAuth `yaml:"auth"`
	// mapstructure tag required: viper.UnmarshalKey matches dash-less field
	// names case-insensitively, so "headers-whitelist" would not bind without it.
	HeadersWhitelist []string `yaml:"headers-whitelist" mapstructure:"headers-whitelist"`
}

// ESAuth mirrors domain.ESAuth for config parsing.
type ESAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// NewConfig loads conf/application.yaml (or the given path) with env overrides.
func NewConfig(path string) (*Config, error) {
	raw, err := readConfigFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := parseConfigBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// readConfigFile reads path in a single open/read (no TOCTOU window across
// atomic symlink swaps), capped at MaxConfigBytes.
func readConfigFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, MaxConfigBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxConfigBytes {
		return nil, fmt.Errorf("config %s exceeds max size (%d bytes)", path, MaxConfigBytes)
	}
	return data, nil
}

// parseConfigBytes parses raw YAML into a Config using an isolated viper
// instance so reloads never share mutable state with the running snapshot.
func parseConfigBytes(raw []byte) (*Config, error) {
	if len(bytes.TrimSpace(raw)) > 0 {
		var root map[string]any
		if err := yaml.Unmarshal(raw, &root); err != nil {
			return nil, fmt.Errorf("config root must be a YAML mapping: %w", err)
		}
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit env bindings (see plan §3.2).
	bindings := map[string]string{
		"secret":                                        "CEREBRO_SECRET",
		"auth.type":                                     "AUTH_TYPE",
		"auth.settings.username":                        "BASIC_AUTH_USER",
		"auth.settings.password":                        "BASIC_AUTH_PWD",
		"auth.settings.url":                             "LDAP_URL",
		"auth.settings.base-dn":                         "LDAP_BASE_DN",
		"auth.settings.method":                          "LDAP_METHOD",
		"auth.settings.user-template":                   "LDAP_USER_TEMPLATE",
		"auth.settings.bind-dn":                         "LDAP_BIND_DN",
		"auth.settings.bind-pw":                         "LDAP_BIND_PWD",
		"auth.settings.group-search.base-dn":            "LDAP_GROUP_BASE_DN",
		"auth.settings.group-search.user-attr":          "LDAP_USER_ATTR",
		"auth.settings.group-search.user-attr-template": "LDAP_USER_ATTR_TEMPLATE",
		"auth.settings.group-search.group":              "LDAP_GROUP",
	}
	for key, env := range bindings {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("bind env %s: %w", env, err)
		}
	}

	if err := v.ReadConfig(bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Secret:          v.GetString("secret"),
		BasePath:        v.GetString("basePath"),
		DataPath:        v.GetString("data.path"),
		RestHistorySize: v.GetInt("rest.history.size"),
		ESGzip:          v.GetBool("es.gzip"),
		Auth: AuthConfig{
			Type: v.GetString("auth.type"),
			Settings: AuthSettings{
				Username:     v.GetString("auth.settings.username"),
				Password:     v.GetString("auth.settings.password"),
				URL:          v.GetString("auth.settings.url"),
				BaseDN:       v.GetString("auth.settings.base-dn"),
				Method:       v.GetString("auth.settings.method"),
				UserTemplate: v.GetString("auth.settings.user-template"),
				BindDN:       v.GetString("auth.settings.bind-dn"),
				BindPW:       v.GetString("auth.settings.bind-pw"),
				GroupSearch: GroupSearchConfig{
					BaseDN:           v.GetString("auth.settings.group-search.base-dn"),
					UserAttr:         v.GetString("auth.settings.group-search.user-attr"),
					UserAttrTemplate: v.GetString("auth.settings.group-search.user-attr-template"),
					Group:            v.GetString("auth.settings.group-search.group"),
				},
			},
		},
	}

	if err := v.UnmarshalKey("hosts", &cfg.Hosts); err != nil {
		return nil, fmt.Errorf("unmarshal hosts: %w", err)
	}

	return cfg, nil
}

// restartOnlyDiff returns the names of restart-only fields that changed
// between prev and next. Values are never included.
func restartOnlyDiff(prev, next *Config) []string {
	if prev == nil || next == nil {
		return nil
	}

	fields := make([]string, 0, 7)
	if prev.Secret != next.Secret {
		fields = append(fields, "secret")
	}
	if prev.BasePath != next.BasePath {
		fields = append(fields, "basePath")
	}
	if prev.DataPath != next.DataPath {
		fields = append(fields, "data.path")
	}
	if prev.RestHistorySize != next.RestHistorySize {
		fields = append(fields, "rest.history.size")
	}
	if prev.ESGzip != next.ESGzip {
		fields = append(fields, "es.gzip")
	}
	if prev.Auth.Type != next.Auth.Type {
		fields = append(fields, "auth.type")
	}
	if prev.Auth.Settings != next.Auth.Settings {
		fields = append(fields, "auth.settings")
	}
	return fields
}
