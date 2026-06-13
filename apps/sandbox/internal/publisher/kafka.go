package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// tells downstream services (bots, scorer) that a sandbox is ready for load testing
type RunStartedEvent struct {
	RunID          string `json:"run_id"`
	SubmissionID   string `json:"submission_id"`
	ContestantID   string `json:"contestant_id"`
	SandboxAddress string `json:"sandbox_address"`
	TargetRPS      int    `json:"target_rps"`
	DurationSecs   int    `json:"duration_secs"`
	Protocol       string `json:"protocol"`
}

// tells downstream services that the sandbox failed to initialize
type SubmissionFailedEvent struct {
	SubmissionID string `json:"submission_id"`
	Error        string `json:"error"`
}

type KafkaPublisher struct {
	writer *kafka.Writer
}

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

func (p *KafkaPublisher) PublishFailed(ctx context.Context, topic string, event SubmissionFailedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka marshal: %w", err)
	}
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(event.SubmissionID),
		Value: data,
	}
	// Note: We create a specific message for the topic since the writer might be bound to the default ProduceTopic
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka publish failed event: %w", err)
	}
	return nil
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
