#!/usr/bin/env bash
set -euo pipefail

echo "==> Running Database Migrations..."

# Wait for database to be ready
until docker compose exec -T timescaledb pg_isready -U tradebench; do
  echo "Waiting for postgres..."
  sleep 2
done

# Run migrations
docker compose exec -T timescaledb psql -U tradebench -d tradebench << 'EOF'

-- 001_create_run_metrics.sql
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

CREATE TABLE IF NOT EXISTS run_metrics (
  time          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
  run_id        TEXT            NOT NULL,
  contestant_id TEXT            NOT NULL,
  p50_ms        DOUBLE PRECISION,
  p90_ms        DOUBLE PRECISION,
  p99_ms        DOUBLE PRECISION,
  max_tps       BIGINT,
  correctness   DOUBLE PRECISION,
  composite     DOUBLE PRECISION
);

SELECT create_hypertable('run_metrics', 'time', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_run_metrics_contestant
  ON run_metrics (contestant_id, time DESC);

-- 002_create_submissions.sql
CREATE TABLE IF NOT EXISTS submissions (
  submission_id TEXT PRIMARY KEY,
  contestant_id TEXT        NOT NULL,
  s3_key        TEXT        NOT NULL,
  language      TEXT        NOT NULL,
  status        TEXT        NOT NULL DEFAULT 'queued',
  hash          TEXT        NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

EOF

echo "==> Migrations complete."
