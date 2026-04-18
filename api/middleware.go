package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ctxUserIDKey is the gin-context key under which the authenticated
// user ID is stored by requireAuth.
const ctxUserIDKey = "ctx.userId"

// requireAuth is a middleware that enforces a valid bearer token and
// stashes the resolved user ID on the gin context.
func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		userID, err := s.auth.Authenticate(c.Request.Context(), token)
		if err != nil || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthenticated",
			})
			return
		}
		c.Set(ctxUserIDKey, userID)
		c.Next()
	}
}

// userID pulls the user ID set by requireAuth. Returns "" if absent.
func userID(c *gin.Context) string {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// extractBearerToken pulls the token out of "Authorization: Bearer X".
// Also accepts a raw token in the X-Auth-Token header as a convenience
// for device clients that do not want to set Authorization.
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
		return strings.TrimSpace(auth)
	}
	return strings.TrimSpace(c.GetHeader("X-Auth-Token"))
}
