package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// WithAPIKey enforces the x-api-key header when key is non-empty.
// Returns a Gin middleware function.
func WithAPIKey(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If no API key is configured, skip validation
		if key == "" {
			c.Next()
			return
		}

		// Check if the x-api-key header matches
		if c.GetHeader("x-api-key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		c.Next()
	}
}
