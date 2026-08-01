// Package api provides HTTP middleware shared across handlers.
package api

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// redactedQueryKeys are query parameters whose values must never reach the
// access log. The token parameter carries a live JWT for SSE and export
// endpoints, so logging the raw query string would persist bearer tokens.
var redactedQueryKeys = map[string]bool{
	"token": true,
}

// redactQuery rewrites a "path?query" string, replacing the values of any
// sensitive query parameters with "REDACTED". Unparseable input is returned
// with the query stripped entirely, failing closed.
func redactQuery(pathWithQuery string) string {
	i := indexByte(pathWithQuery, '?')
	if i < 0 {
		return pathWithQuery
	}
	path, rawQuery := pathWithQuery[:i], pathWithQuery[i+1:]
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return path
	}
	redacted := false
	for key := range values {
		if redactedQueryKeys[key] {
			values.Set(key, "REDACTED")
			redacted = true
		}
	}
	if !redacted {
		return pathWithQuery
	}
	return path + "?" + values.Encode()
}

// indexByte is strings.IndexByte without importing strings for one call.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// redactingLogger returns a gin logger middleware that scrubs sensitive query
// parameters (JWTs passed in the URL) before writing the access log line.
func redactingLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %s | %3d | %13v | %-15s | %-7s %s\n",
			p.TimeStamp.Format("2006/01/02 - 15:04:05"),
			p.StatusCode,
			p.Latency,
			p.ClientIP,
			p.Method,
			redactQuery(p.Path),
		)
	})
}

// clampPagination bounds page and limit to safe ranges. A non-positive or
// negative limit would otherwise disable GORM's LIMIT clause and dump entire
// tables; a non-positive page would produce a negative OFFSET.
func clampPagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// rateLimiter is a fixed-window per-key limiter used to throttle abuse-prone
// endpoints such as login. It is intentionally small and in-process; a
// multi-replica deployment should front it with a shared limiter, but even
// per-replica limiting removes unbounded online password guessing.
type rateLimiter struct {
	mu       sync.Mutex
	windows  map[string]*window
	limit    int
	interval time.Duration
}

type window struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, interval time.Duration) *rateLimiter {
	return &rateLimiter{
		windows:  make(map[string]*window),
		limit:    limit,
		interval: interval,
	}
}

// allow reports whether a request from key may proceed, incrementing its count.
func (r *rateLimiter) allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.windows[key]
	if !ok || now.After(w.reset) {
		r.windows[key] = &window{count: 1, reset: now.Add(r.interval)}
		// Opportunistically evict expired windows so the map cannot grow without
		// bound under a spray of distinct source IPs.
		if len(r.windows) > 1024 {
			for k, v := range r.windows {
				if now.After(v.reset) {
					delete(r.windows, k)
				}
			}
		}
		return true
	}
	if w.count >= r.limit {
		return false
	}
	w.count++
	return true
}

// rateLimitMiddleware throttles requests per client IP. Time is injected via a
// clock func so tests are deterministic; production passes time.Now.
func rateLimitMiddleware(limiter *rateLimiter, now func() time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP(), now()) {
			c.Header("Retry-After", fmt.Sprintf("%d", int(limiter.interval.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
			c.Abort()
			return
		}
		c.Next()
	}
}
