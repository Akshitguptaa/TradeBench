package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string

	// Kafka
	KafkaBrokers  string
	ConsumeTopic  string // telemetry.raw
	ProduceTopic  string // run.completed
	ConsumerGroup string

	// TimescaleDB
	TimescaleDSN string

	// Window
	DefaultRunDurationSec int
}

func Load() Config {
	return Config{
		Port:          envOr("INGESTER_PORT", "8084"),
		KafkaBrokers:  envOr("KAFKA_BROKERS", "redpanda:9092"),
		ConsumeTopic:  envOr("INGESTER_CONSUME_TOPIC", "telemetry.raw"),
		ProduceTopic:  envOr("INGESTER_PRODUCE_TOPIC", "run.completed"),
		ConsumerGroup: envOr("INGESTER_CONSUMER_GROUP", "ingester-group"),
		TimescaleDSN:  envOr("TIMESCALE_DSN", "postgres://tradebench:tradebench@timescaledb:5432/tradebench?sslmode=disable"),

		DefaultRunDurationSec: envOrInt("INGESTER_DEFAULT_RUN_DURATION_SEC", 60),
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
