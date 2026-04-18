package api

import (
	"context"
	"errors"
	"strings"

	"github.com/cannonball10/foundation/connectors/authentication"
)

// Authenticator resolves an incoming request to a stable user ID. It
// is kept narrow so tests can pass a fixture without wiring Clerk.
type Authenticator interface {
	// Authenticate returns the user ID for the given bearer token.
	// Implementations should return ErrUnauthenticated when the token
	// is missing, malformed, or invalid.
	Authenticate(ctx context.Context, token string) (userID string, err error)
}

// ErrUnauthenticated is returned by Authenticator when no valid
// identity can be derived from the request.
var ErrUnauthenticated = errors.New("unauthenticated")

// NopAuth accepts any non-empty bearer token and treats the token
// string itself as the user ID. Intended for local development and
// smoke tests; do not use in production.
type NopAuth struct{}

// Authenticate implements Authenticator.
func (NopAuth) Authenticate(_ context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrUnauthenticated
	}
	return token, nil
}

// StaticAuth maps fixed tokens to user IDs. Convenient for integration
// tests that need to impersonate multiple players.
type StaticAuth struct {
	Tokens map[string]string
}

// Authenticate implements Authenticator.
func (s StaticAuth) Authenticate(_ context.Context, token string) (string, error) {
	if uid, ok := s.Tokens[token]; ok {
		return uid, nil
	}
	return "", ErrUnauthenticated
}

// ClerkAuth adapts the project's Clerk-based AuthenticationConnector.
// It calls VerifyToken and returns the external ID as the user ID.
type ClerkAuth struct {
	Connector authentication.AuthenticationConnector
}

// Authenticate implements Authenticator.
func (c ClerkAuth) Authenticate(ctx context.Context, token string) (string, error) {
	if c.Connector == nil {
		return "", ErrUnauthenticated
	}
	uid, err := c.Connector.VerifyToken(ctx, token)
	if err != nil {
		return "", err
	}
	return uid, nil
}
