package domain

import (
	"encoding/json"
	"sort"
	"strings"
)

// AllowedDynamicPrefixes covers wildcard/namespace settings that do not
// enumerate every possible suffix key inside `defaults` in Elasticsearch and
// OpenSearch.
var AllowedDynamicPrefixes = []string{
	"logger.",
	"cluster.routing.allocation.include.",
	"cluster.routing.allocation.exclude.",
	"cluster.routing.allocation.require.",
	"cluster.routing.allocation.awareness.force.",
}

// ValidateClusterSettingsPayload parses a {"persistent":{...},"transient":{...}}
// payload and verifies every setting key exists in knownClusterSettings (the
// union of defaults, persistent, and transient from GetClusterSettingsFlat) or
// matches AllowedDynamicPrefixes. Returns the sorted list of unknown keys.
func ValidateClusterSettingsPayload(payload, flatClusterSettings json.RawMessage) ([]string, error) {
	var settings struct {
		Persistent map[string]json.RawMessage `json:"persistent"`
		Transient  map[string]json.RawMessage `json:"transient"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &settings); err != nil {
			return nil, err
		}
	}

	var flat struct {
		Defaults   map[string]json.RawMessage `json:"defaults"`
		Persistent map[string]json.RawMessage `json:"persistent"`
		Transient  map[string]json.RawMessage `json:"transient"`
	}
	if len(flatClusterSettings) > 0 {
		if err := json.Unmarshal(flatClusterSettings, &flat); err != nil {
			return nil, err
		}
	}

	known := map[string]bool{}
	for _, m := range []map[string]json.RawMessage{flat.Defaults, flat.Persistent, flat.Transient} {
		collectSettingKeys(m, "", known)
	}

	unknown := map[string]bool{}
	for _, m := range []map[string]json.RawMessage{settings.Persistent, settings.Transient} {
		for key := range m {
			key = strings.TrimSpace(key)
			if key == "" || known[key] || hasAllowedPrefix(key) {
				continue
			}
			unknown[key] = true
		}
	}

	out := make([]string, 0, len(unknown))
	for key := range unknown {
		out = append(out, key)
	}
	sort.Strings(out)
	return out, nil
}

func hasAllowedPrefix(key string) bool {
	for _, prefix := range AllowedDynamicPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// collectSettingKeys flattens nested setting maps into dotted keys so both
// flat_settings=true responses and nested defaults are recognized.
func collectSettingKeys(m map[string]json.RawMessage, prefix string, out map[string]bool) {
	for key, raw := range m {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		out[full] = true
		var nested map[string]json.RawMessage
		if len(raw) > 0 && raw[0] == '{' && json.Unmarshal(raw, &nested) == nil {
			collectSettingKeys(nested, full, out)
		}
	}
}
