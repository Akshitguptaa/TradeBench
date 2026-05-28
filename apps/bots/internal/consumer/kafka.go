package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
)

// matches the JSON published by the sandbox service.
type RunStartedEvent struct {
	RunID          string `json:"run_id"`
	SubmissionID   string `json:"submission_id"`
	ContestantID   string `json:"contestant_id"`
	SandboxAddress string `json:"sandbox_address"`
	TargetRPS      int    `json:"target_rps"`
	DurationSecs   int    `json:"duration_secs"`
	Protocol       string `json:"protocol"`
}

// produced by the bot workers and sent to Kafka.
type TelemetryEvent struct {
	RunID       string `json:"run_id"`
	BotID       string `json:"bot_id"`
	OrderID     string `json:"order_id"`
	SentAtNs    int64  `json:"sent_at_ns"`
	AckAtNs     int64  `json:"ack_at_ns"`
	CorrectFill bool   `json:"correct_fill"`
	OrderType   string `json:"order_type"`
	Rejected    bool   `json:"rejected"`
}

// Handler is called for each RunStartedEvent. It receives a channel where
// telemetry events should be sent. The handler blocks until the run completes.
type Handler func(ctx context.Context, event RunStartedEvent, telemetryCh chan<- TelemetryEvent)

type Config struct {
	Brokers       string
	Topic         string
	GroupID       string
	MaxConcurrent int
}

type Consumer struct {
	reader  *kafka.Reader
	sem     chan struct{}
	handler Handler
}

func New(cfg Config, handler Handler) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(cfg.Brokers, ","),
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	return &Consumer{
		reader:  reader,
		sem:     make(chan struct{}, cfg.MaxConcurrent),
		handler: handler,
	}
}

	// Run blocks forever, reading run.started events and dispatching them to the handler.
func (c *Consumer) Run(ctx context.Context, publisherStartFunc func(context.Context, <-chan TelemetryEvent)) error {
	log.Println("bots consumer: starting consume loop")
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("bots consumer: fetch error: %v", err)
			continue
		}

		var event RunStartedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("bots consumer: unmarshal error: %v", err)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("bots consumer: received run_id=%s sandbox=%s rps=%d duration=%ds",
			event.RunID, event.SandboxAddress, event.TargetRPS, event.DurationSecs)

		c.sem <- struct{}{} // block if at max concurrency

		go func(m kafka.Message, evt RunStartedEvent) {
			defer func() { <-c.sem }()

			telemetryCh := make(chan TelemetryEvent, 1000)

			// Start the background batch publisher for this specific run
			publisherStartFunc(ctx, telemetryCh)

			// Execute the orchestrator (which sends orders and emits telemetry to the channel)
			c.handler(ctx, evt, telemetryCh)

			// Close channel to signal publisher to flush remaining and exit its goroutine
			close(telemetryCh)

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("bots consumer: commit error: %v", err)
			}
		}(msg, event)
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
