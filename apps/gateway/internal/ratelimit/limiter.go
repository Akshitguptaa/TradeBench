// Package ratelimit provides in-memory per-key token-bucket rate limiting
// using golang.org/x/time/rate.
package ratelimit

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter maintains a token-bucket rate limiter per key (typically contestant_id).
type Limiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rpm      int // requests per minute
}

// New creates a Limiter that allows rpm requests per minute per key.
func New(rpm int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
		rpm:      rpm,
	}
}

// getLimiter returns (or lazily creates) the rate.Limiter for the given key.
func (l *Limiter) getLimiter(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.limiters[key]
	if !ok {
		// rate.Every converts an interval to a rate.Limit.
		// rpm requests per minute → 1 token every (60/rpm) seconds.
		rps := rate.Limit(float64(l.rpm) / 60.0)
		lim = rate.NewLimiter(rps, l.rpm) // burst = rpm
		l.limiters[key] = lim
	}
	return lim
}

// Allow reports whether a request from the given key should be permitted.
func (l *Limiter) Allow(key string) bool {
	return l.getLimiter(key).Allow()
}

// KeyFunc extracts the rate-limit key from a request. By convention we use the
// contestant_id stored in the request context by the auth middleware.
type KeyFunc func(r *http.Request) string

// Middleware returns HTTP middleware that enforces the rate limit using the
// provided KeyFunc. Requests without a key (e.g. unauthenticated) are passed
// through — auth middleware should reject them first.
func Middleware(lim *Limiter, keyFn KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				// No key available (unauthenticated path); let auth middleware handle it.
				next.ServeHTTP(w, r)
				return
			}
			if !lim.Allow(key) {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
