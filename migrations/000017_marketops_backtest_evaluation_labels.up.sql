CREATE TABLE IF NOT EXISTS marketops_backtest_evaluation_labels (
  label_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT 'marketops',
  domain TEXT NOT NULL DEFAULT 'market_data',
  use_case TEXT NOT NULL DEFAULT 'daily_market_surveillance',
  source_proposal_id TEXT NOT NULL REFERENCES marketops_dsm_graph_proposals(proposal_id) ON DELETE RESTRICT,
  artifact_id TEXT NOT NULL DEFAULT '',
  signal_id TEXT NOT NULL DEFAULT '',
  subject_symbol TEXT NOT NULL DEFAULT '',
  candidate_type TEXT NOT NULL DEFAULT '',
  graph_fact_key TEXT NOT NULL DEFAULT '',
  decision_status TEXT NOT NULL,
  label TEXT NOT NULL CHECK (label IN ('positive', 'negative', 'superseded', 'unresolved')),
  label_source TEXT NOT NULL DEFAULT 'g080_graph_proposal_decision',
  labeled_by TEXT NOT NULL DEFAULT 'operator-local',
  labeled_at TIMESTAMPTZ NOT NULL,
  label_version TEXT NOT NULL DEFAULT 'marketops.eval_label.v1',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (source_proposal_id, label_version)
);

CREATE INDEX IF NOT EXISTS idx_mo_bt_eval_labels_tenant_created
  ON marketops_backtest_evaluation_labels (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_eval_labels_subject_candidate
  ON marketops_backtest_evaluation_labels (tenant_id, subject_symbol, candidate_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_eval_labels_decision_label
  ON marketops_backtest_evaluation_labels (tenant_id, decision_status, label, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mo_bt_eval_labels_signal
  ON marketops_backtest_evaluation_labels (signal_id);
