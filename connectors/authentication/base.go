package authentication

import (
	"context"

	"github.com/cannonball10/foundation/schemas/authentication"
)

// AuthenticationConnector is an abstract authentication interface that can be extended for concrete implementations like JWT, OAuth, etc.
type AuthenticationConnector interface {
	VerifyToken(ctx context.Context, token string) (externalID string, err error)
	CreateUser(ctx context.Context, userID string) (externalID string, err error)
	AuthenticationProvider(ctx context.Context) (provider authentication.AuthenticationProvider, err error)
}

func DefaultAuthenticationConnector(ctx context.Context) (AuthenticationConnector, error) {
	return DefaultClerkAuthenticationConnector(ctx)
}
