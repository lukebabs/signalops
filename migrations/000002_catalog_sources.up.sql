-- Source catalog foundation for tenant-scoped adapter visibility.

CREATE TABLE IF NOT EXISTS catalog_sources (
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_domain text NOT NULL,
  source_adapter text NOT NULL,
  display_name text NOT NULL,
  description text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('active', 'inactive', 'deprecated')),
  ingestion_modes text[] NOT NULL DEFAULT '{}',
  datasets text[] NOT NULL DEFAULT '{}',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, source_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_sources_tenant_status ON catalog_sources (tenant_id, status, source_id);
CREATE INDEX IF NOT EXISTS idx_catalog_sources_adapter ON catalog_sources (source_adapter, tenant_id);

INSERT INTO catalog_sources (
  tenant_id, source_id, source_domain, source_adapter, display_name, description,
  status, ingestion_modes, datasets, metadata
) VALUES (
  'tenant-local',
  'src-massive',
  'market_data',
  'market_data.massive',
  'Massive Market Data',
  'Scheduled Massive market-data source for equity EOD prices and daily option contracts.',
  'active',
  ARRAY['scheduled_pull'],
  ARRAY['equity_eod_prices', 'option_contracts_daily'],
  '{"provider":"massive","formerly":"polygon.io","streaming":false}'::jsonb
) ON CONFLICT (tenant_id, source_id) DO UPDATE SET
  source_domain = EXCLUDED.source_domain,
  source_adapter = EXCLUDED.source_adapter,
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  ingestion_modes = EXCLUDED.ingestion_modes,
  datasets = EXCLUDED.datasets,
  metadata = EXCLUDED.metadata,
  updated_at = now();
