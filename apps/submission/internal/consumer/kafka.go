package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tradebench/submission/internal/status"
)

type Config struct {
	Brokers        string
	StartedTopic   string
	CompletedTopic string
	ScoreTopic     string
	GroupID        string
}

type Consumer struct {
	startedReader   *kafka.Reader
	completedReader *kafka.Reader
	scoreReader     *kafka.Reader
	tracker         *status.Tracker
}

type RunStartedEvent struct {
	RunID        string `json:"run_id"`
	SubmissionID string `json:"submission_id"`
}

type RunCompletedEvent struct {
	RunID string `json:"run_id"`
}

type ScoreUpdatedEvent struct {
	ContestantID string  `json:"contestant_id"`
	RunID        string  `json:"run_id"`
	Score        float64 `json:"score"`
}

func New(cfg Config, t *status.Tracker) *Consumer {
	brokers := strings.Split(cfg.Brokers, ",")
	
	startedReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          cfg.StartedTopic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	completedReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          cfg.CompletedTopic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	scoreReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          cfg.ScoreTopic,
		GroupID:        cfg.GroupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	return &Consumer{
		startedReader:   startedReader,
		completedReader: completedReader,
		scoreReader:     scoreReader,
		tracker:         t,
	}
}

func (c *Consumer) Run(ctx context.Context) {
	log.Println("submission consumer: starting run.started, run.completed, and score.updated listeners")

	go c.consumeStarted(ctx)
	go c.consumeCompleted(ctx)
	go c.consumeScore(ctx)

	<-ctx.Done()
}

func (c *Consumer) consumeStarted(ctx context.Context) {
	for {
		msg, err := c.startedReader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("submission consumer [started]: fetch error: %v", err)
			continue
		}

		var event RunStartedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("submission consumer [started]: bad message: %v", err)
			continue
		}

		log.Printf("submission consumer: processing run.started for submission_id=%s run_id=%s", event.SubmissionID, event.RunID)
		
		c.tracker.Set(event.SubmissionID, "", status.Running)
		c.tracker.SetRunID(event.SubmissionID, event.RunID)

		if err := c.startedReader.CommitMessages(ctx, msg); err != nil {
			log.Printf("submission consumer [started]: commit error: %v", err)
		}
	}
}

func (c *Consumer) consumeCompleted(ctx context.Context) {
	for {
		msg, err := c.completedReader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("submission consumer [completed]: fetch error: %v", err)
			continue
		}

		var event RunCompletedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("submission consumer [completed]: bad message: %v", err)
			continue
		}

		log.Printf("submission consumer: processing run.completed for run_id=%s", event.RunID)
		
		c.tracker.UpdateByRunID(event.RunID, status.Completed)

		if err := c.completedReader.CommitMessages(ctx, msg); err != nil {
			log.Printf("submission consumer [completed]: commit error: %v", err)
		}
	}
}

func (c *Consumer) consumeScore(ctx context.Context) {
	for {
		msg, err := c.scoreReader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("submission consumer [score]: fetch error: %v", err)
			continue
		}

		var event ScoreUpdatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("submission consumer [score]: bad message: %v", err)
			continue
		}

		log.Printf("submission consumer: processing score.updated for run_id=%s score=%v", event.RunID, event.Score)
		
		c.tracker.UpdateScoreByRunID(event.RunID, event.Score)

		if err := c.scoreReader.CommitMessages(ctx, msg); err != nil {
			log.Printf("submission consumer [score]: commit error: %v", err)
		}
	}
}

func (c *Consumer) Close() error {
	var errs []string
	if err := c.startedReader.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.completedReader.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := c.scoreReader.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
