package publisher

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tradebench/bots/internal/consumer"
)

type Config struct {
	Brokers string
	Topic   string
}

type BatchPublisher struct {
	writer *kafka.Writer
}

func New(cfg Config) *BatchPublisher {
	w := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(cfg.Brokers, ",")...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 100 * time.Millisecond,
		BatchSize:    500,
		RequiredAcks: kafka.RequireNone, // Fire and forget for raw telemetry
		Async:        true,              // Don't block workers on writes
	}
	return &BatchPublisher{writer: w}
}

// Start spawns a background goroutine that drains the telemetry channel and
// writes the messages to Kafka.
func (p *BatchPublisher) Start(ctx context.Context, telemetryCh <-chan consumer.TelemetryEvent) {
	go func() {
		for event := range telemetryCh {
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			msg := kafka.Message{
				Key:   []byte(event.RunID),
				Value: data,
			}

			if err := p.writer.WriteMessages(ctx, msg); err != nil {
				// Avoid noisy logs on shutdown context cancellation
				if ctx.Err() == nil {
					log.Printf("bots publisher: write error: %v", err)
				}
			}
		}
	}()
}

func (p *BatchPublisher) Close() error {
	return p.writer.Close()
}
