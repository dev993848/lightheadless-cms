package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	rateLimitRequests = 10
	rateLimitWindow   = time.Minute
)

// bucket holds the request count and the start of the current window for one IP.
// mu protects only windowStart resets (rare); count is incremented atomically.
type bucket struct {
	mu          sync.Mutex
	windowStart time.Time
	count       atomic.Int64
}

// RateLimiter implements IP-based rate limiting.
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*bucket)}
}

func (rl *RateLimiter) getOrCreate(ip string) *bucket {
	rl.mu.RLock()
	b, ok := rl.buckets[ip]
	rl.mu.RUnlock()
	if ok {
		return b
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if b, ok = rl.buckets[ip]; ok {
		return b
	}
	b = &bucket{windowStart: time.Now()}
	rl.buckets[ip] = b
	return b
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
			rl.mu.Lock()
			for ip, b := range rl.buckets {
				b.mu.Lock()
				expired := now.Sub(b.windowStart) > rateLimitWindow
				b.mu.Unlock()
				if expired {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Middleware returns an HTTP middleware that enforces rate limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		b := rl.getOrCreate(ip)

		// Reset window if expired; mutex guards only this compound check-and-reset.
		now := time.Now()
		b.mu.Lock()
		if now.Sub(b.windowStart) > rateLimitWindow {
			b.windowStart = now
			b.count.Store(0)
		}
		b.mu.Unlock()

		// Atomic increment — no mutex needed for the hot path.
		if b.count.Add(1) > rateLimitRequests {
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
