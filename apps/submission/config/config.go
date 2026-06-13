package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string

	// MinIO
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	// Kafka configuration
	KafkaBrokers string
	KafkaTopic   string
	FailedTopic  string

	MaxFileSizeMB int //default 50
}

func Load() Config {
	return Config{
		Port:           envOr("SUBMISSION_PORT", "8081"),
		MinioEndpoint:  envOr("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: envOr("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: envOr("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    envOr("MINIO_BUCKET", "tradebench"),
		MinioUseSSL:    envOrBool("MINIO_USE_SSL", false),
		KafkaBrokers:   envOr("KAFKA_BROKERS", "kafka:9092"),
		KafkaTopic:     envOr("KAFKA_TOPIC", "submission.queued"),
		FailedTopic:    envOr("KAFKA_FAILED_TOPIC", "submission.failed"),
		MaxFileSizeMB:  envOrInt("MAX_FILE_SIZE_MB", 50),
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
