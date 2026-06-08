package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type ScoreUpdatedEvent struct {
	ContestantID string  `json:"contestant_id"`
	RunID        string  `json:"run_id"`
	Score        float64 `json:"score"`
}

type KafkaPublisher struct {
	writer *kafka.Writer
}

func New(brokers, topic string) *KafkaPublisher {
	w := &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}
	return &KafkaPublisher{writer: w}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event ScoreUpdatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal score.updated: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ContestantID),
		Value: data,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("publish score.updated: %w", err)
	}
	return nil
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
