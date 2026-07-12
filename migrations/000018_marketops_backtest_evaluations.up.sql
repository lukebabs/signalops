CREATE TABLE IF NOT EXISTS marketops_backtest_evaluations (
  evaluation_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT 'marketops',
  domain TEXT NOT NULL DEFAULT 'market_data',
  use_case TEXT NOT NULL DEFAULT 'daily_market_surveillance',
  run_id TEXT NOT NULL REFERENCES marketops_backtest_runs(run_id) ON DELETE RESTRICT,
  detector_id TEXT NOT NULL DEFAULT '',
  dataset TEXT NOT NULL DEFAULT '',
  label_source TEXT NOT NULL DEFAULT 'g080_graph_proposal_decision',
  label_version TEXT NOT NULL DEFAULT 'marketops.eval_label.v1',
  scoring_version TEXT NOT NULL DEFAULT 'marketops.eval_scoring.v1',
  requested_by TEXT NOT NULL DEFAULT 'operator-local',
  candidate_count INTEGER NOT NULL DEFAULT 0,
  labeled_count INTEGER NOT NULL DEFAULT 0,
  positive_count INTEGER NOT NULL DEFAULT 0,
  negative_count INTEGER NOT NULL DEFAULT 0,
  superseded_count INTEGER NOT NULL DEFAULT 0,
  unresolved_count INTEGER NOT NULL DEFAULT 0,
  true_positive INTEGER NOT NULL DEFAULT 0,
  false_positive INTEGER NOT NULL DEFAULT 0,
  true_negative INTEGER NOT NULL DEFAULT 0,
  false_negative INTEGER NOT NULL DEFAULT 0,
  manual_review_count INTEGER NOT NULL DEFAULT 0,
  unscored_count INTEGER NOT NULL DEFAULT 0,
  precision DOUBLE PRECISION NOT NULL DEFAULT 0,
  recall DOUBLE PRECISION NOT NULL DEFAULT 0,
  specificity DOUBLE PRECISION NOT NULL DEFAULT 0,
  accuracy DOUBLE PRECISION NOT NULL DEFAULT 0,
  label_coverage DOUBLE PRECISION NOT NULL DEFAULT 0,
  recommendation TEXT NOT NULL DEFAULT 'needs_more_data',
  recommendation_note TEXT NOT NULL DEFAULT '',
  metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mo_bt_evaluations_tenant_created
  ON marketops_backtest_evaluations (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mo_bt_evaluations_run_created
  ON marketops_backtest_evaluations (run_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mo_bt_evaluations_recommendation
  ON marketops_backtest_evaluations (tenant_id, recommendation, created_at DESC);
