package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// DefaultSecret is the sample secret shipped in conf/application.yaml (also
// shipped by the original Scala cerebro). Deployments with auth enabled must
// replace it: it is public, and the session cookie HMAC is only as strong as
// this value.
const DefaultSecret = "ki:s:[[@=Ag?QI`W2jMwkY:eqvrJ]JqoJyi2axj3ZvOvOv^/KavOT4ViJSv?6YY4[N"

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
	v := viper.New()
	v.SetConfigFile(path)
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

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
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
