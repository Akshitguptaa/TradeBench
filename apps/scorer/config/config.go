package config

import "os"

type Config struct {
	Port string

	KafkaBrokers  string
	ConsumeTopic  string
	ProduceTopic  string
	ConsumerGroup string

	RedisAddr     string
	RedisPassword string
}

func Load() Config {
	return Config{
		Port:          envOr("SCORER_PORT", "8086"),
		KafkaBrokers:  envOr("KAFKA_BROKERS", "redpanda:9092"),
		ConsumeTopic:  envOr("SCORER_CONSUME_TOPIC", "run.completed"),
		ProduceTopic:  envOr("SCORER_PRODUCE_TOPIC", "score.updated"),
		ConsumerGroup: envOr("SCORER_CONSUMER_GROUP", "scorer-group"),
		RedisAddr:     envOr("REDIS_ADDR", "redis:6379"),
		RedisPassword: envOr("REDIS_PASSWORD", ""),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
