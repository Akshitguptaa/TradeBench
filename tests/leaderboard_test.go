package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupTestContestant removes a test contestant from both Redis leaderboard keys.
func cleanupTestContestant(t *testing.T, contestantID string) {
	t.Helper()
	for _, key := range []string{"leaderboard:top", "tradebench:leaderboard"} {
		cmd := exec.Command("docker", "compose", "exec", "-T", "redis", "redis-cli", "ZREM", key, contestantID)
		cmd.Dir = os.Getenv("TRADEBENCH_ROOT")
		if cmd.Dir == "" {
			cmd.Dir = ".."
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("cleanup ZREM %s %s: %v (%s)", key, contestantID, err, string(out))
		}
	}
	// Also remove the metrics hash
	cmd := exec.Command("docker", "compose", "exec", "-T", "redis", "redis-cli", "DEL", "leaderboard:metrics:"+contestantID)
	cmd.Dir = os.Getenv("TRADEBENCH_ROOT")
	if cmd.Dir == "" {
		cmd.Dir = ".."
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("cleanup DEL metrics %s: %v (%s)", contestantID, err, string(out))
	}
}

type LeaderboardEntry struct {
	Rank         int     `json:"rank"`
	ContestantID string  `json:"contestant_id"`
	Score        float64 `json:"score"`
	P50Ms        float64 `json:"p50_ms,omitempty"`
	P99Ms        float64 `json:"p99_ms,omitempty"`
}

type Message struct {
	Type    string             `json:"type"`
	Payload interface{}        `json:"payload"`
}

type ScoreUpdatedEvent struct {
	ContestantID string  `json:"contestant_id"`
	RunID        string  `json:"run_id"`
	Score        float64 `json:"score"`
	P50Ms        float64 `json:"p50_ms"`
	P99Ms        float64 `json:"p99_ms"`
}

func TestLeaderboardWebSocketIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	wsURL := os.Getenv("LEADERBOARD_WS_URL")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8085/ws/leaderboard"
	}

	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		broker = "127.0.0.1:9092"
	}
	brokers := []string{broker}

	// Connect to WebSocket
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err, "failed to connect to leaderboard websocket")
	defer c.Close()

	// Read initial snapshot
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := c.ReadMessage()
	require.NoError(t, err, "failed to read snapshot message")

	var snapshot Message
	err = json.Unmarshal(msg, &snapshot)
	require.NoError(t, err, "failed to unmarshal snapshot JSON")
	assert.Equal(t, "snapshot", snapshot.Type, "expected first message to be snapshot")

	// We don't assert the exact payload here because it depends on the seed data or previous tests,
	// but we just want to ensure it's an array.
	payloadBytes, err := json.Marshal(snapshot.Payload)
	require.NoError(t, err)
	var entries []LeaderboardEntry
	err = json.Unmarshal(payloadBytes, &entries)
	require.NoError(t, err, "snapshot payload should be a list of entries")

	// Now publish an update to Kafka
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    "score.updated",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	contestantID := fmt.Sprintf("test-contestant-%d", time.Now().Unix())
	t.Cleanup(func() { cleanupTestContestant(t, contestantID) })

	scoreUpdate := ScoreUpdatedEvent{
		ContestantID: contestantID,
		RunID:        "run-test-id",
		Score:        99.9,
		P50Ms:        1.2,
		P99Ms:        5.3,
	}

	b, err := json.Marshal(scoreUpdate)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(contestantID),
		Value: b,
	})
	require.NoError(t, err, "failed to produce score.updated event")

	// Wait for the websocket to receive the broadcasted update
	updateReceived := false
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		var update Message
		if err := json.Unmarshal(msg, &update); err != nil {
			continue
		}

		if update.Type == "update" {
			payloadBytes, err := json.Marshal(update.Payload)
			if err != nil {
				continue
			}
			var entry LeaderboardEntry
			if err := json.Unmarshal(payloadBytes, &entry); err != nil {
				continue
			}
			
			if entry.ContestantID == contestantID {
				assert.Equal(t, 99.9, entry.Score)
				assert.Equal(t, 1.2, entry.P50Ms)
				assert.Equal(t, 5.3, entry.P99Ms)
				updateReceived = true
				break
			}
		}
	}

	assert.True(t, updateReceived, "expected to receive live update over websocket")
}
