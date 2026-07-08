CREATE TABLE IF NOT EXISTS alert_ledger (
  alert_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_domain text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  signal_id text NOT NULL REFERENCES signal_ledger(signal_id) ON DELETE CASCADE,
  detector_id text NOT NULL,
  alert_type text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  status text NOT NULL CHECK (status IN ('open', 'acknowledged', 'resolved', 'suppressed')),
  title text NOT NULL,
  summary text NOT NULL,
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  event_ids text[] NOT NULL,
  entities jsonb NOT NULL DEFAULT '[]'::jsonb,
  evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  recommendation jsonb,
  correlation_id text NOT NULL,
  first_observed_at timestamptz NOT NULL,
  last_observed_at timestamptz NOT NULL,
  acknowledged_at timestamptz,
  acknowledged_by text,
  resolved_at timestamptz,
  resolved_by text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alert_ledger_tenant_status_time
  ON alert_ledger (tenant_id, status, last_observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_ledger_source_time
  ON alert_ledger (tenant_id, source_id, last_observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_ledger_severity_time
  ON alert_ledger (tenant_id, severity, last_observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_ledger_signal
  ON alert_ledger (signal_id);
CREATE INDEX IF NOT EXISTS idx_alert_ledger_event_ids
  ON alert_ledger USING gin (event_ids);

CREATE TABLE IF NOT EXISTS insight_ledger (
  insight_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_domain text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  signal_id text NOT NULL REFERENCES signal_ledger(signal_id) ON DELETE CASCADE,
  detector_id text NOT NULL,
  insight_type text NOT NULL,
  status text NOT NULL CHECK (status IN ('active', 'reviewed', 'dismissed', 'archived')),
  title text NOT NULL,
  summary text NOT NULL,
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  event_ids text[] NOT NULL,
  entities jsonb NOT NULL DEFAULT '[]'::jsonb,
  supporting_metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  semantic_evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  recommendation jsonb,
  correlation_id text NOT NULL,
  observed_at timestamptz NOT NULL,
  reviewed_at timestamptz,
  reviewed_by text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_insight_ledger_tenant_status_time
  ON insight_ledger (tenant_id, status, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_insight_ledger_source_time
  ON insight_ledger (tenant_id, source_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_insight_ledger_type_time
  ON insight_ledger (tenant_id, insight_type, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_insight_ledger_signal
  ON insight_ledger (signal_id);
CREATE INDEX IF NOT EXISTS idx_insight_ledger_event_ids
  ON insight_ledger USING gin (event_ids);
