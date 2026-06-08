package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tradebench/scorer/config"
	"github.com/tradebench/scorer/internal/consumer"
	"github.com/tradebench/scorer/internal/leaderboard"
	"github.com/tradebench/scorer/internal/publisher"
)

func main() {
	cfg := config.Load()

	lb := leaderboard.New(cfg.RedisAddr, cfg.RedisPassword)
	defer lb.Close()

	pub := publisher.New(cfg.KafkaBrokers, cfg.ProduceTopic)
	defer pub.Close()

	cons := consumer.New(consumer.Config{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.ConsumeTopic,
		GroupID: cfg.ConsumerGroup,
	}, lb, pub)
	defer cons.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("scorer: shutting down")
		cancel()
	}()

	go func() {
		if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
			log.Fatalf("scorer consumer error: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("scorer: listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("scorer http: %v", err)
	}
}
