-- DB-backed scheduled-job operations status for Admin Workbench.
-- Runtime job state belongs in the dedicated MarketOps database, not in git-tracked JSON files.

CREATE TABLE marketops_scheduled_job_statuses (
  job_id text PRIMARY KEY,
  schedule text NOT NULL,
  timezone text NOT NULL,
  status text NOT NULL CHECK (status IN ('pending','running','succeeded','failed','skipped','recovery_needed','recovering','degraded')),
  reason text NOT NULL DEFAULT '',
  started_at timestamptz,
  completed_at timestamptz,
  exit_code integer,
  detail jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(detail) = 'object'),
  runner text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE marketops_scheduled_job_runs (
  run_id text PRIMARY KEY,
  job_id text NOT NULL,
  schedule text NOT NULL,
  timezone text NOT NULL,
  status text NOT NULL CHECK (status IN ('running','succeeded','failed','skipped','recovery_needed','recovering','degraded')),
  reason text NOT NULL DEFAULT '',
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  exit_code integer,
  detail jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(detail) = 'object'),
  runner text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_marketops_scheduled_job_runs_job_started ON marketops_scheduled_job_runs (job_id, started_at DESC);
CREATE INDEX idx_marketops_scheduled_job_runs_status_started ON marketops_scheduled_job_runs (status, started_at DESC);

REVOKE ALL ON marketops_scheduled_job_statuses, marketops_scheduled_job_runs FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON marketops_scheduled_job_statuses, marketops_scheduled_job_runs TO signalops;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_gateway') THEN
    GRANT SELECT ON marketops_scheduled_job_statuses, marketops_scheduled_job_runs TO signalops_subscriber_gateway;
  END IF;
END $$;
