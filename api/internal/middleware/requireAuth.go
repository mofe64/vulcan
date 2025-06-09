package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mofe64/vulkan/api/internal/auth"
)

func RequireAuth(va *auth.VulkanAuth) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		rawToken := strings.TrimPrefix(auth, "Bearer ")

		idToken, err := va.Verifier.Verify(c, rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		var claims struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
		}
		_ = idToken.Claims(&claims)

		// add to request context for handlers to use
		c.Set("user_id", claims.Sub)
		c.Set("user_email", claims.Email)
		// todo: add permissions required for OPA middleware

		c.Next()
	}
}
