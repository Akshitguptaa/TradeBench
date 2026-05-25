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
	rpm      int
}

func New(rpm int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
		rpm:      rpm,
	}
}

// lazily creates a token-bucket limiter for each unique key (contestant)
func (l *Limiter) getLimiter(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.limiters[key]
	if !ok {
		rps := rate.Limit(float64(l.rpm) / 60.0)
		lim = rate.NewLimiter(rps, l.rpm) // burst = rpm
		l.limiters[key] = lim
	}
	return lim
}

func (l *Limiter) Allow(key string) bool {
	return l.getLimiter(key).Allow()
}

type KeyFunc func(r *http.Request) string

func Middleware(lim *Limiter, keyFn KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
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
