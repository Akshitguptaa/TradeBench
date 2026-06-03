package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/tradebench/ingester/internal/db"
	"github.com/tradebench/ingester/internal/publisher"
	"github.com/tradebench/ingester/internal/window"
)

// Config holds Kafka consumer configuration.
type Config struct {
	Brokers string
	Topic   string
	GroupID string
}

// Consumer reads telemetry.raw events at high throughput, feeds them into the
// window manager, and periodically flushes completed runs to TimescaleDB + Kafka.
type Consumer struct {
	reader    *kafka.Reader
	manager   *window.Manager
	writer    *db.Writer
	publisher *publisher.KafkaPublisher
}

// New creates a new telemetry consumer.
func New(cfg Config, mgr *window.Manager, w *db.Writer, pub *publisher.KafkaPublisher) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(cfg.Brokers, ","),
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	return &Consumer{
		reader:    reader,
		manager:   mgr,
		writer:    w,
		publisher: pub,
	}
}

// Run starts the consume loop. It reads telemetry events, feeds them into the
// window manager, and checks for completed runs every second.
func (c *Consumer) Run(ctx context.Context) error {
	log.Println("ingester consumer: starting consume loop")

	// Background ticker to check for completed runs
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.flushCompleted(ctx)
			}
		}
	}()

	// Main consume loop
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("ingester consumer: fetch error: %v", err)
			continue
		}

		var event window.TelemetryEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("ingester consumer: unmarshal error: %v", err)
			continue
		}

		c.manager.AddEvent(event)
	}
}

// flushCompleted checks for completed run windows, writes metrics to TimescaleDB,
// and publishes run.completed events downstream.
func (c *Consumer) flushCompleted(ctx context.Context) {
	completed := c.manager.CheckCompleted()
	for _, snap := range completed {
		log.Printf("ingester: run completed run_id=%s p50=%.2fms p90=%.2fms p99=%.2fms tps=%d correctness=%.2f",
			snap.RunID, snap.P50Ms, snap.P90Ms, snap.P99Ms, snap.MaxTPS, snap.Correctness)

		// Write to TimescaleDB
		if err := c.writer.Insert(ctx, snap); err != nil {
			log.Printf("ingester: db insert error run_id=%s: %v", snap.RunID, err)
		}

		// Publish run.completed for the scorer
		event := publisher.RunCompletedEvent{
			RunID:        snap.RunID,
			ContestantID: snap.ContestantID,
			P50Ms:        snap.P50Ms,
			P90Ms:        snap.P90Ms,
			P99Ms:        snap.P99Ms,
			MaxTPS:       snap.MaxTPS,
			Correctness:  snap.Correctness,
		}
		if err := c.publisher.Publish(ctx, event); err != nil {
			log.Printf("ingester: publish error run_id=%s: %v", snap.RunID, err)
		}
	}
}

// Close shuts down the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
