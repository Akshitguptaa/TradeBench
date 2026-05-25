// Package config loads submission-service configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all submission service configuration values.
type Config struct {
	Port string // HTTP listen port (default "8081")

	// MinIO
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	// Kafka
	KafkaBrokers string // comma-separated broker list
	KafkaTopic   string // topic name (default "submission.queued")

	// Upload limits
	MaxFileSizeMB int // max upload size in MB (default 50)
}

// Load reads configuration from the environment with sensible defaults.
func Load() Config {
	return Config{
		Port:          envOr("SUBMISSION_PORT", "8081"),
		MinioEndpoint: envOr("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: envOr("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: envOr("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:   envOr("MINIO_BUCKET", "tradebench"),
		MinioUseSSL:   envOrBool("MINIO_USE_SSL", false),
		KafkaBrokers:  envOr("KAFKA_BROKERS", "kafka:9092"),
		KafkaTopic:    envOr("KAFKA_TOPIC", "submission.queued"),
		MaxFileSizeMB: envOrInt("MAX_FILE_SIZE_MB", 50),
	}
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

func envOrBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
