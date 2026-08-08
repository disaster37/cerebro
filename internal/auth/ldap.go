package auth

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"

	"cerebro/internal/config"
)

// ldapDialTimeout bounds connects to the LDAP server so an unreachable or
// slow directory cannot hang login requests indefinitely.
const ldapDialTimeout = 10 * time.Second

type LDAPAuth struct {
	cfg config.AuthSettings
}

func NewLDAPAuth(cfg config.AuthSettings) (*LDAPAuth, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("LDAP URL is required")
	}
	return &LDAPAuth{cfg: cfg}, nil
}

func (l *LDAPAuth) IsEnabled() bool { return true }

func (l *LDAPAuth) dial() (*ldap.Conn, error) {
	return ldap.DialURL(l.cfg.URL, ldap.DialWithDialer(&net.Dialer{Timeout: ldapDialTimeout}))
}

func (l *LDAPAuth) Auth(user, password string) (string, bool) {
	if l.cfg.GroupSearch.Group != "" {
		if !l.checkGroupMembership(user) {
			return "", false
		}
	}
	if !l.checkUserAuth(user, password) {
		return "", false
	}
	return user, true
}

func (l *LDAPAuth) checkUserAuth(user, password string) bool {
	// The username is interpolated into a DN template: escape it per RFC 4514
	// so it cannot break out of the RDN (LDAP DN injection).
	dn := fmt.Sprintf(l.cfg.UserTemplate, escapeLDAPDN(user), l.cfg.BaseDN)
	conn, err := l.dial()
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()
	if l.cfg.Method == "" {
		l.cfg.Method = "simple"
	}
	if err = conn.Bind(dn, password); err != nil {
		return false
	}
	return true
}

func (l *LDAPAuth) checkGroupMembership(user string) bool {
	conn, err := l.dial()
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()

	// Bind credentials are never logged.
	if err = conn.Bind(l.cfg.BindDN, l.cfg.BindPW); err != nil {
		return false
	}

	// The username is interpolated into a DN template and the result into a
	// search filter: escape per RFC 4514 (DN) and RFC 4515 (filter) so it
	// cannot alter either (LDAP filter injection).
	userDN := fmt.Sprintf(l.cfg.GroupSearch.UserAttrTemplate, escapeLDAPDN(user), l.cfg.BaseDN)
	filter := fmt.Sprintf("(&(%s=%s)(%s))", l.cfg.GroupSearch.UserAttr, escapeLDAPFilter(userDN), l.cfg.GroupSearch.Group)

	req := ldap.NewSearchRequest(
		l.cfg.GroupSearch.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"dn"},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return false
	}
	return len(res.Entries) > 0
}

// escapeLDAPDN escapes s for interpolation into a distinguished name
// (RFC 4514 §2.4): the characters ", +, , ; < > \ = are always escaped, a
// leading '#' and a leading/trailing space are escaped, and NUL becomes \00.
func escapeLDAPDN(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	last := len(s) - 1
	for i, r := range s {
		switch {
		case r == '#' && i == 0, r == ' ' && (i == 0 || i == last):
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '"' || r == '+' || r == ',' || r == ';' || r == '<' || r == '>' || r == '\\' || r == '=':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == 0:
			b.WriteString(`\00`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeLDAPFilter escapes s for interpolation into an LDAP search filter
// (RFC 4515 §3): \, *, (, ) and NUL are hex-escaped so the value cannot
// alter the filter structure.
func escapeLDAPFilter(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\5c`)
		case '*':
			b.WriteString(`\2a`)
		case '(':
			b.WriteString(`\28`)
		case ')':
			b.WriteString(`\29`)
		case 0:
			b.WriteString(`\00`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
