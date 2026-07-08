CREATE TABLE IF NOT EXISTS normalized_event_ledger (
  event_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  idempotency_key text NOT NULL,
  schema_id text NOT NULL,
  schema_version text NOT NULL,
  observation_time timestamptz NOT NULL,
  processing_time timestamptz NOT NULL,
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  raw_topic text NOT NULL,
  raw_partition integer NOT NULL,
  raw_offset bigint NOT NULL,
  normalized_topic text NOT NULL,
  normalized_partition integer NOT NULL,
  normalized_offset bigint NOT NULL,
  normalized_payload jsonb NOT NULL,
  entities jsonb NOT NULL DEFAULT '[]'::jsonb,
  evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  event jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, source_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_normalized_event_source_time
  ON normalized_event_ledger (tenant_id, source_id, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_normalized_event_dataset_time
  ON normalized_event_ledger (dataset, observation_time DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_normalized_event_raw_position
  ON normalized_event_ledger (raw_topic, raw_partition, raw_offset);
