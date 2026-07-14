CREATE TABLE IF NOT EXISTS marketops_backtest_calibration_readiness (
  readiness_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT 'marketops',
  domain TEXT NOT NULL DEFAULT 'market_data',
  use_case TEXT NOT NULL DEFAULT 'daily_market_surveillance',
  baseline_id TEXT NOT NULL DEFAULT '',
  comparison_id TEXT NOT NULL DEFAULT '',
  evaluation_id TEXT NOT NULL DEFAULT '',
  candidate_id TEXT NOT NULL DEFAULT '',
  detector_id TEXT NOT NULL DEFAULT '',
  dataset_scope TEXT[] NOT NULL DEFAULT '{}'::text[],
  universe_group TEXT NOT NULL DEFAULT '',
  window_start TIMESTAMPTZ,
  window_end TIMESTAMPTZ,
  readiness_status TEXT NOT NULL CHECK (readiness_status IN ('calibration_ready', 'needs_more_historical_data', 'needs_more_labels', 'label_quality_blocked', 'regression_detected', 'manual_review_required', 'blocked')),
  readiness_reasons TEXT[] NOT NULL DEFAULT '{}'::text[],
  coverage_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
  label_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
  evaluation_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
  thresholds JSONB NOT NULL DEFAULT '{}'::jsonb,
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  requested_by TEXT NOT NULL DEFAULT 'operator-local',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mo_bt_readiness_tenant_status_created
  ON marketops_backtest_calibration_readiness (tenant_id, readiness_status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_readiness_evidence_refs
  ON marketops_backtest_calibration_readiness (tenant_id, baseline_id, comparison_id, evaluation_id, candidate_id);
