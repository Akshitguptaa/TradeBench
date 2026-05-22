// Package config loads sandbox-service configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all sandbox service configuration values.
type Config struct {
	Port string // HTTP listen port for health endpoint (default "8082")

	// Kafka
	KafkaBrokers  string // comma-separated broker list
	ConsumeTopic  string // topic to consume (default "submission.queued")
	ProduceTopic  string // topic to publish (default "run.started")
	ConsumerGroup string // consumer group ID (default "sandbox-group")

	// MinIO
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	// Docker / Sandbox
	SandboxImage     string // base image (default "tradebench/sandbox-base:latest")
	SandboxNetwork   string // Docker network (default "sandbox-net")
	SandboxRuntime   string // OCI runtime (default "runsc")
	MaxConcurrent    int    // max concurrent sandboxes (default 4)
	HealthTimeoutSec int    // health-check timeout in seconds (default 30)

	// Resource limits
	CPUQuota    int64 // CPU quota (default 200000 = 2 vCPU)
	CPUPeriod   int64 // CPU period (default 100000)
	MemoryBytes int64 // hard memory limit (default 512MB)
	PidsLimit   int64 // max PIDs (default 256)

	// Sandbox run defaults
	DefaultTargetRPS   int    // default RPS for load test (default 1000)
	DefaultDurationSec int    // default duration in seconds (default 60)
	DefaultProtocol    string // default protocol (default "rest")
}

// Load reads configuration from the environment with sensible defaults.
func Load() Config {
	return Config{
		Port:          envOr("SANDBOX_PORT", "8082"),
		KafkaBrokers:  envOr("KAFKA_BROKERS", "kafka:9092"),
		ConsumeTopic:  envOr("SANDBOX_CONSUME_TOPIC", "submission.queued"),
		ProduceTopic:  envOr("SANDBOX_PRODUCE_TOPIC", "run.started"),
		ConsumerGroup: envOr("SANDBOX_CONSUMER_GROUP", "sandbox-group"),

		MinioEndpoint:  envOr("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: envOr("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: envOr("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    envOr("MINIO_BUCKET", "tradebench"),
		MinioUseSSL:    envOrBool("MINIO_USE_SSL", false),

		SandboxImage:     envOr("SANDBOX_IMAGE", "tradebench/sandbox-base:latest"),
		SandboxNetwork:   envOr("SANDBOX_NETWORK", "sandbox-net"),
		SandboxRuntime:   envOr("SANDBOX_RUNTIME", "runsc"),
		MaxConcurrent:    envOrInt("SANDBOX_MAX_CONCURRENT", 4),
		HealthTimeoutSec: envOrInt("SANDBOX_HEALTH_TIMEOUT_SEC", 30),

		CPUQuota:    envOrInt64("SANDBOX_CPU_QUOTA", 200000),
		CPUPeriod:   envOrInt64("SANDBOX_CPU_PERIOD", 100000),
		MemoryBytes: envOrInt64("SANDBOX_MEMORY_BYTES", 512*1024*1024),
		PidsLimit:   envOrInt64("SANDBOX_PIDS_LIMIT", 256),

		DefaultTargetRPS:   envOrInt("SANDBOX_DEFAULT_TARGET_RPS", 1000),
		DefaultDurationSec: envOrInt("SANDBOX_DEFAULT_DURATION_SEC", 60),
		DefaultProtocol:    envOr("SANDBOX_DEFAULT_PROTOCOL", "rest"),
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

func envOrInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
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
