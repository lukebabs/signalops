-- Replay worker heartbeats provide durable operational visibility for always-on replay execution.
CREATE TABLE IF NOT EXISTS replay_worker_heartbeats (
  worker_id text PRIMARY KEY,
  status text NOT NULL,
  process_started_at timestamptz NOT NULL,
  last_seen_at timestamptz NOT NULL,
  last_claimed_at timestamptz,
  last_claimed_replay_job_id text,
  last_completed_at timestamptz,
  last_completed_replay_job_id text,
  last_error_at timestamptz,
  last_error_message text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (status IN ('idle', 'running', 'error', 'stopping'))
);

CREATE INDEX IF NOT EXISTS idx_replay_worker_heartbeats_last_seen ON replay_worker_heartbeats (last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_replay_worker_heartbeats_status_seen ON replay_worker_heartbeats (status, last_seen_at DESC);
