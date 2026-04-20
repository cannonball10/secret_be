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
