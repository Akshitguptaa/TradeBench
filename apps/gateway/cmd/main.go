package main

// =============================================================================
// TradeBench API Gateway — Route Contract
// =============================================================================
//
// All routes are served on :8080. JWT (Authorization: Bearer <token>) is
// required for every route except /health.
//
//   POST   /api/v1/auth/token          → issue JWT (dev mode: no real auth)
//   POST   /api/v1/submissions         → upload binary → submission svc
//   GET    /api/v1/submissions/:id     → get submission status
//   GET    /api/v1/runs/:id            → get run status + metrics
//   GET    /api/v1/leaderboard         → current top-50 (REST fallback)
//   WS     /ws/leaderboard             → WebSocket stream → leaderboard svc
//   GET    /health                     → liveness probe (no auth)
//
// See docs/api-contract.md for full request/response schemas.
// =============================================================================

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tradebench/gateway/config"
	"github.com/tradebench/gateway/internal/auth"
	"github.com/tradebench/gateway/internal/proxy"
	"github.com/tradebench/gateway/internal/ratelimit"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const contestantKey contextKey = "contestant_id"

func main() {
	cfg := config.Load()

	// --- Auth ---
	jwtAuth := auth.New(cfg.JWTSecret, cfg.JWTExpiry)

	// --- Rate limiters ---
	globalLimiter := ratelimit.New(cfg.GlobalRPM)
	submissionLimiter := ratelimit.New(cfg.SubmissionRPM)

	keyFn := func(r *http.Request) string {
		if v, ok := r.Context().Value(contestantKey).(string); ok {
			return v
		}
		return ""
	}

	// --- Reverse proxies ---
	submissionProxy := proxy.NewProxy(cfg.SubmissionURL)
	leaderboardProxy := proxy.NewProxy(cfg.LeaderboardURL)
	scorerProxy := proxy.NewProxy(cfg.ScorerURL)
	ingesterProxy := proxy.NewProxy(cfg.IngesterURL)

	// --- Mux ---
	mux := http.NewServeMux()

	// Health (no auth, no rate limit)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Issue JWT (dev mode)
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			ContestantID string `json:"contestant_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ContestantID == "" {
			http.Error(w, `{"error":"contestant_id is required"}`, http.StatusBadRequest)
			return
		}

		token, err := jwtAuth.IssueToken(body.ContestantID)
		if err != nil {
			http.Error(w, `{"error":"failed to issue token"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      token,
			"expires_in": int(cfg.JWTExpiry.Seconds()),
		})
	})

	// Submissions — POST (create) and GET (status)
	mux.HandleFunc("/api/v1/submissions", func(w http.ResponseWriter, r *http.Request) {
		submissionProxy.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/v1/submissions/", func(w http.ResponseWriter, r *http.Request) {
		submissionProxy.ServeHTTP(w, r)
	})

	// Runs — GET status + metrics
	mux.HandleFunc("/api/v1/runs/", func(w http.ResponseWriter, r *http.Request) {
		// Route to ingester for raw data, scorer for computed metrics.
		// For now, proxy to ingester; scorer augments via Kafka.
		ingesterProxy.ServeHTTP(w, r)
		_ = scorerProxy // available when scorer endpoint is added
	})

	// Leaderboard — REST fallback
	mux.HandleFunc("/api/v1/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		leaderboardProxy.ServeHTTP(w, r)
	})

	// Leaderboard — WebSocket stream
	mux.HandleFunc("/ws/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		leaderboardProxy.ServeHTTP(w, r)
	})

	// --- Middleware chain ---
	// Build the final handler: JWT auth → rate limit → mux
	var handler http.Handler = mux

	// Submission-specific rate limit (applied before global, checked on path)
	handler = submissionRateLimit(handler, submissionLimiter, keyFn)

	// Global rate limit
	handler = ratelimit.Middleware(globalLimiter, keyFn)(handler)

	// JWT auth (skips /health)
	handler = jwtMiddleware(handler, jwtAuth)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("gateway listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// jwtMiddleware validates the Bearer token on all routes except /health and
// injects the contestant_id into the request context.
func jwtMiddleware(next http.Handler, jwtAuth *auth.JWTAuth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Allow token issuance without existing JWT
		if r.URL.Path == "/api/v1/auth/token" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		contestantID, err := jwtAuth.VerifyToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contestantKey, contestantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// submissionRateLimit applies a stricter rate limit specifically to the
// /api/v1/submissions path.
func submissionRateLimit(next http.Handler, lim *ratelimit.Limiter, keyFn ratelimit.KeyFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/submissions") {
			key := keyFn(r)
			if key != "" && !lim.Allow(key) {
				http.Error(w, `{"error":"submission rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
