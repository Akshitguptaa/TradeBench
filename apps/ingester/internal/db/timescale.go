package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tradebench/ingester/internal/window"
)

// Writer inserts run metric snapshots into the TimescaleDB run_metrics hypertable.
type Writer struct {
	pool *pgxpool.Pool
}

// NewWriter creates a connection pool to TimescaleDB with retry logic.
func NewWriter(ctx context.Context, dsn string) (*Writer, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	const maxRetries = 10
	const retryDelay = 2 * time.Second

	var pool *pgxpool.Pool
	for attempt := 1; attempt <= maxRetries; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			log.Printf("db: connection attempt %d/%d failed: %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("connect to timescaledb after %d attempts: %w", maxRetries, err)
		}

		// verify connectivity
		if err = pool.Ping(ctx); err != nil {
			pool.Close()
			log.Printf("db: ping attempt %d/%d failed: %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, fmt.Errorf("ping timescaledb after %d attempts: %w", maxRetries, err)
		}

		break
	}

	log.Println("db: connected to TimescaleDB")
	return &Writer{pool: pool}, nil
}

// Insert writes a completed run snapshot into the run_metrics hypertable.
func (w *Writer) Insert(ctx context.Context, snap window.RunSnapshot) error {
	const query = `
		INSERT INTO run_metrics (time, run_id, contestant_id, p50_ms, p90_ms, p99_ms, max_tps, correctness)
		VALUES (NOW(), $1, $2, $3, $4, $5, $6, $7)
	`

	_, err := w.pool.Exec(ctx, query,
		snap.RunID,
		snap.ContestantID,
		snap.P50Ms,
		snap.P90Ms,
		snap.P99Ms,
		snap.MaxTPS,
		snap.Correctness,
	)
	if err != nil {
		return fmt.Errorf("insert run_metrics run_id=%s: %w", snap.RunID, err)
	}

	log.Printf("db: inserted metrics for run_id=%s p50=%.2fms p99=%.2fms tps=%d correctness=%.2f",
		snap.RunID, snap.P50Ms, snap.P99Ms, snap.MaxTPS, snap.Correctness)
	return nil
}

// Close shuts down the connection pool.
func (w *Writer) Close() {
	w.pool.Close()
}
