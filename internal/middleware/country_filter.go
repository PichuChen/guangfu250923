package middleware

import (
	"context"
	"errors"
	"guangfu250923/internal/notify"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IPFilter (formerly CountryFilter) restricts POST/PATCH requests based on Cloudflare-provided country header.
// Despite the name change, current implementation still uses Cf-Ipcountry. If you later need pure IP allowlisting,
// extend here to parse client IP (see middleware.clientIP) and compare with an ALLOWED_IPS list.
//
// Environment variables:
//
//	ALLOWED_COUNTRIES   comma-separated list, case-insensitive (e.g. "TW,JP")
//	ALLOW_NO_COUNTRY    if "true", missing Cf-Ipcountry header is allowed
//	ALLOWED_IPS         optional comma-separated list of IPs or CIDRs (e.g. "1.2.3.4,10.0.0.0/8,2001:db8::/32"). If present, client IP must fall inside one of them.
//
// Behavior:
//   - Only affects POST & PATCH.
//   - If ALLOWED_COUNTRIES unset/empty => no-op.
//   - 403 on disallowed or missing (unless ALLOW_NO_COUNTRY=true).
func IPFilter(pool *pgxpool.Pool) gin.HandlerFunc {
	// Country list (optional)
	allowedCountriesRaw := os.Getenv("ALLOWED_COUNTRIES")
	allowSet := map[string]struct{}{}
	if strings.TrimSpace(allowedCountriesRaw) != "" {
		for _, p := range strings.Split(allowedCountriesRaw, ",") {
			p = strings.TrimSpace(strings.ToUpper(p))
			if p == "" {
				continue
			}
			allowSet[p] = struct{}{}
		}
	}

	allowNoHeader := strings.EqualFold(os.Getenv("ALLOW_NO_COUNTRY"), "true")

	// IP/CIDR list (optional)
	allowedIPsRaw := os.Getenv("ALLOWED_IPS")
	var ipNets []*net.IPNet
	if strings.TrimSpace(allowedIPsRaw) != "" {
		for _, token := range strings.Split(allowedIPsRaw, ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if strings.Contains(token, "/") {
				if _, network, err := net.ParseCIDR(token); err == nil {
					ipNets = append(ipNets, network)
				}
				continue
			}
			// Single IP -> convert to /32 or /128
			ip := net.ParseIP(token)
			if ip == nil {
				continue
			}
			var mask net.IPMask
			if ip.To4() != nil {
				mask = net.CIDRMask(32, 32)
			} else {
				mask = net.CIDRMask(128, 128)
			}
			ipNet := &net.IPNet{IP: ip, Mask: mask}
			ipNets = append(ipNets, ipNet)
		}
	}

	// Denylist cache (ip_denylist table). We keep a slice of *net.IPNet; single IP stored as /32 or /128.
	type denyCache struct {
		loadedAt time.Time
		nets     []*net.IPNet
		singles  map[string]struct{} // exact IP strings
	}
	var cache atomic.Value
	loadDeny := func(ctx context.Context) denyCache {
		dc := denyCache{loadedAt: time.Now(), singles: map[string]struct{}{}}
		if pool == nil {
			return dc
		}
		rows, err := pool.Query(ctx, `select pattern from ip_denylist`)
		if err != nil {
			return dc
		}
		defer rows.Close()
		for rows.Next() {
			var pat string
			if err := rows.Scan(&pat); err != nil {
				continue
			}
			pat = strings.TrimSpace(pat)
			if pat == "" {
				continue
			}
			if strings.Contains(pat, "/") {
				if _, netw, err := net.ParseCIDR(pat); err == nil {
					dc.nets = append(dc.nets, netw)
				}
				continue
			}
			ip := net.ParseIP(pat)
			if ip == nil {
				continue
			}
			dc.singles[ip.String()] = struct{}{}
		}
		return dc
	}
	// Preload once.
	cache.Store(loadDeny(context.Background()))
	// Background refresher (lazy, only if middleware invoked & stale > 60s)
	const refreshInterval = 60 * time.Second

	ensureFresh := func() denyCache {
		v := cache.Load().(denyCache)
		if time.Since(v.loadedAt) < refreshInterval {
			return v
		}
		// Refresh async to avoid blocking all requests.
		go func() { cache.Store(loadDeny(context.Background())) }()
		return v
	}

	// Fast no-op if no constraints (no allow countries, no allow IPs, and denylist empty)
	fastNoConstraint := len(allowSet) == 0 && len(ipNets) == 0

	isIPAllowed := func(ipStr string) bool {
		if len(ipNets) == 0 {
			return true
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		for _, n := range ipNets {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}

	isIPDenied := func(ipStr string, dc denyCache) bool {
		if ipStr == "" {
			return false
		}
		if _, ok := dc.singles[ipStr]; ok {
			return true
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		for _, n := range dc.nets {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}

	// block constructs a uniform 403 response and records an error for the RequestLogger.
	block := func(c *gin.Context, reason, ip string, details gin.H) {
		c.Error(errors.New("blocked: " + reason)) //nolint:errcheck
		payload := gin.H{"error": "blocked", "reason": reason, "ip": ip}
		for k, v := range details {
			payload[k] = v
		}
		c.JSON(http.StatusForbidden, payload)
		c.Abort()
	}

	writeRateLimitSeconds, _ := strconv.Atoi(os.Getenv("WRITE_RATE_LIMIT_INTERVAL_SECONDS"))
	writeRateLimitCount, _ := strconv.Atoi(os.Getenv("WRITE_RATE_LIMIT_COUNT"))
	writeRateLimitPathPattern := os.Getenv("WRITE_RATE_LIMIT_PATH_PATTERN")
	checkRateLimit := NewWriteRequestCache(pool, 60*time.Second, writeRateLimitSeconds, writeRateLimitCount, writeRateLimitPathPattern)

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		dc := ensureFresh()
		cip := clientIP(c)

		// Denylist precedes allow rules.
		if isIPDenied(cip, dc) {
			block(c, "ip denied", cip, gin.H{})
			return
		}

		if checkRateLimit(c) {
			var itemID string
			dc.singles[cip] = struct{}{}
			err := pool.QueryRow(context.Background(), `insert into ip_denylist(pattern,reason) values($1,$2) returning id`,
				cip, "rate limit").Scan(&itemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			webhook := os.Getenv("DISCORD_WEBHOOK_URL")
			if webhook != "" {
				clientIP := clientIP(c)
				country := strings.ToUpper(strings.TrimSpace(c.GetHeader("Cf-Ipcountry")))
				ipWithCountry := clientIP
				if country != "" {
					ipWithCountry = clientIP + " (" + country + ")"
				}

				ua := c.GetHeader("User-Agent")
				msg := "**自動封鎖 IP 🚫**\n"
				msg += "IP: " + ipWithCountry + "\n"
				msg += "User-Agent: " + ua
				payload := map[string]any{"id": itemID, "ip": clientIP, "country": country, "user_agent": ua}
				notify.SendDiscordWebhookAndRecordAsync(pool, webhook, "ip.rate_limit", itemID, msg, payload)
			}

			block(c, "ip not allowed", cip, gin.H{})
			return
		}

		// If there are no allow constraints, just proceed (still honoring denylist above).
		if fastNoConstraint {
			c.Next()
			return
		}

		// Allowlist IP check
		if len(ipNets) > 0 && !isIPAllowed(cip) {
			block(c, "ip not allowed", cip, gin.H{})
			return
		}

		// Country enforcement (after IP allow)
		if len(allowSet) > 0 {
			country := c.GetHeader("Cf-Ipcountry")
			if country == "" {
				if !allowNoHeader {
					block(c, "missing Cf-Ipcountry", cip, gin.H{})
					return
				}
			} else {
				country = strings.ToUpper(strings.TrimSpace(country))
				if _, ok := allowSet[country]; !ok {
					block(c, "disallowed country", cip, gin.H{"country": country})
					return
				}
			}
		}

		c.Next()
	}
}
