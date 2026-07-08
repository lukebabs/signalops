-- Rules catalog foundation for tenant-scoped signal and quality-rule visibility.

CREATE TABLE IF NOT EXISTS catalog_rules (
  tenant_id text NOT NULL,
  rule_id text NOT NULL,
  rule_name text NOT NULL,
  description text NOT NULL DEFAULT '',
  rule_type text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  status text NOT NULL CHECK (status IN ('active', 'inactive', 'deprecated')),
  version integer NOT NULL DEFAULT 1 CHECK (version > 0),
  source_id text,
  pipeline_id text,
  dataset_scope text[] NOT NULL DEFAULT '{}',
  entity_scope text[] NOT NULL DEFAULT '{}',
  expression jsonb NOT NULL DEFAULT '{}'::jsonb,
  actions text[] NOT NULL DEFAULT '{}',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_rules_tenant_status ON catalog_rules (tenant_id, status, rule_id);
CREATE INDEX IF NOT EXISTS idx_catalog_rules_tenant_type ON catalog_rules (tenant_id, rule_type, rule_id);
CREATE INDEX IF NOT EXISTS idx_catalog_rules_source_pipeline ON catalog_rules (tenant_id, source_id, pipeline_id);

INSERT INTO catalog_rules (
  tenant_id, rule_id, rule_name, description, rule_type, severity, status, version,
  source_id, pipeline_id, dataset_scope, entity_scope, expression, actions, metadata
) VALUES (
  'tenant-local',
  'rule-marketdata-eod-price-quality',
  'Market Data EOD Price Quality',
  'Flags Massive EOD equity records with missing or non-positive close prices before downstream signal evaluation.',
  'quality_check',
  'medium',
  'active',
  1,
  'src-massive',
  'pipeline-massive-raw-ingest',
  ARRAY['equity_eod_prices'],
  ARRAY['ticker'],
  '{"language":"json_logic","conditions":[{"field":"close","operator":"exists"},{"field":"close","operator":">","value":0}],"mode":"all"}'::jsonb,
  ARRAY['emit_alert','mark_event_quality_failed'],
  '{"provider":"massive","formerly":"polygon.io","execution":"catalog_only","streaming":false}'::jsonb
) ON CONFLICT (tenant_id, rule_id) DO UPDATE SET
  rule_name = EXCLUDED.rule_name,
  description = EXCLUDED.description,
  rule_type = EXCLUDED.rule_type,
  severity = EXCLUDED.severity,
  status = EXCLUDED.status,
  version = EXCLUDED.version,
  source_id = EXCLUDED.source_id,
  pipeline_id = EXCLUDED.pipeline_id,
  dataset_scope = EXCLUDED.dataset_scope,
  entity_scope = EXCLUDED.entity_scope,
  expression = EXCLUDED.expression,
  actions = EXCLUDED.actions,
  metadata = EXCLUDED.metadata,
  updated_at = now();
