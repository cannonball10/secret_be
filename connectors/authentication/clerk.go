package authentication

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/cannonball10/foundation/schemas/authentication"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"
)

type ClerkAuthentication struct {
	Client *user.Client
}

func NewClerkAuthentication(client *user.Client) *ClerkAuthentication {
	return &ClerkAuthentication{Client: client}
}

func DefaultClerkAuthenticationConnector(ctx context.Context) (*ClerkAuthentication, error) {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		return nil, errors.New("CLERK_SECRET_KEY is not set")
	}
	backendURL := os.Getenv("CLERK_BACKEND_URL")
	if backendURL == "" {
		return nil, errors.New("CLERK_BACKEND_URL is not set")
	}
	// clerk.SetKey populates the package-level backend client that
	// jwt.Verify uses to fetch the JWKs. Without this the JWKs call
	// goes out without proper authorization and Clerk returns
	// "authorization_header_format_invalid" on the backend fetch —
	// which surfaces as a token-verify failure even though the JWT
	// itself is fine. The client we build below is kept for the
	// CreateUser path; the jwt.Verify path doesn't see it.
	clerk.SetKey(secretKey)
	client := user.NewClient(&clerk.ClientConfig{BackendConfig: clerk.BackendConfig{Key: &secretKey, URL: &backendURL}})
	return NewClerkAuthentication(client), nil
}

func (c *ClerkAuthentication) AuthenticationProvider(ctx context.Context) (provider authentication.AuthenticationProvider, err error) {
	return authentication.AuthenticationProvider_Clerk, nil
}

func (c *ClerkAuthentication) VerifyToken(ctx context.Context, token string) (externalID string, err error) {
	token = strings.TrimPrefix(token, "Bearer ")
	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
		Token: token,
	})
	if err != nil {
		slog.ErrorContext(ctx, "clerk VerifyToken failed", "error", err)
		return "", err
	}
	return claims.Subject, nil
}

func (c *ClerkAuthentication) CreateUser(ctx context.Context, userID string) (externalID string, err error) {
	user, err := c.Client.Create(ctx, &user.CreateParams{
		ExternalID: &userID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "clerk CreateUser failed", "error", err)
		return "", err
	}
	return user.ID, nil
}
