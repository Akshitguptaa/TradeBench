package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/tradebench/leaderboard/internal/hub"
	redisclient "github.com/tradebench/leaderboard/internal/redis"
)

// ScoreUpdatedEvent matches the JSON published to score.updated by the scorer.
type ScoreUpdatedEvent struct {
	ContestantID string  `json:"contestant_id"`
	RunID        string  `json:"run_id"`
	Score        float64 `json:"score"`
	P50Ms        float64 `json:"p50_ms"`
	P99Ms        float64 `json:"p99_ms"`
}

// Config holds Kafka consumer configuration.
type Config struct {
	Brokers string
	Topic   string
	GroupID string
}

// Consumer reads score.updated events, updates Redis, and broadcasts to WebSocket clients.
type Consumer struct {
	reader *kafka.Reader
	hub    *hub.Hub
	redis  *redisclient.Client
}

// New creates a new score.updated consumer.
func New(cfg Config, h *hub.Hub, r *redisclient.Client) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(cfg.Brokers, ","),
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	return &Consumer{
		reader: reader,
		hub:    h,
		redis:  r,
	}
}

// Run starts the consume loop. It reads score.updated events, updates Redis,
// and broadcasts the update to all WebSocket clients.
func (c *Consumer) Run(ctx context.Context) error {
	log.Println("leaderboard consumer: starting consume loop")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("leaderboard consumer: fetch error: %v", err)
			continue
		}

		var event ScoreUpdatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("leaderboard consumer: unmarshal error: %v", err)
			continue
		}

		log.Printf("leaderboard consumer: score.updated contestant=%s score=%.2f",
			event.ContestantID, event.Score)

		// Update Redis sorted set
		if err := c.redis.UpdateScore(ctx, event.ContestantID, event.Score); err != nil {
			log.Printf("leaderboard consumer: redis update error: %v", err)
		}

		// Broadcast update to all WebSocket clients
		update := hub.Message{
			Type: "update",
			Payload: hub.LeaderboardEntry{
				ContestantID: event.ContestantID,
				Score:        event.Score,
				P50Ms:        event.P50Ms,
				P99Ms:        event.P99Ms,
			},
		}
		c.hub.BroadcastJSON(update)
	}
}

// Close shuts down the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
