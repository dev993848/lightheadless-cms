package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	rateLimitRequests = 10
	rateLimitWindow   = time.Minute
)

// bucket holds the request count and the start of the current window for one IP.
type bucket struct {
	mu        sync.Mutex
	count     int
	windowStart time.Time
}

// RateLimiter implements IP-based rate limiting.
type RateLimiter struct {
	buckets sync.Map // map[string]*bucket
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

// Cleanup removes stale IP buckets every 5 minutes until ctx is cancelled.
func (rl *RateLimiter) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			rl.buckets.Range(func(key, value any) bool {
				b := value.(*bucket)
				b.mu.Lock()
				expired := now.Sub(b.windowStart) > rateLimitWindow
				b.mu.Unlock()
				if expired {
					rl.buckets.Delete(key)
				}
				return true
			})
		}
	}
}

// Middleware returns an HTTP middleware that enforces rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)

		val, _ := rl.buckets.LoadOrStore(ip, &bucket{windowStart: time.Now()})
		b := val.(*bucket)

		b.mu.Lock()
		now := time.Now()
		if now.Sub(b.windowStart) > rateLimitWindow {
			b.count = 0
			b.windowStart = now
		}
		b.count++
		count := b.count
		b.mu.Unlock()

		if count > rateLimitRequests {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "error",
				"message": "Too many requests. Please try again later.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// realIP extracts the client IP, respecting X-Forwarded-For and X-Real-IP headers.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list.
		if idx := len(xff); idx > 0 {
			for i, ch := range xff {
				if ch == ',' {
					xff = xff[:i]
					break
				}
			}
			return xff
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
