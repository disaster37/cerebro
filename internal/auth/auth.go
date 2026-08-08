package auth

import "cerebro/internal/config"

type AuthService interface {
	Auth(username, password string) (string, bool)
	IsEnabled() bool
}

type NoneAuth struct{}

func (n *NoneAuth) Auth(_, _ string) (string, bool) { return "", false }
func (n *NoneAuth) IsEnabled() bool                 { return false }

func NewAuthService(cfg config.AuthConfig) (AuthService, error) {
	switch cfg.Type {
	case "basic":
		return NewBasicAuth(cfg.Settings), nil
	case "ldap":
		return NewLDAPAuth(cfg.Settings)
	default:
		return &NoneAuth{}, nil
	}
}
