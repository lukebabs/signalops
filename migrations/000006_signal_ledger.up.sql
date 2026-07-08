CREATE TABLE IF NOT EXISTS signal_ledger (
  signal_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_domain text NOT NULL,
  source_adapter text NOT NULL,
  ingestion_mode text NOT NULL,
  dataset text NOT NULL,
  event_ids text[] NOT NULL,
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
  trace_id text,
  causation_id text,
  replay_job_id text,
  broker_topic text NOT NULL,
  broker_partition integer NOT NULL,
  broker_offset bigint NOT NULL,
  event jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (broker_topic, broker_partition, broker_offset)
);

CREATE INDEX IF NOT EXISTS idx_signal_ledger_tenant_time
  ON signal_ledger (tenant_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_source_time
  ON signal_ledger (tenant_id, source_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_detector_time
  ON signal_ledger (tenant_id, detector_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_severity_time
  ON signal_ledger (tenant_id, severity, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_event_ids
  ON signal_ledger USING gin (event_ids);
