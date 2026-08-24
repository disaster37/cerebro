package domain

// AuthService authenticates a user. Implemented by internal/auth.
type AuthService interface {
	Auth(username, password string) (string, bool)
	IsEnabled() bool
}
