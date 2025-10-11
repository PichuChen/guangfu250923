package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func APIKeyVerifier(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// if apiKey is not set, means we should accept all requests
		if apiKey == "" {
			c.Next()
			return
		}

		receivedKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if receivedKey == "" || receivedKey != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "reason": "invalid API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}
