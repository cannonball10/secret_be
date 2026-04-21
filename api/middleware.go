package api

import (
	"net/http"
	"strings"

	authschema "github.com/cannonball10/foundation/schemas/authentication"
	"github.com/gin-gonic/gin"
)

// Gin context keys set by requireAuth.
const (
	// ctxUserIDKey holds our internal User.UserID (ULID). Handlers
	// should use userID(c) to read it.
	ctxUserIDKey = "ctx.userId"
	// ctxProviderKey holds the authentication provider that verified
	// this request (clerk, guest). Used by /me + /me/link.
	ctxProviderKey = "ctx.provider"
	// ctxAuthIDKey holds the raw external ID (Clerk subject or
	// deviceId). Used by /me/link to find the prior guest User row.
	ctxAuthIDKey = "ctx.authId"
)

// requireAuth is a middleware that enforces a valid bearer token,
// resolves it to an internal User (creating the row on first
// sighting), and stashes the internal UserID + provider + external
// ID on the gin context for downstream handlers.
func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		externalID, provider, err := s.auth.Authenticate(c.Request.Context(), token)
		if err != nil || externalID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthenticated",
			})
			return
		}

		user, err := s.users.Resolve(c.Request.Context(), provider, externalID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "user resolution failed",
			})
			return
		}

		c.Set(ctxUserIDKey, user.UserID)
		c.Set(ctxProviderKey, provider)
		c.Set(ctxAuthIDKey, externalID)
		c.Next()
	}
}

// userID pulls the internal User.UserID set by requireAuth.
func userID(c *gin.Context) string {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// authProvider pulls the provider that verified this request. Returns
// "" when no auth middleware has run (e.g. on /healthz).
func authProvider(c *gin.Context) authschema.AuthenticationProvider {
	v, ok := c.Get(ctxProviderKey)
	if !ok {
		return ""
	}
	p, _ := v.(authschema.AuthenticationProvider)
	return p
}

// authExternalID pulls the external ID (Clerk subject or deviceId).
func authExternalID(c *gin.Context) string {
	v, ok := c.Get(ctxAuthIDKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// extractBearerToken pulls the token out of "Authorization: Bearer X".
// Falls back to the X-Auth-Token header, then to a "token" query-string
// parameter — browser EventSource cannot set custom headers, so the SSE
// clients embed the token in the URL. In production you'd either
// terminate auth upstream or swap EventSource for fetch-based SSE so
// that query-string tokens never hit logs.
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
		return strings.TrimSpace(auth)
	}
	if t := strings.TrimSpace(c.GetHeader("X-Auth-Token")); t != "" {
		return t
	}
	return strings.TrimSpace(c.Query("token"))
}
