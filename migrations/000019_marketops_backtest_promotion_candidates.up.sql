CREATE TABLE IF NOT EXISTS marketops_backtest_promotion_candidates (
  candidate_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT 'marketops',
  domain TEXT NOT NULL DEFAULT 'market_data',
  use_case TEXT NOT NULL DEFAULT 'daily_market_surveillance',
  baseline_id TEXT NOT NULL REFERENCES marketops_backtest_calibration_baselines(baseline_id) ON DELETE RESTRICT,
  comparison_id TEXT NOT NULL REFERENCES marketops_backtest_calibration_comparisons(comparison_id) ON DELETE RESTRICT,
  evaluation_id TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  detector_id TEXT NOT NULL DEFAULT '',
  detector_version TEXT NOT NULL DEFAULT '',
  dataset TEXT NOT NULL DEFAULT '',
  policy_version TEXT NOT NULL DEFAULT '',
  candidate_version TEXT NOT NULL DEFAULT '',
  readiness_status TEXT NOT NULL CHECK (readiness_status IN ('ready_for_review', 'needs_more_data', 'manual_review_required', 'regression_detected', 'blocked')),
  readiness_reasons TEXT[] NOT NULL DEFAULT '{}'::text[],
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'approved_for_promotion', 'rejected', 'deferred', 'superseded')),
  requested_by TEXT NOT NULL DEFAULT 'operator-local',
  reviewed_by TEXT NOT NULL DEFAULT '',
  reviewed_at TIMESTAMPTZ,
  decision_note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mo_bt_promo_candidates_tenant_status_created
  ON marketops_backtest_promotion_candidates (tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_promo_candidates_readiness_created
  ON marketops_backtest_promotion_candidates (tenant_id, readiness_status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_promo_candidates_evidence_refs
  ON marketops_backtest_promotion_candidates (tenant_id, baseline_id, comparison_id, evaluation_id);
