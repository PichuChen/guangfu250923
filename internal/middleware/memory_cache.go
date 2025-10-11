package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type cacheStore struct {
	mu    sync.RWMutex
	items map[string]*memoryCacheEntry
}

// memoryCacheEntry represents a cached HTTP response
type memoryCacheEntry struct {
	status  int
	header  http.Header
	body    []byte
	expires time.Time
	size    int
}

// MemoryCache returns a middleware that caches successful GET responses in-memory for ttl.
// It also clears the cache on any state-changing request (POST, PATCH, PUT, DELETE) to avoid stale data.
// maxBody limits the size of response body to cache (in bytes). Set <=0 for 1MB default.
func MemoryCache(ttl time.Duration, maxBody int) gin.HandlerFunc {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if maxBody <= 0 {
		maxBody = 1 << 20 // 1MB
	}
	store := &cacheStore{items: make(map[string]*memoryCacheEntry)}
	setGlobalStore(store)

	// helper to build cache key
	buildKey := func(c *gin.Context) string {
		// Use the actual request path (not the route pattern) to keep distinct keys per entity id.
		path := c.Request.URL.Path
		return c.Request.Method + " " + path + "?" + c.Request.URL.RawQuery
	}

	// simple allow-list for caching; skip admin/auth/healthz by default
	shouldSkip := func(c *gin.Context) bool {
		if c.Request.Method != http.MethodGet {
			return false
		}
		p := c.FullPath()
		if p == "" {
			p = c.Request.URL.Path
		}
		if strings.HasPrefix(p, "/_admin/") || strings.HasPrefix(p, "/auth/") || p == "/healthz" {
			return true
		}
		if strings.HasPrefix(p, "/swagger/") {
			return true
		}
		return false
	}

	return func(c *gin.Context) {
		// Only cache GET
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}
		if shouldSkip(c) {
			c.Next()
			return
		}

		key := buildKey(c)

		// Try read lock and serve from cache if fresh
		store.mu.RLock()
		if ent, ok := store.items[key]; ok {
			if time.Now().Before(ent.expires) {

				// 檢查如果 Request 有 If-None-Match 標頭，且與緩存的 ETag 匹配，則返回 304 Not Modified
				if inm := c.Request.Header.Get("If-None-Match"); inm != "" {
					etag := ent.header.Get("ETag")
					parts := strings.Split(inm, ",")
					for _, p := range parts {
						if strings.TrimSpace(p) == etag {

							c.Writer.Header().Set("ETag", etag)
							if c.Writer.Header().Get("Cache-Control") == "" {
								c.Writer.Header().Set("Cache-Control", cacheControlForPath(c.FullPath(), c.Request.URL.RawQuery))
							}
							c.Writer.Header().Set("Vary", "Accept-Encoding")
							if c.Writer.Header().Get("Last-Modified") == "" {
								c.Writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
							}
							c.Writer.WriteHeader(http.StatusNotModified)
							store.mu.RUnlock()
							// Abort so downstream handlers/middlewares are not executed
							c.Abort()
							return
						}
					}
				}

				// serve cached
				for k, vals := range ent.header {
					// Overwrite existing header values to cached ones
					c.Writer.Header().Del(k)
					for _, v := range vals {
						slog.Info("memory cache hit", "header", k, "value", v)
						c.Writer.Header().Add(k, v)
					}
				}

				c.Writer.WriteHeader(ent.status)
				if len(ent.body) > 0 {
					c.Writer.Write(ent.body)
				}
				store.mu.RUnlock()
				// Abort so downstream handlers/middlewares are not executed
				c.Abort()
				return
			}
		}
		store.mu.RUnlock()

		// Cache miss: capture response
		rec := &memRecorder{ResponseWriter: c.Writer, limit: maxBody}
		c.Writer = rec
		c.Next()

		status := c.Writer.Status()
		slog.Info("memory cache miss", "path", c.Request.URL.Path, "status", status, "size", rec.buf.Len(), "exceeded", rec.exceeded)

		// Only cache successful 200 OK
		if status != http.StatusOK {
			return
		}
		// Skip if exceeded size cap
		if rec.exceeded {
			return
		}
		// store final headers/body/status with TTL
		hdr := http.Header{}
		for k, v := range rec.Header() {
			vv := make([]string, len(v))
			copy(vv, v)
			hdr[k] = vv
		}
		bodyCopy := make([]byte, rec.buf.Len())
		copy(bodyCopy, rec.buf.Bytes())

		ent := &memoryCacheEntry{status: status, header: hdr, body: bodyCopy, expires: time.Now().Add(ttl), size: len(bodyCopy)}

		store.mu.Lock()
		store.items[key] = ent
		store.mu.Unlock()
	}
}

// memRecorder buffers response up to a limit to allow caching.
type memRecorder struct {
	gin.ResponseWriter
	status   int
	buf      bytes.Buffer
	limit    int
	exceeded bool
}

func (r *memRecorder) WriteHeader(code int) {
	slog.Info("memRecorder WriteHeader", "code", code)
	r.status = code

	r.ResponseWriter.WriteHeader(code)
}

func (r *memRecorder) WriteString(s string) (int, error) {
	if !r.exceeded {
		if r.buf.Len()+len(s) > r.limit {
			r.exceeded = true
			// do not cache; but still pass through
		} else {
			r.buf.WriteString(s)
		}
	}
	return r.ResponseWriter.WriteString(s)
}
func (r *memRecorder) Write(b []byte) (int, error) {
	if !r.exceeded {
		if r.buf.Len()+len(b) > r.limit {
			r.exceeded = true
			// do not cache; but still pass through
		} else {
			r.buf.Write(b)
		}
	}
	return r.ResponseWriter.Write(b)
}

// ---- Global store access & invalidation helpers ----

var globalMu sync.RWMutex
var globalStore *cacheStore

func setGlobalStore(s *cacheStore) {
	globalMu.Lock()
	globalStore = s
	globalMu.Unlock()
}

// InvalidateAllMemoryCache clears all cached entries.
func InvalidateAllMemoryCache() {
	globalMu.RLock()
	s := globalStore
	globalMu.RUnlock()
	if s == nil {
		return
	}
	s.mu.Lock()
	s.items = make(map[string]*memoryCacheEntry)
	s.mu.Unlock()
}

// InvalidateMemoryCacheByPrefix clears all GET cache entries whose path starts with the given prefix.
// Prefix should be a URL path prefix (e.g., "/shelters").
func InvalidateMemoryCacheByPrefix(prefix string) {
	if prefix == "" || prefix == "/" {
		InvalidateAllMemoryCache()
		return
	}
	globalMu.RLock()
	s := globalStore
	globalMu.RUnlock()
	if s == nil {
		return
	}
	s.mu.Lock()
	for k := range s.items {
		// Key format: "GET /path?query"
		if strings.HasPrefix(k, "GET "+prefix) {
			delete(s.items, k)
		}
	}
	s.mu.Unlock()
}

// InvalidateMemoryCachePaths clears cache entries for the exact path(s), any query string.
// It matches keys with the given path prefix followed by either end or '?'.
func InvalidateMemoryCachePaths(paths ...string) {
	globalMu.RLock()
	s := globalStore
	globalMu.RUnlock()
	if s == nil || len(paths) == 0 {
		return
	}
	s.mu.Lock()
	for _, p := range paths {
		exact := "GET " + p
		prefixQ := exact + "?"
		for k := range s.items {
			if k == exact || strings.HasPrefix(k, prefixQ) {
				delete(s.items, k)
			}
		}
	}
	s.mu.Unlock()
}
