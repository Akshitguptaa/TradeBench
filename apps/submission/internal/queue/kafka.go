package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// SubmissionEvent represents a new code submission queued for sandbox execution
type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ContestantID string `json:"contestant_id"`
	S3Key        string `json:"s3_key"`
	Language     string `json:"language"`
	SubmittedAt  int64  `json:"submitted_at"`
}

type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a producer that routes messages to the specified topic
func NewKafkaProducer(brokers, topic string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}
	return &KafkaProducer{writer: w}
}

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

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
