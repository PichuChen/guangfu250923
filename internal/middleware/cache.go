package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheHeaders adds basic caching headers (ETag, Cache-Control) for idempotent GET responses.
// It computes a weak ETag from the response body for 200 OK GET responses up to a size limit.
// If the client sends If-None-Match matching the computed ETag, a 304 Not Modified is returned.
func CacheHeaders(maxBody int) gin.HandlerFunc {
	if maxBody <= 0 {
		maxBody = 512 * 1024 // 512KB buffer threshold
	}
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}
		rw := &cacheRecorder{ResponseWriter: c.Writer, status: 200, limit: maxBody}
		c.Writer = rw
		c.Next()

		// Non-200: just ensure buffered content flushed if we buffered
		if rw.status != http.StatusOK {
			if !rw.streaming {
				writeBuffered(rw)
			}
			return
		}

		// If we had to stream (body exceeded limit) we cannot safely set ETag after body start
		if rw.streaming {
			// We can still set a basic Cache-Control if absent
			if rw.Header().Get("Cache-Control") == "" {
				rw.Header().Set("Cache-Control", cacheControlForPath(c.FullPath(), c.Request.URL.RawQuery))
			}
			return
		}

		body := rw.buf.Bytes()
		// Decide strong vs weak ETag based on path: only requests under /photos/* get strong ETag
		pattern := c.FullPath()
		if pattern == "" {
			pattern = c.Request.URL.Path
		}
		isPhoto := strings.HasPrefix(pattern, "/photos")

		h := sha256.Sum256(body)
		var etagHeader string
		var rawTag string // normalized tag for comparison
		if isPhoto {
			// Strong ETag: full SHA-256
			rawTag = hex.EncodeToString([]byte(c.FullPath()))
			etagHeader = fmt.Sprintf("\"%s\"", rawTag)
		} else {
			// Weak ETag: shorter tag
			rawTag = hex.EncodeToString(h[:8])
			etagHeader = fmt.Sprintf("W/\"%s\"", rawTag)
		}
		hdr := rw.Header()

		// Handle conditional If-None-Match
		if inm := c.Request.Header.Get("If-None-Match"); inm != "" {
			parts := strings.Split(inm, ",")
			for _, p := range parts {
				tok := normalizeETagToken(p)
				if tok == rawTag {
					hdr.Del("Content-Length")
					hdr.Set("ETag", etagHeader)
					if hdr.Get("Cache-Control") == "" {
						hdr.Set("Cache-Control", cacheControlForPath(c.FullPath(), c.Request.URL.RawQuery))
					}
					hdr.Set("Vary", "Accept-Encoding")
					if hdr.Get("Last-Modified") == "" {
						hdr.Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
					}
					rw.ResponseWriter.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}

		hdr.Set("ETag", etagHeader)
		if hdr.Get("Cache-Control") == "" {
			hdr.Set("Cache-Control", cacheControlForPath(c.FullPath(), c.Request.URL.RawQuery))
		}
		hdr.Add("Vary", "Accept-Encoding")
		if hdr.Get("Last-Modified") == "" {
			hdr.Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		}
		hdr.Set("Content-Length", strconv.Itoa(len(body)))

		// Flush buffered body now
		if rw.headerWritten {
			rw.ResponseWriter.WriteHeader(rw.status)
		}
		if len(body) > 0 {
			rw.ResponseWriter.Write(body)
		}
	}
}

type cacheRecorder struct {
	gin.ResponseWriter
	status        int
	buf           bytes.Buffer
	limit         int
	headerWritten bool
	streaming     bool // true if body exceeded limit and we streamed directly
}

func (r *cacheRecorder) WriteHeader(code int) {
	r.status = code
	r.headerWritten = true /* defer real write until flush */
}
func (r *cacheRecorder) Write(b []byte) (int, error) {
	// If already streaming, just passthrough
	if r.streaming {
		return r.ResponseWriter.Write(b)
	}
	// Try to buffer
	if r.buf.Len()+len(b) > r.limit {
		// Write what fits then flush and switch to streaming mode
		remain := r.limit - r.buf.Len()
		if remain > 0 {
			r.buf.Write(b[:remain])
			b = b[remain:]
		}
		// Flush buffered part now (headers not yet written to client)
		if !r.headerWritten {
			// To allow downstream to set status we delay until end; but since we must stream now, force header write
			r.headerWritten = true
			// underlying status might still be 200 (default)
			r.ResponseWriter.WriteHeader(r.status)
		}
		if r.buf.Len() > 0 {
			r.ResponseWriter.Write(r.buf.Bytes())
			// Clear buffer to avoid double use; but keep contents for potential misuse? We discard to save memory
			r.buf.Reset()
		}
		// Now write the remainder and mark streaming (no ETag possible)
		if len(b) > 0 {
			r.ResponseWriter.Write(b)
		}
		r.streaming = true
		return len(b) + remain, nil
	}
	// Still within limit; buffer only
	r.buf.Write(b)
	return len(b), nil
}

func writeBuffered(r *cacheRecorder) {
	if r.headerWritten {
		r.ResponseWriter.WriteHeader(r.status)
	}
	if r.buf.Len() > 0 {
		r.ResponseWriter.Write(r.buf.Bytes())
	}
}

// cacheControlForPath decides cache policy based on path pattern and query string.
func cacheControlForPath(pattern, rawQuery string) string {
	// public: 僅限沒有登入的東西
	// no-store: 即時變更，不能快取 (登入頁面、管理介面)
	// no-cache: 需要即時變更，但允許瀏覽器/中介快取，回傳前需重新驗證 (動態內容、使用者相關)
	// public, max-age=xxx: 允許公開快取，適合不常變更的靜態內容 (大部分 GET API)
	// private, max-age=xxx: 允許使用者端快取，禁止中介快取 (使用者專屬內容)
	// must-revalidate: 過期後需重新驗證 (避免過期後繼續使用陳舊內容)
	// immutable: 不會變更的內容，允許長期快取
	if strings.HasPrefix(pattern, "/photos/") {
		// 照片內容不常變更，允許公開快取三天
		return "public, max-age=31536000, immutable"
	}

	if strings.HasPrefix(pattern, "/_admin/") || pattern == "/healthz" || strings.HasPrefix(pattern, "/auth/") {
		return "no-store"
	}
	// Highly dynamic aggregated embedding: disable cache to reflect near real-time changes
	if pattern == "/supplies" || pattern == "/human_resources" {
		// 需要即時回應
		return "public, no-cache"
	} else if strings.HasPrefix(pattern, "/supplies") || strings.HasPrefix(pattern, "/human_resources") {
		// 需要即時回應
		return "public, no-cache"
	}
	// list endpoints usually have limit/offset
	// if strings.Contains(rawQuery, "offset=") || strings.Contains(rawQuery, "limit=") {
	// 	// 廁所等等，不需要及時變更
	// 	return "public, max-age=300"
	// }
	// entity endpoints (contain :id)
	// if strings.Contains(pattern, ":id") {
	// 	return "public, max-age=60"
	// }
	// default short cache
	return "public, no-cache"
}

// normalizeETagToken strips weak validators and quotes, returning the raw tag for comparison.
func normalizeETagToken(token string) string {
	t := strings.TrimSpace(token)
	// Strip weak prefix W/
	if strings.HasPrefix(t, "W/") || strings.HasPrefix(t, "w/") {
		t = strings.TrimSpace(t[2:])
	}
	// Remove surrounding quotes if present
	if len(t) >= 2 && t[0] == '"' && t[len(t)-1] == '"' {
		t = t[1 : len(t)-1]
	}
	return t
}
