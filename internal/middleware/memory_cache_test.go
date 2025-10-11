package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
)

// Test that middleware does not alter 404 status on missing routes and does not cache it.
func TestMemoryCache_Respects404_NoRoute(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.Use(MemoryCache(100*time.Millisecond, 1024))

    // No route registered; Gin should return 404
    req := httptest.NewRequest(http.MethodGet, "/missing", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404 on first request, got %d", w.Code)
    }

    // Second request should also be 404 (and not be served from cache as 200)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req)
    if w2.Code != http.StatusNotFound {
        t.Fatalf("expected 404 on second request, got %d", w2.Code)
    }
}

// Test that WriteString path with explicit 404 is preserved.
func TestMemoryCache_Respects404_WriteString(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.Use(MemoryCache(100*time.Millisecond, 1024))

    r.GET("/echo404", func(c *gin.Context) {
        c.Writer.WriteHeader(http.StatusNotFound)
        c.Writer.WriteString("nope")
    })

    req := httptest.NewRequest(http.MethodGet, "/echo404", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d", w.Code)
    }
    if got := w.Body.String(); got != "nope" {
        t.Fatalf("unexpected body: %q", got)
    }

    // Repeat to ensure not cached as 200
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req)
    if w2.Code != http.StatusNotFound {
        t.Fatalf("expected 404 again, got %d", w2.Code)
    }
}
