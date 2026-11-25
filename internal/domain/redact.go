package domain

import (
	"net/url"
	"strings"
)

// RedactHostURL removes userinfo credentials from a host URL before it is
// logged or embedded in an error. URLs embedded in logs must never leak
// credentials (CWE-532).
func RedactHostURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	target := trimmed
	if !strings.Contains(trimmed, "://") {
		target = "http://" + trimmed
	}
	u, err := url.Parse(target)
	if err != nil {
		if strings.Contains(raw, "@") {
			return "<redacted-url>"
		}
		return raw
	}
	if u.User != nil {
		u.User = nil
		return u.String()
	}
	return raw
}
