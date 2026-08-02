CREATE TABLE IF NOT EXISTS marketops_valuation_snapshots (
  snapshot_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  available_at timestamptz NOT NULL,
  sector text NOT NULL DEFAULT '',
  industry text NOT NULL DEFAULT '',
  provider text NOT NULL DEFAULT 'massive',
  provider_request_ids text[] NOT NULL DEFAULT '{}',
  input_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, symbol, session_date, available_at)
);
CREATE INDEX IF NOT EXISTS idx_marketops_valuation_snapshots_peer ON marketops_valuation_snapshots (tenant_id, session_date DESC, sector, industry, symbol);

CREATE TABLE IF NOT EXISTS marketops_valuation_results (
  result_id text PRIMARY KEY,
  snapshot_id text NOT NULL REFERENCES marketops_valuation_snapshots(snapshot_id),
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  algorithm_id text NOT NULL,
  model_version text NOT NULL,
  score double precision NOT NULL,
  fair_value double precision NOT NULL,
  classification text NOT NULL,
  confidence integer NOT NULL CHECK (confidence >= 0 AND confidence <= 100),
  confidence_label text NOT NULL,
  evaluation_status text NOT NULL,
  eligible boolean NOT NULL,
  result_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (snapshot_id, algorithm_id)
);
CREATE INDEX IF NOT EXISTS idx_marketops_valuation_results_current ON marketops_valuation_results (tenant_id, algorithm_id, session_date DESC, eligible DESC, score DESC);

INSERT INTO algorithm_definitions (tenant_id,algorithm_id,name,description,algorithm_type,runtime_type,input_features,input_event_types,output_schema,config_schema,default_config,version,status,metadata) VALUES
('tenant-local','signalops.algorithms.valuation_composite_v3','Valuation Composite','Deterministic current-daily valuation composite using provider-backed fundamentals.','trend_detection','container_plugin',ARRAY['marketops_fundamental_snapshot'],ARRAY['marketops_fundamental_snapshot'],'{"type":"object"}'::jsonb,'{"type":"object"}'::jsonb,'{"model_version":"vc-dosm-3.0"}'::jsonb,'v3','active','{"research_only":true,"marketops_role":"valuation_composite"}'::jsonb),
('tenant-local','signalops.algorithms.distressed_opportunity_scoring_v3','Distressed Opportunity Scoring Model','Deterministic current-daily fundamental and technical opportunity composite.','trend_detection','container_plugin',ARRAY['marketops_fundamental_snapshot','valuation_composite_v3'],ARRAY['marketops_fundamental_snapshot'],'{"type":"object"}'::jsonb,'{"type":"object"}'::jsonb,'{"model_version":"vc-dosm-3.0"}'::jsonb,'v3','active','{"research_only":true,"marketops_role":"distressed_opportunity_scoring"}'::jsonb)
ON CONFLICT (tenant_id,algorithm_id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,algorithm_type=EXCLUDED.algorithm_type,runtime_type=EXCLUDED.runtime_type,input_features=EXCLUDED.input_features,input_event_types=EXCLUDED.input_event_types,output_schema=EXCLUDED.output_schema,config_schema=EXCLUDED.config_schema,default_config=EXCLUDED.default_config,version=EXCLUDED.version,status=EXCLUDED.status,metadata=EXCLUDED.metadata,updated_at=now();
