package elastic

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Flavor identifies a backend cluster flavor.
type Flavor string

const (
	FlavorElasticsearch Flavor = "elasticsearch"
	FlavorOpenSearch    Flavor = "opensearch"
)

// flavorTTL is the cache TTL for detected flavor info.
const flavorTTL = 10 * time.Minute

// FlavorInfo is a detected (or configured-fallback) backend identity.
type FlavorInfo struct {
	Flavor  Flavor
	Version string
}

// NormalizeFlavor maps a configured type string to a Flavor ("" -> "").
// Comparison is case-insensitive; unknown values map to "".
func NormalizeFlavor(s string) Flavor {
	switch strings.ToLower(s) {
	case "elasticsearch":
		return FlavorElasticsearch
	case "opensearch":
		return FlavorOpenSearch
	default:
		return ""
	}
}

// parseFlavor sniffs a GET / response body:
//
//	{"version":{"distribution":"opensearch","number":"3.4.0",...}} -> {opensearch, "3.4.0"}
//	{"version":{"number":"9.4.3",...},"tagline":...}               -> {elasticsearch, "9.4.3"}
//
// Distribution comparison is case-insensitive. Error if body is not a JSON
// object with a parseable version block.
func parseFlavor(body []byte) (FlavorInfo, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return FlavorInfo{}, fmt.Errorf("parse flavor: invalid JSON: %w", err)
	}
	versionRaw, ok := root["version"]
	if !ok {
		return FlavorInfo{}, fmt.Errorf("parse flavor: missing version block")
	}
	var version map[string]json.RawMessage
	if err := json.Unmarshal(versionRaw, &version); err != nil {
		return FlavorInfo{}, fmt.Errorf("parse flavor: invalid version block: %w", err)
	}
	info := FlavorInfo{}
	if distRaw, ok := version["distribution"]; ok {
		var dist string
		if json.Unmarshal(distRaw, &dist) == nil && strings.ToLower(dist) == "opensearch" {
			info.Flavor = FlavorOpenSearch
		}
	}
	if info.Flavor == "" {
		info.Flavor = FlavorElasticsearch
	}
	if numRaw, ok := version["number"]; ok {
		var num string
		if json.Unmarshal(numRaw, &num) == nil {
			info.Version = num
		}
	}
	if info.Version == "" {
		return FlavorInfo{}, fmt.Errorf("parse flavor: missing version.number")
	}
	return info, nil
}

// flavorEntry is a single cache entry.
type flavorEntry struct {
	info FlavorInfo
	at   time.Time
}

// flavorCache is a thread-safe TTL cache of FlavorInfo keyed by host URL.
type flavorCache struct {
	mu      sync.RWMutex
	entries map[string]flavorEntry
	ttl     time.Duration
}

func newFlavorCache(ttl time.Duration) *flavorCache {
	return &flavorCache{
		entries: make(map[string]flavorEntry),
		ttl:     ttl,
	}
}

// get returns the cached FlavorInfo for url and true when present and fresh;
// false when missing or expired.
func (c *flavorCache) get(url string) (FlavorInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[url]
	if !ok {
		return FlavorInfo{}, false
	}
	if time.Since(e.at) > c.ttl {
		return FlavorInfo{}, false
	}
	return e.info, true
}

// set stores info for url with the current timestamp.
func (c *flavorCache) set(url string, info FlavorInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[url] = flavorEntry{info: info, at: time.Now()}
}
