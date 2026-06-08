// Package config loads gateway configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all gateway configuration values, loaded from env vars.
type Config struct {
	Port string // HTTP listen port (default "8080")

	JWTSecret  string        // HMAC-SHA256 signing key
	JWTExpiry  time.Duration // token lifetime (default 24h)

	// Downstream service URLs
	SubmissionURL  string
	SandboxURL     string
	BotsURL        string
	IngesterURL    string
	ScorerURL      string
	LeaderboardURL string

	// Rate limiting
	GlobalRPM      int // requests per minute per contestant (default 100)
	SubmissionRPM   int // requests per minute for /api/v1/submissions (default 10)
}

// Load reads configuration from the environment with sensible defaults.
func Load() Config {
	c := Config{
		Port:           envOr("GATEWAY_PORT", "8080"),
		JWTSecret:      envOr("JWT_SECRET", "dev-secret-change-me"),
		SubmissionURL:  envOr("SUBMISSION_URL", "http://submission:8081"),
		SandboxURL:     envOr("SANDBOX_URL", "http://sandbox:8082"),
		BotsURL:        envOr("BOTS_URL", "http://bots:8083"),
		IngesterURL:    envOr("INGESTER_URL", "http://ingester:8084"),
		ScorerURL:      envOr("SCORER_URL", "http://scorer:8085"),
		LeaderboardURL: envOr("LEADERBOARD_URL", "http://leaderboard:8085"),
		GlobalRPM:      envOrInt("GLOBAL_RPM", 100),
		SubmissionRPM:  envOrInt("SUBMISSION_RPM", 10),
	}

	expiry := envOr("JWT_EXPIRY", "24h")
	dur, err := time.ParseDuration(expiry)
	if err != nil {
		dur = 24 * time.Hour
	}
	c.JWTExpiry = dur

	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
