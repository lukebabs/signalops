CREATE TABLE IF NOT EXISTS syncratic_context_windows (
  context_window_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  subject_type text NOT NULL DEFAULT 'ticker',
  subject_id text NOT NULL DEFAULT '',
  subject_symbol text NOT NULL,
  window_start timestamptz NOT NULL,
  window_end timestamptz NOT NULL,
  context_strategy text NOT NULL,
  context_builder_version text NOT NULL,
  signal_types text[] NOT NULL DEFAULT '{}',
  detector_ids text[] NOT NULL DEFAULT '{}',
  event_ids text[] NOT NULL DEFAULT '{}',
  signal_ids text[] NOT NULL DEFAULT '{}',
  alert_ids text[] NOT NULL DEFAULT '{}',
  artifact_ids text[] NOT NULL DEFAULT '{}',
  graph_proposal_ids text[] NOT NULL DEFAULT '{}',
  label_ids text[] NOT NULL DEFAULT '{}',
  baseline_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
  evaluation_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
  promotion_candidate_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
  summary_metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  evidence_digest text NOT NULL,
  idempotency_key text NOT NULL,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'superseded')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, use_case, context_strategy, subject_symbol, window_start, window_end, context_builder_version)
);

CREATE INDEX IF NOT EXISTS idx_syncratic_context_windows_tenant_time
  ON syncratic_context_windows (tenant_id, app_id, domain, use_case, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_syncratic_context_windows_subject_time
  ON syncratic_context_windows (tenant_id, subject_symbol, context_strategy, window_end DESC);

CREATE INDEX IF NOT EXISTS idx_syncratic_context_windows_digest
  ON syncratic_context_windows (tenant_id, evidence_digest);

CREATE TABLE IF NOT EXISTS syncratic_insights (
  syncratic_insight_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  context_window_id text NOT NULL REFERENCES syncratic_context_windows(context_window_id) ON DELETE CASCADE,
  insight_type text NOT NULL,
  subject_type text NOT NULL DEFAULT 'ticker',
  subject_id text NOT NULL DEFAULT '',
  subject_symbol text NOT NULL,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'reviewed', 'dismissed', 'archived', 'superseded')),
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  title text NOT NULL,
  summary text NOT NULL,
  explanation text NOT NULL,
  supporting_alert_ids text[] NOT NULL DEFAULT '{}',
  supporting_signal_ids text[] NOT NULL DEFAULT '{}',
  supporting_event_ids text[] NOT NULL DEFAULT '{}',
  supporting_artifact_ids text[] NOT NULL DEFAULT '{}',
  related_graph_proposal_ids text[] NOT NULL DEFAULT '{}',
  related_label_ids text[] NOT NULL DEFAULT '{}',
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  recommendation jsonb NOT NULL DEFAULT '{}'::jsonb,
  builder_version text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (context_window_id, insight_type, builder_version)
);

CREATE INDEX IF NOT EXISTS idx_syncratic_insights_tenant_time
  ON syncratic_insights (tenant_id, app_id, domain, use_case, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_syncratic_insights_subject_time
  ON syncratic_insights (tenant_id, subject_symbol, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_syncratic_insights_context
  ON syncratic_insights (context_window_id);
