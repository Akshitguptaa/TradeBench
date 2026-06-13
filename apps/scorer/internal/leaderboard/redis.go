package leaderboard

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const leaderboardKey = "tradebench:leaderboard"

type Entry struct {
	ContestantID string  `json:"contestant_id"`
	Score        float64 `json:"score"`
	Rank         int64   `json:"rank"`
}

type RedisLeaderboard struct {
	client *redis.Client
}

func New(addr, password string) *RedisLeaderboard {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &RedisLeaderboard{client: rdb}
}

// Upsert adds or updates a contestant's score. Only keeps the best score.
func (lb *RedisLeaderboard) Upsert(ctx context.Context, contestantID string, score, p50Ms, p99Ms float64) error {
	// ZADD with GT flag — only update if new score is greater than existing
	cmd := lb.client.ZAddGT(ctx, leaderboardKey, redis.Z{
		Score:  score,
		Member: contestantID,
	})
	if cmd.Err() != nil {
		return fmt.Errorf("leaderboard upsert: %w", cmd.Err())
	}

	if cmd.Val() > 0 {
		// The score was updated (or added), so update the metrics hash as well
		metricsKey := fmt.Sprintf("tradebench:metrics:%s", contestantID)
		if err := lb.client.HSet(ctx, metricsKey, map[string]interface{}{
			"p50_ms": p50Ms,
			"p99_ms": p99Ms,
		}).Err(); err != nil {
			return fmt.Errorf("metrics hash upsert: %w", err)
		}
	}
	return nil
}

// TopN returns the top N contestants, highest score first.
func (lb *RedisLeaderboard) TopN(ctx context.Context, n int64) ([]Entry, error) {
	results, err := lb.client.ZRevRangeWithScores(ctx, leaderboardKey, 0, n-1).Result()
	if err != nil {
		return nil, fmt.Errorf("leaderboard top-n: %w", err)
	}

	entries := make([]Entry, len(results))
	for i, z := range results {
		entries[i] = Entry{
			ContestantID: z.Member.(string),
			Score:        z.Score,
			Rank:         int64(i + 1),
		}
	}
	return entries, nil
}

// GetRank returns a contestant's current rank and score. Rank is 1-indexed.
func (lb *RedisLeaderboard) GetRank(ctx context.Context, contestantID string) (Entry, error) {
	rank, err := lb.client.ZRevRank(ctx, leaderboardKey, contestantID).Result()
	if err != nil {
		return Entry{}, fmt.Errorf("leaderboard get rank: %w", err)
	}

	score, err := lb.client.ZScore(ctx, leaderboardKey, contestantID).Result()
	if err != nil {
		return Entry{}, fmt.Errorf("leaderboard get score: %w", err)
	}

	return Entry{
		ContestantID: contestantID,
		Score:        score,
		Rank:         rank + 1,
	}, nil
}

func (lb *RedisLeaderboard) Close() error {
	return lb.client.Close()
}
