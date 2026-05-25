// Package consumer implements a Kafka consumer for the submission.queued topic.
// On each message it orchestrates: pull binary → spawn sandbox → publish run.started.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/tradebench/sandbox/internal/publisher"
	"github.com/tradebench/sandbox/internal/puller"
	"github.com/tradebench/sandbox/internal/runner"
)

// SubmissionEvent mirrors the JSON payload from the submission service.
type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ContestantID string `json:"contestant_id"`
	S3Key        string `json:"s3_key"`
	Language     string `json:"language"`
	SubmittedAt  int64  `json:"submitted_at"`
}

// Consumer reads from the submission.queued Kafka topic and processes each
// submission through the sandbox pipeline.
type Consumer struct {
	reader    *kafka.Reader
	puller    *puller.MinioPuller
	runner    *runner.Runner
	publisher *publisher.KafkaPublisher
	sem       chan struct{} // concurrency semaphore

	// Run defaults
	defaultTargetRPS   int
	defaultDurationSec int
	defaultProtocol    string
}

// Config bundles the parameters needed to create a Consumer.
type Config struct {
	Brokers        string
	Topic          string
	GroupID        string
	MaxConcurrent  int
	TargetRPS      int
	DurationSec    int
	Protocol       string
}

// New creates a Consumer wired to Kafka, MinIO puller, Docker runner, and
// the run.started publisher.
func New(cfg Config, p *puller.MinioPuller, r *runner.Runner, pub *publisher.KafkaPublisher) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(cfg.Brokers, ","),
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader:             reader,
		puller:             p,
		runner:             r,
		publisher:          pub,
		sem:                make(chan struct{}, cfg.MaxConcurrent),
		defaultTargetRPS:   cfg.TargetRPS,
		defaultDurationSec: cfg.DurationSec,
		defaultProtocol:    cfg.Protocol,
	}
}

// Run starts the consume loop. It blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	log.Println("consumer: starting consume loop")
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("consumer: fetch error: %v", err)
			continue
		}

		var event SubmissionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("consumer: unmarshal error: %v", err)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("consumer: received submission_id=%s", event.SubmissionID)

		// Acquire semaphore slot (blocks if at max concurrency)
		c.sem <- struct{}{}

		go func(m kafka.Message, evt SubmissionEvent) {
			defer func() { <-c.sem }()

			if err := c.process(ctx, evt); err != nil {
				log.Printf("consumer: process error submission_id=%s: %v", evt.SubmissionID, err)
			}

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("consumer: commit error: %v", err)
			}
		}(msg, event)
	}
}

// process handles a single submission: pull → spawn → publish.
func (c *Consumer) process(ctx context.Context, event SubmissionEvent) error {
	// 1. Pull binary from MinIO
	binaryPath, err := c.puller.Pull(ctx, event.SubmissionID, event.S3Key)
	if err != nil {
		return fmt.Errorf("pull binary: %w", err)
	}
	defer c.puller.Cleanup(event.SubmissionID)

	// 2. Spawn sandbox container
	containerID, address, err := c.runner.SpawnSandbox(ctx, event.SubmissionID, binaryPath)
	if err != nil {
		return fmt.Errorf("spawn sandbox: %w", err)
	}

	log.Printf("consumer: sandbox spawned container=%s address=%s", containerID[:12], address)

	// 3. Publish RunStartedEvent
	runID := uuid.New().String()
	runEvent := publisher.RunStartedEvent{
		RunID:          runID,
		SubmissionID:   event.SubmissionID,
		ContestantID:   event.ContestantID,
		SandboxAddress: address,
		TargetRPS:      c.defaultTargetRPS,
		DurationSecs:   c.defaultDurationSec,
		Protocol:       c.defaultProtocol,
	}

	if err := c.publisher.Publish(ctx, runEvent); err != nil {
		// Sandbox is running but event failed — log but don't terminate
		return fmt.Errorf("publish run.started: %w", err)
	}

	log.Printf("consumer: published run.started run_id=%s submission_id=%s", runID, event.SubmissionID)
	return nil
}

// Close shuts down the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
