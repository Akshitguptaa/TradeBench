package hub

import (
	"encoding/json"
	"log"
	"sync"
)

// LeaderboardEntry represents a single row on the leaderboard.
type LeaderboardEntry struct {
	Rank         int     `json:"rank"`
	ContestantID string  `json:"contestant_id"`
	Score        float64 `json:"score"`
	P50Ms        float64 `json:"p50_ms,omitempty"`
	P99Ms        float64 `json:"p99_ms,omitempty"`
}

// Message is the JSON envelope sent over the WebSocket.
type Message struct {
	Type    string      `json:"type"`    // "snapshot" or "update"
	Payload interface{} `json:"payload"`
}

// Client represents a single WebSocket client.
type Client struct {
	Send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan []byte
}

// New creates a new Hub.
func New() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub event loop. It should be launched in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("leaderboard hub: client connected (%d total)", h.ClientCount())

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("leaderboard hub: client disconnected (%d total)", h.ClientCount())

		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client is slow; drop it.
					go func(c *Client) {
						h.Unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastJSON marshals v and sends it to all connected clients.
func (h *Hub) BroadcastJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("leaderboard hub: marshal error: %v", err)
		return
	}
	h.Broadcast <- data
}
