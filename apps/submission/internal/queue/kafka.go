// Package queue wraps the segmentio/kafka-go writer for publishing
// submission events to Kafka.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// SubmissionEvent is the JSON payload published to the submission.queued topic.
type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ContestantID string `json:"contestant_id"`
	S3Key        string `json:"s3_key"`
	Language     string `json:"language"`
	SubmittedAt  int64  `json:"submitted_at"` // unix millis
}

// KafkaProducer is a thin wrapper around kafka.Writer.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka producer for the given brokers and topic.
func NewKafkaProducer(brokers, topic string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond, // low latency for dev
		RequiredAcks: kafka.RequireOne,
	}
	return &KafkaProducer{writer: w}
}

// Publish serialises the event as JSON and writes it to Kafka. The message
// key is set to the submission_id for partition affinity.
func (p *KafkaProducer) Publish(ctx context.Context, event SubmissionEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka marshal: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.SubmissionID),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

// Close flushes and closes the underlying Kafka writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
