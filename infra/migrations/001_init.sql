-- TradeBench Schema (TimescaleDB)

-- Submissions
CREATE TABLE IF NOT EXISTS submissions (
    submission_id TEXT PRIMARY KEY,
    contestant_id TEXT NOT NULL,
    s3_key        TEXT NOT NULL,
    language      TEXT NOT NULL,
    sha256_hash   TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'queued',
    run_id        TEXT,
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_contestant ON submissions(contestant_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status     ON submissions(status);

-- Run Metrics (Hypertable)
CREATE TABLE IF NOT EXISTS run_metrics (
    time          TIMESTAMPTZ NOT NULL,
    run_id        TEXT NOT NULL,
    contestant_id TEXT NOT NULL,
    p50_ms        DOUBLE PRECISION,
    p90_ms        DOUBLE PRECISION,
    p99_ms        DOUBLE PRECISION,
    max_tps       BIGINT,
    correctness   DOUBLE PRECISION
);

SELECT create_hypertable('run_metrics', 'time', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_run_metrics_run ON run_metrics(run_id, time DESC);
