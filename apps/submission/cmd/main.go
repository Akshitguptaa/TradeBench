package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tradebench/submission/config"
	"github.com/tradebench/submission/internal/handler"
	"github.com/tradebench/submission/internal/queue"
	"github.com/tradebench/submission/internal/status"
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
	tracker := status.NewTracker()
	uploadHandler := handler.NewUploadHandler(minioStore, producer, tracker, cfg.MaxFileSizeMB)

	// Routes
	mux := http.NewServeMux()

	// Upload handler
	mux.Handle("/api/v1/submissions", uploadHandler)

	// Get submission status
	mux.HandleFunc("/api/v1/submissions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// extract the submission ID from the URL path
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/submissions/")
		if id == "" {
			http.Error(w, `{"error":"submission_id is required"}`, http.StatusBadRequest)
			return
		}

		entry := tracker.Get(id)
		if entry == nil {
			http.Error(w, `{"error":"submission not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entry)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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
