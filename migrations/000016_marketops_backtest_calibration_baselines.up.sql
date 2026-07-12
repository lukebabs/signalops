CREATE TABLE IF NOT EXISTS marketops_backtest_calibration_baselines (
  baseline_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT 'marketops',
  domain TEXT NOT NULL DEFAULT 'market_data',
  use_case TEXT NOT NULL DEFAULT 'daily_market_surveillance',
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  summary_id TEXT NOT NULL REFERENCES marketops_backtest_calibration_summaries(summary_id) ON DELETE RESTRICT,
  detector_id TEXT NOT NULL DEFAULT '',
  dataset TEXT NOT NULL DEFAULT '',
  scope JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_by TEXT NOT NULL DEFAULT 'operator-local',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mo_bt_cal_baselines_tenant_status_created
  ON marketops_backtest_calibration_baselines (tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_cal_baselines_detector_dataset
  ON marketops_backtest_calibration_baselines (tenant_id, detector_id, dataset, created_at DESC);

CREATE TABLE IF NOT EXISTS marketops_backtest_calibration_comparisons (
  comparison_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  baseline_id TEXT NOT NULL REFERENCES marketops_backtest_calibration_baselines(baseline_id) ON DELETE RESTRICT,
  baseline_summary_id TEXT NOT NULL REFERENCES marketops_backtest_calibration_summaries(summary_id) ON DELETE RESTRICT,
  candidate_summary_id TEXT NOT NULL REFERENCES marketops_backtest_calibration_summaries(summary_id) ON DELETE RESTRICT,
  detector_id TEXT NOT NULL DEFAULT '',
  dataset TEXT NOT NULL DEFAULT '',
  comparison_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
  recommendation TEXT NOT NULL CHECK (recommendation IN ('needs_more_data', 'regression_candidate', 'improvement_candidate', 'neutral_candidate', 'manual_review_required')),
  recommendation_reason TEXT NOT NULL DEFAULT '',
  created_by TEXT NOT NULL DEFAULT 'operator-local',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mo_bt_cal_comparisons_tenant_created
  ON marketops_backtest_calibration_comparisons (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_cal_comparisons_baseline_created
  ON marketops_backtest_calibration_comparisons (baseline_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_cal_comparisons_recommendation_created
  ON marketops_backtest_calibration_comparisons (tenant_id, recommendation, created_at DESC);
