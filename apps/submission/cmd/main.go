package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tradebench/submission/config"
	"github.com/tradebench/submission/internal/handler"
	"github.com/tradebench/submission/internal/queue"
	"github.com/tradebench/submission/internal/store"
)

func main() {
	cfg := config.Load()

	// MinIO
	minioStore, err := store.NewMinIO(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Fatalf("minio init: %v", err)
	}

	// Kafka
	producer := queue.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer producer.Close()

	// Handler
	uploadHandler := handler.NewUploadHandler(minioStore, producer, cfg.MaxFileSizeMB)

	// Routes
	mux := http.NewServeMux()
	mux.Handle("/api/v1/submissions", uploadHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("submission service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
