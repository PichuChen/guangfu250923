package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// MemoryCacheInvalidator clears in-memory GET cache after successful write operations.
// It targets known resource prefixes to avoid total cache flush.
func MemoryCacheInvalidator() gin.HandlerFunc {
    // map route patterns to path prefixes for invalidation
    prefixes := []string{
        "/shelters",
        "/medical_stations",
        "/mental_health_resources",
        "/accommodations",
        "/shower_stations",
        "/water_refill_stations",
        "/restrooms",
        "/volunteer_organizations",
        "/human_resources",
        "/supplies",
        "/supply_items",
        "/reports",
        "/spam_results",
        "/supply_providers",
        "/places",
        "/requirements_hr",
    "/requirements_supplies",
    }
    return func(c *gin.Context) {
        method := c.Request.Method
        if method == http.MethodGet || method == http.MethodOptions || method == http.MethodHead {
            c.Next()
            return
        }
        // process write request
        c.Next()
        // invalidate only if success (2xx/3xx)
        if c.Writer.Status() >= 200 && c.Writer.Status() < 400 {
            // choose the best prefix for this request path
            path := c.Request.URL.Path
            for _, p := range prefixes {
                if strings.HasPrefix(path, p) {
                    InvalidateMemoryCacheByPrefix(p)
                    return
                }
            }
            // fallback: clear all if path not recognized
            InvalidateAllMemoryCache()
        }
    }
}
