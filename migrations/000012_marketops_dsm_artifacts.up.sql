-- First-class persisted MarketOps DSM artifact proposals derived from signal.v1 semantic evidence.

CREATE TABLE IF NOT EXISTS marketops_dsm_artifacts (
  artifact_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  signal_id text NOT NULL REFERENCES signal_ledger(signal_id) ON DELETE CASCADE,
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
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_artifacts_tenant_time
  ON marketops_dsm_artifacts (tenant_id, app_id, domain, use_case, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_artifacts_signal
  ON marketops_dsm_artifacts (signal_id);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_artifacts_type_time
  ON marketops_dsm_artifacts (tenant_id, signal_type, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_artifacts_symbol_time
  ON marketops_dsm_artifacts (tenant_id, subject_symbol, updated_at DESC);
