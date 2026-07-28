-- Isolated algorithm evaluation and bounded equity-backfill campaign ledgers.
-- These tables never write production algorithm_results or signal ledgers.

CREATE TABLE IF NOT EXISTS marketops_algorithm_evaluation_runs (
  run_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  universe_group text NOT NULL,
  algorithm_ids text[] NOT NULL,
  modes text[] NOT NULL,
  window_start date NOT NULL,
  window_end date NOT NULL,
  as_of_date date NOT NULL,
  status text NOT NULL CHECK (status IN ('running','succeeded','partial','failed')),
  parameters jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(parameters) = 'object'),
  coverage jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(coverage) = 'object'),
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metrics) = 'object'),
  error_message text NOT NULL DEFAULT '',
  requested_by text NOT NULL DEFAULT 'operator-local',
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (window_end > window_start),
  CHECK (as_of_date >= window_start)
);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_evaluation_runs_tenant_time
  ON marketops_algorithm_evaluation_runs (tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_evaluation_runs_status
  ON marketops_algorithm_evaluation_runs (tenant_id, status, started_at DESC);

CREATE TABLE IF NOT EXISTS marketops_algorithm_evaluation_results (
  evaluation_result_id text PRIMARY KEY,
  run_id text NOT NULL REFERENCES marketops_algorithm_evaluation_runs(run_id) ON DELETE CASCADE,
  tenant_id text NOT NULL,
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  evaluation_mode text NOT NULL CHECK (evaluation_mode IN ('retrospective','walk_forward')),
  evaluation_profile text NOT NULL CHECK (evaluation_profile IN ('directional','event_study','forecast','classification')),
  result_type text NOT NULL,
  symbol text NOT NULL,
  observation_session_date date NOT NULL,
  score double precision NOT NULL CHECK (score >= 0),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  severity text NOT NULL CHECK (severity IN ('info','low','medium','high','critical')),
  direction text NOT NULL CHECK (direction IN ('upside','downside','non_directional')),
  result_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result_payload) = 'object'),
  input_provenance jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input_provenance) = 'object'),
  source_event_ids text[] NOT NULL DEFAULT '{}',
  feature_value_ids text[] NOT NULL DEFAULT '{}',
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (run_id, deterministic_key)
);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_evaluation_results_run_algorithm
  ON marketops_algorithm_evaluation_results (run_id, algorithm_id, observation_session_date);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_evaluation_results_symbol
  ON marketops_algorithm_evaluation_results (tenant_id, symbol, observation_session_date DESC);

CREATE TABLE IF NOT EXISTS marketops_algorithm_evaluation_outcomes (
  evaluation_outcome_id text PRIMARY KEY,
  run_id text NOT NULL REFERENCES marketops_algorithm_evaluation_runs(run_id) ON DELETE CASCADE,
  evaluation_result_id text NOT NULL REFERENCES marketops_algorithm_evaluation_results(evaluation_result_id) ON DELETE CASCADE,
  tenant_id text NOT NULL,
  horizon_sessions integer NOT NULL CHECK (horizon_sessions IN (1,5,10,20)),
  outcome_status text NOT NULL CHECK (outcome_status IN ('pending','matured','missing_price')),
  matured_session_date date,
  forward_return double precision,
  absolute_forward_return double precision,
  max_favorable_excursion double precision,
  max_adverse_excursion double precision,
  maximum_drawdown double precision,
  realized_vol_change double precision,
  directional_hit boolean,
  threshold_hit boolean,
  outcome_event_ids text[] NOT NULL DEFAULT '{}',
  outcome_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(outcome_payload) = 'object'),
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (run_id, deterministic_key),
  CHECK ((outcome_status = 'matured' AND matured_session_date IS NOT NULL AND forward_return IS NOT NULL)
    OR (outcome_status <> 'matured' AND matured_session_date IS NULL AND forward_return IS NULL))
);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_evaluation_outcomes_run_status
  ON marketops_algorithm_evaluation_outcomes (run_id, outcome_status, horizon_sessions);

CREATE TABLE IF NOT EXISTS marketops_algorithm_evaluation_backfill_campaigns (
  campaign_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  universe_group text NOT NULL,
  window_start date NOT NULL,
  window_end date NOT NULL,
  status text NOT NULL CHECK (status IN ('planned','running','succeeded','partial','failed')),
  parameters jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(parameters) = 'object'),
  coverage jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(coverage) = 'object'),
  child_run_ids text[] NOT NULL DEFAULT '{}',
  error_message text NOT NULL DEFAULT '',
  requested_by text NOT NULL DEFAULT 'operator-local',
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (window_end > window_start)
);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_evaluation_backfill_campaigns_tenant_time
  ON marketops_algorithm_evaluation_backfill_campaigns (tenant_id, created_at DESC);
