-- Pipeline catalog foundation for tenant-scoped processing topology visibility.

CREATE TABLE IF NOT EXISTS catalog_pipelines (
  tenant_id text NOT NULL,
  pipeline_id text NOT NULL,
  source_id text NOT NULL,
  source_domain text NOT NULL,
  pipeline_name text NOT NULL,
  description text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('active', 'inactive', 'deprecated')),
  stages text[] NOT NULL DEFAULT '{}',
  input_datasets text[] NOT NULL DEFAULT '{}',
  output_topics text[] NOT NULL DEFAULT '{}',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, pipeline_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_pipelines_tenant_status ON catalog_pipelines (tenant_id, status, pipeline_id);
CREATE INDEX IF NOT EXISTS idx_catalog_pipelines_source ON catalog_pipelines (tenant_id, source_id, pipeline_id);

INSERT INTO catalog_pipelines (
  tenant_id, pipeline_id, source_id, source_domain, pipeline_name, description,
  status, stages, input_datasets, output_topics, metadata
) VALUES (
  'tenant-local',
  'pipeline-massive-raw-ingest',
  'src-massive',
  'market_data',
  'Massive Raw Ingest',
  'Scheduled Massive market-data pull through raw event publication, raw ledger persistence, and idempotency tracking.',
  'active',
  ARRAY['scheduled_pull', 'raw_event_build', 'broker_publish', 'raw_ledger_persist', 'idempotency_persist'],
  ARRAY['equity_eod_prices', 'option_contracts_daily'],
  ARRAY['signalops.local.raw.v1'],
  '{"adapter":"market_data.massive","provider":"massive","formerly":"polygon.io","streaming":false}'::jsonb
) ON CONFLICT (tenant_id, pipeline_id) DO UPDATE SET
  source_id = EXCLUDED.source_id,
  source_domain = EXCLUDED.source_domain,
  pipeline_name = EXCLUDED.pipeline_name,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  stages = EXCLUDED.stages,
  input_datasets = EXCLUDED.input_datasets,
  output_topics = EXCLUDED.output_topics,
  metadata = EXCLUDED.metadata,
  updated_at = now();
