-- Persisted MarketOps back-test calibration summary snapshots.
-- These rows summarize isolated back-test runs and never mutate production signal/artifact/proposal ledgers.

CREATE TABLE IF NOT EXISTS marketops_backtest_calibration_summaries (
  summary_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL DEFAULT '',
  dataset text NOT NULL DEFAULT '',
  detector_id text NOT NULL DEFAULT '',
  status_filter text NOT NULL DEFAULT '',
  requested_by text NOT NULL DEFAULT 'operator-local',
  run_ids text[] NOT NULL DEFAULT '{}',
  run_count integer NOT NULL DEFAULT 0 CHECK (run_count >= 0),
  succeeded_count integer NOT NULL DEFAULT 0 CHECK (succeeded_count >= 0),
  failed_count integer NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
  zero_input_count integer NOT NULL DEFAULT 0 CHECK (zero_input_count >= 0),
  scanned integer NOT NULL DEFAULT 0 CHECK (scanned >= 0),
  signals integer NOT NULL DEFAULT 0 CHECK (signals >= 0),
  artifacts integer NOT NULL DEFAULT 0 CHECK (artifacts >= 0),
  graph_proposals integer NOT NULL DEFAULT 0 CHECK (graph_proposals >= 0),
  policy_results integer NOT NULL DEFAULT 0 CHECK (policy_results >= 0),
  signal_yield double precision NOT NULL DEFAULT 0 CHECK (signal_yield >= 0),
  policy_results_per_signal double precision NOT NULL DEFAULT 0 CHECK (policy_results_per_signal >= 0),
  recommendation_counts jsonb NOT NULL DEFAULT '{}'::jsonb,
  recommendation_shares jsonb NOT NULL DEFAULT '{}'::jsonb,
  dominant_recommendation jsonb NOT NULL DEFAULT '{}'::jsonb,
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_calibration_tenant_time
  ON marketops_backtest_calibration_summaries (tenant_id, app_id, domain, use_case, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_calibration_detector_dataset
  ON marketops_backtest_calibration_summaries (tenant_id, detector_id, dataset, created_at DESC);
