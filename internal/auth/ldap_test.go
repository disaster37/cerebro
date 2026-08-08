package auth

import "testing"

func TestEscapeLDAPDN(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"alice", "alice"},
		{"alice.smith", "alice.smith"},
		{`a,b`, `a\,b`},
		{`a"b`, `a\"b`},
		{`a+b`, `a\+b`},
		{`a<b>c`, `a\<b\>c`},
		{`a;b`, `a\;b`},
		{`a=b`, `a\=b`},
		{`a\b`, `a\\b`},
		{"#admin", `\#admin`}, // leading '#'
		{"admin#", "admin#"},  // '#' only special when leading
		{" admin", `\ admin`}, // leading space
		{"admin ", `admin\ `}, // trailing space
		{"a b", "a b"},        // interior space untouched
		{"a\x00b", `a\00b`},   // NUL
		// The classic DN breakout: closing the RDN and injecting a new one.
		{"alice,ou=admins", `alice\,ou\=admins`},
	}
	for _, tc := range cases {
		if got := escapeLDAPDN(tc.in); got != tc.want {
			t.Errorf("escapeLDAPDN(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEscapeLDAPFilter(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"alice", "alice"},
		{"*", `\2a`},
		{"(", `\28`},
		{")", `\29`},
		{`\`, `\5c`},
		{"\x00", `\00`},
		// Filter breakout attempt: close the attr, inject an OR branch that
		// matches anything.
		{"alice)(|(userAttr=*", `alice\29\28|\28userAttr=\2a`},
	}
	for _, tc := range cases {
		if got := escapeLDAPFilter(tc.in); got != tc.want {
			t.Errorf("escapeLDAPFilter(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
