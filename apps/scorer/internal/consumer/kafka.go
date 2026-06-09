package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/tradebench/scorer/internal/leaderboard"
	"github.com/tradebench/scorer/internal/publisher"
	"github.com/tradebench/scorer/internal/scoring"
)

type RunCompletedEvent struct {
	RunID        string  `json:"run_id"`
	ContestantID string  `json:"contestant_id"`
	P50Ms        float64 `json:"p50_ms"`
	P90Ms        float64 `json:"p90_ms"`
	P99Ms        float64 `json:"p99_ms"`
	MaxTPS       int64   `json:"max_tps"`
	Correctness  float64 `json:"correctness"`
}

type Config struct {
	Brokers string
	Topic   string
	GroupID string
}

type Consumer struct {
	reader *kafka.Reader
	lb     *leaderboard.RedisLeaderboard
	pub    *publisher.KafkaPublisher
}

func New(cfg Config, lb *leaderboard.RedisLeaderboard, pub *publisher.KafkaPublisher) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(cfg.Brokers, ","),
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	return &Consumer{
		reader: reader,
		lb:     lb,
		pub:    pub,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	log.Println("scorer consumer: listening for run.completed events")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("scorer consumer: fetch error: %v", err)
			continue
		}

		var event RunCompletedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("scorer consumer: bad message: %v", err)
			continue
		}

		score := scoring.Calculate(event.P99Ms, event.MaxTPS, event.Correctness)

		log.Printf("scorer: run_id=%s contestant=%s score=%.2f (p99=%.1fms tps=%d correctness=%.2f)",
			event.RunID, event.ContestantID, score, event.P99Ms, event.MaxTPS, event.Correctness)

		if err := c.lb.Upsert(ctx, event.ContestantID, score); err != nil {
			log.Printf("scorer: redis upsert failed: %v", err)
		}

		scoreEvent := publisher.ScoreUpdatedEvent{
			ContestantID: event.ContestantID,
			RunID:        event.RunID,
			Score:        score,
			P50Ms:        event.P50Ms,
			P99Ms:        event.P99Ms,
		}
		if err := c.pub.Publish(ctx, scoreEvent); err != nil {
			log.Printf("scorer: publish score.updated failed: %v", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
