package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewWriteRequestCache(
	pool *pgxpool.Pool,
	refreshInterval time.Duration,
	writeRateLimitSeconds int,
	writeRateLimitCount int,
	writeRateLimitPathPattern string,
) func(c *gin.Context) bool {
	paths := map[string]struct{}{}
	if writeRateLimitPathPattern != "" {
		for _, path := range strings.Split(writeRateLimitPathPattern, ",") {
			paths[strings.TrimSpace(path)] = struct{}{}
		}
	}

	type requestPattern struct {
		loadedAt time.Time
		patterns map[string]int
	}
	var cache atomic.Value
	loadRequestPattern := func(ctx context.Context) requestPattern {
		r := requestPattern{loadedAt: time.Now(), patterns: map[string]int{}}
		rows, err := pool.Query(
			ctx,
			`select ip, path, count(*) from request_logs
			where ip is not null and created_at > now() - ($1 * '1 second'::interval) 
			and method in ('POST','PATCH')
			group by ip, path`,
			writeRateLimitSeconds-60)
		if err != nil {
			return r
		}
		defer rows.Close()

		for rows.Next() {
			var ip, path, pat string
			var count int
			if err := rows.Scan(&ip, &path, &count); err != nil {
				continue
			}

			if len(paths) > 0 {
				pat = fmt.Sprintf("%v_%v", ip, path)
			} else {
				pat = ip
			}

			r.patterns[pat] += count
		}
		return r
	}
	cache.Store(loadRequestPattern(context.Background()))

	ensureFresh := func() requestPattern {
		v := cache.Load().(requestPattern)
		if time.Since(v.loadedAt) < refreshInterval {
			return v
		}
		go func() { cache.Store(loadRequestPattern(context.Background())) }()
		return v
	}

	return func(c *gin.Context) bool {
		if writeRateLimitSeconds == 0 || writeRateLimitCount == 0 {
			return false
		}
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPatch {
			return false
		}
		if _, ok := paths[c.FullPath()]; len(paths) > 0 && !ok {
			return false
		}

		rp := ensureFresh()
		cip := clientIP(c)

		if cip == "" {
			return false
		}

		var pat string
		if len(paths) > 0 {
			pat = fmt.Sprintf("%v_%v", cip, c.FullPath())
		} else {
			pat = cip
		}

		rp.patterns[pat] += 1
		return rp.patterns[pat] > writeRateLimitCount
	}
}
