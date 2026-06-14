package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/tradebench/sandbox/internal/publisher"
	"github.com/tradebench/sandbox/internal/puller"
	"github.com/tradebench/sandbox/internal/runner"
)

type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ContestantID string `json:"contestant_id"`
	S3Key        string `json:"s3_key"`
	Language     string `json:"language"`
	SubmittedAt  int64  `json:"submitted_at"`
}

// submission through the sandbox pipeline.
type Consumer struct {
	cfg       Config
	reader    *kafka.Reader
	puller    *puller.MinioPuller
	runner    *runner.Runner
	publisher *publisher.KafkaPublisher
	sem       chan struct{} // concurrency semaphore
}

type Config struct {
	Brokers       string
	Topic         string
	FailedTopic   string
	GroupID       string
	MaxConcurrent int
	TargetRPS     int
	DurationSec   int
	Protocol      string
}

func New(cfg Config, p *puller.MinioPuller, r *runner.Runner, pub *publisher.KafkaPublisher) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(cfg.Brokers, ","),
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		cfg:       cfg,
		reader:    reader,
		puller:    p,
		runner:    r,
		publisher: pub,
		sem:       make(chan struct{}, cfg.MaxConcurrent),
	}
}

// Run blocks forever, reading submission events and dispatching them to sandbox workers.
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
			_ = c.reader.CommitMessages(ctx, msg) // skip bad messages so we don't get stuck
			continue
		}

		log.Printf("consumer: received submission_id=%s", event.SubmissionID)

		c.sem <- struct{}{} // blocks if we're already at max concurrency

		go func(m kafka.Message, evt SubmissionEvent) {
			defer func() { <-c.sem }()

			if err := c.process(ctx, evt); err != nil {
				log.Printf("consumer: process error submission_id=%s: %v", evt.SubmissionID, err)
				
				// publish a failed event so downstream knows this submission will never run
				if pErr := c.publisher.PublishFailed(ctx, c.cfg.FailedTopic, publisher.SubmissionFailedEvent{
					SubmissionID: evt.SubmissionID,
					Error:        err.Error(),
				}); pErr != nil {
					log.Printf("consumer: error publishing failed event for %s: %v", evt.SubmissionID, pErr)
				}
			}

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("consumer: commit error: %v", err)
			}
		}(msg, event)
	}
}

// process handles a single submission: pull binary → spawn sandbox → tell the bots service it's ready
func (c *Consumer) process(ctx context.Context, event SubmissionEvent) error {
	binaryPath, err := c.puller.Pull(ctx, event.SubmissionID, event.S3Key)
	if err != nil {
		return fmt.Errorf("pull binary: %w", err)
	}
	defer c.puller.Cleanup(event.SubmissionID)

	containerID, address, err := c.runner.SpawnSandbox(ctx, event.SubmissionID, binaryPath, event.Language)
	if err != nil {
		return fmt.Errorf("spawn sandbox: %w", err)
	}

	log.Printf("consumer: sandbox spawned container=%s address=%s", containerID[:12], address)

	runID := uuid.New().String()
	runEvent := publisher.RunStartedEvent{
		RunID:          runID,
		SubmissionID:   event.SubmissionID,
		ContestantID:   event.ContestantID,
		SandboxAddress: address,
		TargetRPS:      c.cfg.TargetRPS,
		DurationSecs:   c.cfg.DurationSec,
		Protocol:       c.cfg.Protocol,
	}

	if err := c.publisher.Publish(ctx, runEvent); err != nil {
		return fmt.Errorf("publish run.started: %w", err)
	}

	log.Printf("consumer: published run.started run_id=%s submission_id=%s", runID, event.SubmissionID)

	// Terminate sandbox container after the run completes + 30s grace period
	go func(cid string, durSecs int) {
		time.Sleep(time.Duration(durSecs+30) * time.Second)
		_ = c.runner.TerminateSandbox(context.Background(), cid)
		log.Printf("consumer: sandbox cleanup complete for container=%s", cid[:12])
	}(containerID, c.cfg.DurationSec)

	return nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
