package auth

import (
	"cerebro/internal/config"
	"cerebro/internal/domain"
)

type NoneAuth struct{}

func (n *NoneAuth) Auth(_, _ string) (string, bool) { return "", false }
func (n *NoneAuth) IsEnabled() bool                 { return false }

func NewAuthService(cfg config.AuthConfig) (domain.AuthService, error) {
	switch cfg.Type {
	case "basic":
		return NewBasicAuth(cfg.Settings), nil
	case "ldap":
		return NewLDAPAuth(cfg.Settings)
	default:
		return &NoneAuth{}, nil
	}
}
