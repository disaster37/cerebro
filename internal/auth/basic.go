package auth

import (
	"crypto/subtle"

	"cerebro/internal/config"
)

type BasicAuth struct {
	username string
	password string
}

func NewBasicAuth(s config.AuthSettings) *BasicAuth {
	return &BasicAuth{
		username: s.Username,
		password: s.Password,
	}
}

func (b *BasicAuth) Auth(username, password string) (string, bool) {
	if subtle.ConstantTimeCompare([]byte(username), []byte(b.username)) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(b.password)) == 1 {
		return username, true
	}
	return "", false
}

func (b *BasicAuth) IsEnabled() bool { return true }
