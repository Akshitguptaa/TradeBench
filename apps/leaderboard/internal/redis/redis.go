package redisclient

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tradebench/leaderboard/internal/hub"
)

// Client wraps a Redis connection for leaderboard operations.
type Client struct {
	rdb *redis.Client
	key string
}

// New creates a new Redis client.
func New(addr, key string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	return &Client{rdb: rdb, key: key}
}

// Ping checks the Redis connection.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// UpdateScore sets the score and metrics for a contestant.
func (c *Client) UpdateScore(ctx context.Context, contestantID string, score, p50, p99 float64) error {
	pipe := c.rdb.TxPipeline()
	pipe.ZAdd(ctx, c.key, redis.Z{
		Score:  score,
		Member: contestantID,
	})
	metricsKey := c.key + ":metrics:" + contestantID
	pipe.HSet(ctx, metricsKey, "p50", p50, "p99", p99)
	_, err := pipe.Exec(ctx)
	return err
}

// Top50 fetches the top 50 contestants from the sorted set (highest score first).
func (c *Client) Top50(ctx context.Context) ([]hub.LeaderboardEntry, error) {
	results, err := c.rdb.ZRevRangeWithScores(ctx, c.key, 0, 49).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ZREVRANGE: %w", err)
	}

	entries := make([]hub.LeaderboardEntry, 0, len(results))
	pipe := c.rdb.Pipeline()
	var hashCmds []*redis.MapStringStringCmd
	for _, z := range results {
		contestantID := z.Member.(string)
		metricsKey := c.key + ":metrics:" + contestantID
		hashCmds = append(hashCmds, pipe.HGetAll(ctx, metricsKey))
	}
	if len(hashCmds) > 0 {
		_, _ = pipe.Exec(ctx)
	}

	for i, z := range results {
		contestantID := z.Member.(string)
		metrics := hashCmds[i].Val()
		p50, _ := strconv.ParseFloat(metrics["p50"], 64)
		p99, _ := strconv.ParseFloat(metrics["p99"], 64)

		entries = append(entries, hub.LeaderboardEntry{
			Rank:         i + 1,
			ContestantID: contestantID,
			Score:        z.Score,
			P50Ms:        p50,
			P99Ms:        p99,
		})
	}
	return entries, nil
}

// Close shuts down the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// WaitReady blocks until Redis is reachable or the context is cancelled.
func (c *Client) WaitReady(ctx context.Context) error {
	for {
		if err := c.Ping(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// SeedDemoData populates the sorted set with a few demo entries if it's empty.
// This is useful for initial testing when no real scores have been published yet.
func (c *Client) SeedDemoData(ctx context.Context) {
	count, err := c.rdb.ZCard(ctx, c.key).Result()
	if err != nil {
		log.Printf("leaderboard redis: seed check error: %v", err)
		return
	}
	if count > 0 {
		return // already has data
	}

	log.Println("leaderboard redis: seeding demo data")
	demoEntries := []redis.Z{
		{Score: 95.5, Member: "demo-contestant-alpha"},
		{Score: 89.2, Member: "demo-contestant-beta"},
		{Score: 82.7, Member: "demo-contestant-gamma"},
	}
	_ = c.rdb.ZAdd(ctx, c.key, demoEntries...).Err()
}

// ParseEntries converts a map of contestant -> score strings to LeaderboardEntries.
func ParseEntries(raw map[string]string) []hub.LeaderboardEntry {
	type entry struct {
		id    string
		score float64
	}
	var entries []entry
	for id, s := range raw {
		score, err := strconv.ParseFloat(s, 64)
		if err != nil {
			continue
		}
		entries = append(entries, entry{id: id, score: score})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].score > entries[j].score })

	result := make([]hub.LeaderboardEntry, 0, len(entries))
	for i, e := range entries {
		result = append(result, hub.LeaderboardEntry{
			Rank:         i + 1,
			ContestantID: e.id,
			Score:        e.score,
		})
	}
	return result
}

// IsConnRefused returns true if the error is a connection refused error.
func IsConnRefused(err error) bool {
	if err == nil {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Op == "dial"
	}
	return false
}
