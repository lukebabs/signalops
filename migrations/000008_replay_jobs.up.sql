-- Replay jobs are control-plane requests for replaying Timescale temporal ledgers.
CREATE TABLE IF NOT EXISTS replay_jobs (
  replay_job_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text,
  dataset text,
  source_kind text NOT NULL,
  replay_mode text NOT NULL,
  status text NOT NULL,
  requested_by text NOT NULL,
  window_start timestamptz NOT NULL,
  window_end timestamptz NOT NULL,
  started_at timestamptz,
  completed_at timestamptz,
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  options jsonb NOT NULL DEFAULT '{}'::jsonb,
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (window_end > window_start),
  CHECK (source_kind IN ('raw_events', 'normalized_events', 'signals')),
  CHECK (replay_mode IN ('original', 'latest_compatible', 'explicit')),
  CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'canceled'))
);

CREATE INDEX IF NOT EXISTS idx_replay_jobs_tenant_created ON replay_jobs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_replay_jobs_status_created ON replay_jobs (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_replay_jobs_source_window ON replay_jobs (tenant_id, source_id, dataset, source_kind, window_start, window_end);
