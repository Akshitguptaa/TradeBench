package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// RunCompletedEvent is published to run.completed for the scorer service.
type RunCompletedEvent struct {
	RunID        string  `json:"run_id"`
	ContestantID string  `json:"contestant_id"`
	P50Ms        float64 `json:"p50_ms"`
	P90Ms        float64 `json:"p90_ms"`
	P99Ms        float64 `json:"p99_ms"`
	MaxTPS       int64   `json:"max_tps"`
	Correctness  float64 `json:"correctness"`
}

// KafkaPublisher publishes RunCompletedEvents to the run.completed topic.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher creates a publisher for run.completed events.
func NewKafkaPublisher(brokers, topic string) *KafkaPublisher {
	w := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}
	return &KafkaPublisher{writer: w}
}

// Publish sends a RunCompletedEvent to Kafka.
func (p *KafkaPublisher) Publish(ctx context.Context, event RunCompletedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal RunCompletedEvent: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.RunID),
		Value: data,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("publish run.completed: %w", err)
	}
	return nil
}

// Close flushes and closes the Kafka writer.
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
