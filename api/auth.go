package api

import (
	"context"
	"errors"
	"strings"

	"github.com/cannonball10/foundation/connectors/authentication"
	authschema "github.com/cannonball10/foundation/schemas/authentication"
)

// Authenticator resolves an incoming request to a stable external
// identity. Implementations return the provider (clerk, guest, …) so
// the user-stitching layer in requireAuth can look up the right User
// row by (provider, externalID) without guessing.
//
// Kept narrow so tests can pass a fixture without wiring Clerk.
type Authenticator interface {
	// Authenticate returns the external ID and provider for the given
	// bearer token. ErrUnauthenticated signals a missing, malformed,
	// or invalid token.
	Authenticate(ctx context.Context, token string) (externalID string, provider authschema.AuthenticationProvider, err error)
}

// ErrUnauthenticated is returned by Authenticator when no valid
// identity can be derived from the request.
var ErrUnauthenticated = errors.New("unauthenticated")

// NopAuth accepts any non-empty bearer token and treats the token
// string itself as a guest authentication ID. The middleware maps it
// to a provider=guest User row. Intended for local development and
// smoke tests; do not use in production without Clerk in front.
type NopAuth struct{}

// Authenticate implements Authenticator.
func (NopAuth) Authenticate(_ context.Context, token string) (string, authschema.AuthenticationProvider, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", ErrUnauthenticated
	}
	return token, authschema.AuthenticationProvider_Guest, nil
}

// StaticAuth maps fixed tokens to external IDs under a single
// provider. Convenient for integration tests that need to impersonate
// specific accounts without running Clerk.
type StaticAuth struct {
	Tokens   map[string]string
	Provider authschema.AuthenticationProvider // defaults to Clerk if unset
}

// Authenticate implements Authenticator.
func (s StaticAuth) Authenticate(_ context.Context, token string) (string, authschema.AuthenticationProvider, error) {
	uid, ok := s.Tokens[token]
	if !ok {
		return "", "", ErrUnauthenticated
	}
	provider := s.Provider
	if provider == "" {
		provider = authschema.AuthenticationProvider_Clerk
	}
	return uid, provider, nil
}

// ClerkAuth adapts the project's Clerk-based AuthenticationConnector.
// It calls VerifyToken and returns the Clerk external ID paired with
// provider=clerk. Requires CLERK_SECRET_KEY + CLERK_BACKEND_URL.
type ClerkAuth struct {
	Connector authentication.AuthenticationConnector
}

// Authenticate implements Authenticator.
func (c ClerkAuth) Authenticate(ctx context.Context, token string) (string, authschema.AuthenticationProvider, error) {
	if c.Connector == nil {
		return "", "", ErrUnauthenticated
	}
	uid, err := c.Connector.VerifyToken(ctx, token)
	if err != nil {
		return "", "", err
	}
	return uid, authschema.AuthenticationProvider_Clerk, nil
}

// HybridAuth lets a single route accept both real JWTs (signed-in
// Clerk users) and opaque device tokens (guests). We sniff by shape:
// a JWT has three dot-separated base64url segments; anything else is
// a guest deviceId. Used on the player-facing endpoints so that
// "scan QR → type name → join" works without a sign-in gate, while
// still letting a player who has signed in get credit toward their
// persistent passport.
type HybridAuth struct {
	// Clerk, when nil, causes all callers to be treated as guests.
	// Handy for dev environments where Clerk isn't configured yet.
	Clerk Authenticator
	// Guest is the fallback for tokens that don't look like JWTs.
	// Defaults to NopAuth{}.
	Guest Authenticator
}

// Authenticate implements Authenticator.
func (h HybridAuth) Authenticate(ctx context.Context, token string) (string, authschema.AuthenticationProvider, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", ErrUnauthenticated
	}
	if h.Clerk != nil && looksLikeJWT(token) {
		return h.Clerk.Authenticate(ctx, token)
	}
	guest := h.Guest
	if guest == nil {
		guest = NopAuth{}
	}
	return guest.Authenticate(ctx, token)
}

// looksLikeJWT is a cheap shape check — three base64url segments
// separated by two dots. We don't actually try to decode them here;
// the downstream ClerkAuth will verify properly. The purpose is only
// to route: either "attempt Clerk verify" or "treat as guest."
func looksLikeJWT(token string) bool {
	dots := 0
	for _, r := range token {
		if r == '.' {
			dots++
		}
	}
	return dots == 2
}
