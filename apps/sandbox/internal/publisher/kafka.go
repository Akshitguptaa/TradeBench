// Package publisher wraps a Kafka writer for publishing RunStartedEvent
// messages to the run.started topic.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// RunStartedEvent is the JSON payload published to the run.started topic.
type RunStartedEvent struct {
	RunID          string `json:"run_id"`
	SubmissionID   string `json:"submission_id"`
	ContestantID   string `json:"contestant_id"`
	SandboxAddress string `json:"sandbox_address"`
	TargetRPS      int    `json:"target_rps"`
	DurationSecs   int    `json:"duration_secs"`
	Protocol       string `json:"protocol"`
}

// KafkaPublisher publishes RunStartedEvent messages to Kafka.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher creates a Kafka producer for the given brokers and topic.
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

// Publish serialises the event as JSON and writes it to Kafka.
func (p *KafkaPublisher) Publish(ctx context.Context, event RunStartedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka marshal: %w", err)
	}
	msg := kafka.Message{
		Key:   []byte(event.RunID),
		Value: data,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

// Close flushes and closes the underlying Kafka writer.
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
