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

	"github.com/tradebench/sandbox/config"
	"github.com/tradebench/sandbox/internal/consumer"
	"github.com/tradebench/sandbox/internal/publisher"
	"github.com/tradebench/sandbox/internal/puller"
	"github.com/tradebench/sandbox/internal/runner"
)

func main() {
	cfg := config.Load()

	// --- MinIO puller ---
	pull, err := puller.NewMinioPuller(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Fatalf("minio puller init: %v", err)
	}
	defer pull.Close()

	// --- Docker runner ---
	run, err := runner.New(runner.SandboxConfig{
		Image:      cfg.SandboxImage,
		Runtime:    cfg.SandboxRuntime,
		Network:    cfg.SandboxNetwork,
		CPUQuota:   cfg.CPUQuota,
		CPUPeriod:  cfg.CPUPeriod,
		Memory:     cfg.MemoryBytes,
		MemorySwap: cfg.MemoryBytes, // disable swap
		PidsLimit:  cfg.PidsLimit,
	})
	if err != nil {
		log.Fatalf("docker runner init: %v", err)
	}

	// --- Kafka publisher ---
	pub := publisher.NewKafkaPublisher(cfg.KafkaBrokers, cfg.ProduceTopic)
	defer pub.Close()

	// --- Kafka consumer ---
	cons := consumer.New(consumer.Config{
		Brokers:       cfg.KafkaBrokers,
		Topic:         cfg.ConsumeTopic,
		GroupID:       cfg.ConsumerGroup,
		MaxConcurrent: cfg.MaxConcurrent,
		TargetRPS:     cfg.DefaultTargetRPS,
		DurationSec:   cfg.DefaultDurationSec,
		Protocol:      cfg.DefaultProtocol,
	}, pull, run, pub)
	defer cons.Close()

	// --- Health endpoint ---
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// --- Graceful shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("sandbox: shutting down...")
		cancel()
		srv.Shutdown(context.Background())
	}()

	// Start consumer in background
	go func() {
		if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
			log.Fatalf("consumer error: %v", err)
		}
	}()

	log.Printf("sandbox service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
