-- Isolated MarketOps back-test substrate ledgers. These tables do not mutate or reference production signal/artifact/proposal ledgers.

CREATE TABLE IF NOT EXISTS marketops_backtest_runs (
  run_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL DEFAULT '',
  source_adapter text NOT NULL DEFAULT 'market_data.massive',
  dataset text NOT NULL DEFAULT '',
  detector_id text NOT NULL,
  detector_version text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('started', 'succeeded', 'failed', 'canceled')),
  requested_by text NOT NULL DEFAULT 'operator-local',
  window_start timestamptz NOT NULL,
  window_end timestamptz NOT NULL,
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (window_end > window_start)
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_runs_tenant_time
  ON marketops_backtest_runs (tenant_id, app_id, domain, use_case, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_runs_status_time
  ON marketops_backtest_runs (tenant_id, status, started_at DESC);

CREATE TABLE IF NOT EXISTS marketops_backtest_signals (
  run_id text NOT NULL REFERENCES marketops_backtest_runs(run_id) ON DELETE CASCADE,
  signal_id text NOT NULL,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_domain text NOT NULL,
  source_adapter text NOT NULL,
  ingestion_mode text NOT NULL,
  dataset text NOT NULL,
  event_ids text[] NOT NULL DEFAULT '{}',
  artifact_ids text[] NOT NULL DEFAULT '{}',
  signal_type text NOT NULL,
  detector_id text NOT NULL,
  detector_version text NOT NULL,
  model_version text NOT NULL,
  signal_time timestamptz NOT NULL,
  observation_time timestamptz NOT NULL,
  effective_time timestamptz NOT NULL,
  processing_time timestamptz NOT NULL,
  window_start timestamptz NOT NULL,
  window_end timestamptz NOT NULL,
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  entities jsonb NOT NULL DEFAULT '[]'::jsonb,
  supporting_metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  graph_targets jsonb NOT NULL DEFAULT '[]'::jsonb,
  semantic_evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  recommendation jsonb,
  correlation_id text NOT NULL,
  trace_id text NOT NULL DEFAULT '',
  causation_id text NOT NULL DEFAULT '',
  replay_job_id text NOT NULL DEFAULT '',
  event jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (run_id, signal_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_signals_run_time
  ON marketops_backtest_signals (run_id, signal_time DESC);

CREATE TABLE IF NOT EXISTS marketops_backtest_artifacts (
  run_id text NOT NULL REFERENCES marketops_backtest_runs(run_id) ON DELETE CASCADE,
  artifact_id text NOT NULL,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  signal_id text NOT NULL,
  signal_type text NOT NULL,
  detector_id text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  event_ids text[] NOT NULL DEFAULT '{}',
  subject_symbol text NOT NULL DEFAULT '',
  artifact_type text NOT NULL,
  artifact jsonb NOT NULL DEFAULT '{}'::jsonb,
  semantic_evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
  graph_targets jsonb NOT NULL DEFAULT '[]'::jsonb,
  supporting_metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  quality_issues text[] NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (run_id, artifact_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_artifacts_run_signal
  ON marketops_backtest_artifacts (run_id, signal_id);

CREATE TABLE IF NOT EXISTS marketops_backtest_graph_proposals (
  run_id text NOT NULL REFERENCES marketops_backtest_runs(run_id) ON DELETE CASCADE,
  proposal_id text NOT NULL,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  artifact_id text NOT NULL,
  signal_id text NOT NULL,
  signal_type text NOT NULL,
  detector_id text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  event_ids text[] NOT NULL DEFAULT '{}',
  subject_symbol text NOT NULL DEFAULT '',
  candidate_type text NOT NULL CHECK (candidate_type IN ('node_candidate', 'relationship_candidate')),
  node_id text NOT NULL DEFAULT '',
  from_node text NOT NULL DEFAULT '',
  relationship text NOT NULL DEFAULT '',
  to_node text NOT NULL DEFAULT '',
  labels text[] NOT NULL DEFAULT '{}',
  properties jsonb NOT NULL DEFAULT '{}'::jsonb,
  raw_candidate jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL DEFAULT 'proposed',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (run_id, proposal_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_graph_proposals_run_symbol
  ON marketops_backtest_graph_proposals (run_id, subject_symbol, updated_at DESC);

CREATE TABLE IF NOT EXISTS marketops_backtest_policy_results (
  run_id text NOT NULL REFERENCES marketops_backtest_runs(run_id) ON DELETE CASCADE,
  policy_result_id text NOT NULL,
  proposal_id text NOT NULL,
  artifact_id text NOT NULL,
  signal_id text NOT NULL,
  tenant_id text NOT NULL,
  subject_symbol text NOT NULL DEFAULT '',
  candidate_type text NOT NULL,
  recommendation text NOT NULL CHECK (recommendation IN ('auto_accept_candidate', 'auto_reject_candidate', 'manual_review_required', 'supersede_candidate')),
  reason text NOT NULL,
  policy_version text NOT NULL,
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  decision_inputs jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (run_id, policy_result_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_policy_results_run_recommendation
  ON marketops_backtest_policy_results (run_id, recommendation, created_at DESC);
