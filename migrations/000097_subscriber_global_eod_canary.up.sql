-- Subscriber Project S4: approval-gated shared EOD canary preparation.
-- This migration records a frozen cohort only. It does not enqueue provider work,
-- alter existing MarketOps scheduling, or permit collection execution.

CREATE TABLE subscriber_global_eod_canary_runs (
  canary_run_id text PRIMARY KEY,
  plan_run_id text NOT NULL REFERENCES subscriber_global_eod_hot_set_plan_runs (plan_run_id) ON DELETE RESTRICT,
  canary_version text NOT NULL,
  session_date date NOT NULL,
  execution_state text NOT NULL CHECK (execution_state IN ('prepared', 'approved', 'executing', 'completed', 'failed', 'rolled_back')),
  max_symbols integer NOT NULL CHECK (max_symbols > 0 AND max_symbols <= 50),
  selected_count integer NOT NULL CHECK (selected_count > 0 AND selected_count <= max_symbols),
  parity_required boolean NOT NULL DEFAULT true,
  provider_execution_enabled boolean NOT NULL DEFAULT false CHECK (provider_execution_enabled = false),
  scheduled_execution_enabled boolean NOT NULL DEFAULT false CHECK (scheduled_execution_enabled = false),
  report jsonb NOT NULL DEFAULT '{}'::jsonb,
  prepared_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  prepared_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (plan_run_id, session_date)
);

CREATE TABLE subscriber_global_eod_canary_members (
  canary_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_runs (canary_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets (global_asset_id) ON DELETE RESTRICT,
  priority integer NOT NULL CHECK (priority > 0),
  source_rank integer,
  selection_reason text NOT NULL,
  PRIMARY KEY (canary_run_id, global_asset_id),
  UNIQUE (canary_run_id, priority)
);
CREATE INDEX idx_subscriber_global_eod_canary_members_asset
  ON subscriber_global_eod_canary_members (global_asset_id, canary_run_id DESC);

ALTER TABLE subscriber_global_eod_canary_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_members OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_canary_runs, subscriber_global_eod_canary_members FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_eod_canary_runs, subscriber_global_eod_canary_members TO signalops_subscriber_global_eod;
