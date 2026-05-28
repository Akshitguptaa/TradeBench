package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string

	// Kafka
	KafkaBrokers  string
	ConsumeTopic  string
	ProduceTopic  string
	ConsumerGroup string

	// Bot fleet defaults
	DefaultRPS         int
	DefaultDurationSec int
	MaxConcurrentRuns  int
}

func Load() Config {
	return Config{
		Port:          envOr("BOTS_PORT", "8083"),
		KafkaBrokers:  envOr("KAFKA_BROKERS", "kafka:9092"),
		ConsumeTopic:  envOr("BOTS_CONSUME_TOPIC", "run.started"),
		ProduceTopic:  envOr("BOTS_PRODUCE_TOPIC", "telemetry.raw"),
		ConsumerGroup: envOr("BOTS_CONSUMER_GROUP", "bots-group"),

		DefaultRPS:         envOrInt("BOT_DEFAULT_RPS", 1000),
		DefaultDurationSec: envOrInt("BOT_DEFAULT_DURATION_SECS", 60),
		MaxConcurrentRuns:  envOrInt("BOTS_MAX_CONCURRENT_RUNS", 4),
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
