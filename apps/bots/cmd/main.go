package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tradebench/bots/config"
	"github.com/tradebench/bots/internal/consumer"
	"github.com/tradebench/bots/internal/orchestrator"
	"github.com/tradebench/bots/internal/publisher"
)

func main() {
	cfg := config.Load()

	// Use orchestrator to manage bot workers and telemetry forwarding
	handler := func(ctx context.Context, event consumer.RunStartedEvent, telemetryCh chan<- consumer.TelemetryEvent) {
		orchestrator.Run(ctx, event, telemetryCh)
	}

	cons := consumer.New(consumer.Config{
		Brokers:       cfg.KafkaBrokers,
		Topic:         cfg.ConsumeTopic,
		GroupID:       cfg.ConsumerGroup,
		MaxConcurrent: cfg.MaxConcurrentRuns,
	}, handler)
	defer func() { _ = cons.Close() }()

	pub := publisher.New(publisher.Config{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.ProduceTopic,
	})
	defer func() { _ = pub.Close() }()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("bots: shutting down...")
		cancel()
		_ = srv.Shutdown(context.Background())
	}()

	go func() {
		if err := cons.Run(ctx, pub.Start); err != nil && ctx.Err() == nil {
			log.Fatalf("bots consumer error: %v", err)
		}
	}()

	log.Printf("bots service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("bots server error: %v", err)
	}
}
