package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// ModifyAPIKeyRequired enforces that the request includes an API key that exists in ALLOW_MODIFY_API_KEY_LIST.
// Accepted headers: X-Api-Key: <key> or Authorization: Bearer <key>
// If ALLOW_MODIFY_API_KEY_LIST is empty, all requests are rejected.
func ModifyAPIKeyRequired() gin.HandlerFunc {
	// Parse env allowlist once per middleware instance
	allowed := parseAllowlist(os.Getenv("ALLOW_MODIFY_API_KEY_LIST"))
	return func(c *gin.Context) {
		if len(allowed) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "modification not allowed"})
			c.Abort()
			return
		}
		key := strings.TrimSpace(c.GetHeader("X-Api-Key"))
		if key == "" {
			// Try Authorization: Bearer <key>
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				key = strings.TrimSpace(auth[7:])
			}
		}
		if key == "" || !allowed[key] {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func parseAllowlist(s string) map[string]bool {
	m := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			m[p] = true
		}
	}
	return m
}

// IsAPIKeyAllowed returns true if the request carries an API key (X-Api-Key or Bearer) contained in ALLOW_MODIFY_API_KEY_LIST.
// When the allowlist is empty, it returns false.
func IsAPIKeyAllowed(c *gin.Context) bool {
	allowed := parseAllowlist(os.Getenv("ALLOW_MODIFY_API_KEY_LIST"))
	if len(allowed) == 0 {
		return false
	}
	key := strings.TrimSpace(c.GetHeader("X-Api-Key"))
	if key == "" {
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			key = strings.TrimSpace(auth[7:])
		}
	}
	if key == "" {
		return false
	}
	return allowed[key]
}
