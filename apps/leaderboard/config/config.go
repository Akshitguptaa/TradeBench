package config

import "os"

type Config struct {
	Port string

	// Kafka
	KafkaBrokers  string
	ConsumeTopic  string // score.updated
	ConsumerGroup string

	// Redis
	RedisAddr string
	RedisKey  string // sorted set key for leaderboard
}

func Load() Config {
	return Config{
		Port:          envOr("LEADERBOARD_PORT", "8085"),
		KafkaBrokers:  envOr("KAFKA_BROKERS", "redpanda:9092"),
		ConsumeTopic:  envOr("LEADERBOARD_CONSUME_TOPIC", "score.updated"),
		ConsumerGroup: envOr("LEADERBOARD_CONSUMER_GROUP", "leaderboard-group"),
		RedisAddr:     envOr("REDIS_ADDR", "redis:6379"),
		RedisKey:      envOr("LEADERBOARD_REDIS_KEY", "leaderboard:top"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
