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

	"github.com/gorilla/websocket"

	"github.com/tradebench/leaderboard/config"
	"github.com/tradebench/leaderboard/internal/consumer"
	"github.com/tradebench/leaderboard/internal/hub"
	redisclient "github.com/tradebench/leaderboard/internal/redis"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redis
	rdb := redisclient.New(cfg.RedisAddr, cfg.RedisKey)
	defer rdb.Close()

	if err := rdb.WaitReady(ctx); err != nil {
		log.Fatalf("redis not ready: %v", err)
	}
	log.Println("leaderboard: redis connected")

	// Seed demo data for testing
	rdb.SeedDemoData(ctx)

	// WebSocket hub
	h := hub.New()
	go h.Run()

	// Kafka consumer
	cons := consumer.New(consumer.Config{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.ConsumeTopic,
		GroupID: cfg.ConsumerGroup,
	}, h, rdb)
	defer cons.Close()

	// HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/ws/leaderboard", wsHandler(h, rdb))

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
		log.Println("leaderboard: shutting down...")
		cancel()
		_ = srv.Shutdown(context.Background())
	}()

	// Consumer loop runs in background
	go func() {
		if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
			log.Fatalf("leaderboard consumer error: %v", err)
		}
	}()

	log.Printf("leaderboard service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("leaderboard server error: %v", err)
	}
}

// wsHandler returns an HTTP handler that upgrades connections to WebSocket and
// sends the initial top-50 snapshot from Redis.
func wsHandler(h *hub.Hub, rdb *redisclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("leaderboard ws: upgrade error: %v", err)
			return
		}

		client := &hub.Client{Send: make(chan []byte, 256)}
		h.Register <- client

		// Send initial snapshot
		entries, err := rdb.Top50(r.Context())
		if err != nil {
			log.Printf("leaderboard ws: redis top50 error: %v", err)
		} else {
			snapshot := hub.Message{
				Type:    "snapshot",
				Payload: entries,
			}
			data, _ := json.Marshal(snapshot)
			_ = conn.WriteMessage(websocket.TextMessage, data)
		}

		// Writer goroutine: sends broadcast messages to this client
		go func() {
			defer conn.Close()
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		// Reader goroutine: drains incoming messages (keep-alive / close detection)
		go func() {
			defer func() { h.Unregister <- client }()
			conn.SetReadLimit(512)
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			conn.SetPongHandler(func(string) error {
				_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				return nil
			})
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	}
}
