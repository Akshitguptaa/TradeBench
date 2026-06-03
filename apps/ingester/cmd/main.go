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

	"github.com/tradebench/ingester/config"
	"github.com/tradebench/ingester/internal/consumer"
	"github.com/tradebench/ingester/internal/db"
	"github.com/tradebench/ingester/internal/publisher"
	"github.com/tradebench/ingester/internal/window"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TimescaleDB
	writer, err := db.NewWriter(ctx, cfg.TimescaleDSN)
	if err != nil {
		log.Fatalf("timescaledb init: %v", err)
	}
	defer writer.Close()

	// Window manager
	mgr := window.NewManager(cfg.DefaultRunDurationSec)

	// Kafka publisher for run.completed
	pub := publisher.NewKafkaPublisher(cfg.KafkaBrokers, cfg.ProduceTopic)
	defer pub.Close()

	// Kafka consumer for telemetry.raw
	cons := consumer.New(consumer.Config{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.ConsumeTopic,
		GroupID: cfg.ConsumerGroup,
	}, mgr, writer, pub)
	defer cons.Close()

	// Health server
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

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("ingester: shutting down...")
		cancel()
		_ = srv.Shutdown(context.Background())
	}()

	// Consumer loop runs in background
	go func() {
		if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
			log.Fatalf("ingester consumer error: %v", err)
		}
	}()

	log.Printf("ingester service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("ingester server error: %v", err)
	}
}
