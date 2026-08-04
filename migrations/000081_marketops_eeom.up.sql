CREATE TABLE IF NOT EXISTS marketops_eeom_results (
  result_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  earnings_event_id text NOT NULL,
  earnings_date date NOT NULL,
  session_date date NOT NULL,
  model_version text NOT NULL,
  score double precision NOT NULL CHECK (score >= 0 AND score <= 100),
  posture text NOT NULL,
  classification text NOT NULL,
  evidence_quality text NOT NULL,
  eligible boolean NOT NULL,
  result_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, symbol, earnings_event_id, session_date, model_version)
);
CREATE INDEX IF NOT EXISTS idx_marketops_eeom_results_current
  ON marketops_eeom_results (tenant_id, earnings_date, session_date DESC, score DESC, symbol);

INSERT INTO algorithm_definitions (tenant_id,algorithm_id,name,description,algorithm_type,runtime_type,input_features,input_event_types,output_schema,config_schema,default_config,version,status,metadata) VALUES
('tenant-local','signalops.algorithms.earnings_event_opportunity_v1','Earnings Event Opportunity Model','Deterministic pre-earnings setup-quality orchestration. Research-only; not an earnings forecast or recommendation.','trend_detection','container_plugin',ARRAY['risk_reward_temporal_v1','valuation_composite_v3','distressed_opportunity_scoring_v3','tactical_market_posture_v1'],ARRAY['market_data.massive.earnings_calendar'],'{"type":"object"}'::jsonb,'{"type":"object"}'::jsonb,'{"model_version":"eeom-v1"}'::jsonb,'v1','active','{"research_only":true,"marketops_role":"earnings_event_opportunity"}'::jsonb)
ON CONFLICT (tenant_id,algorithm_id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,input_features=EXCLUDED.input_features,input_event_types=EXCLUDED.input_event_types,default_config=EXCLUDED.default_config,version=EXCLUDED.version,status=EXCLUDED.status,metadata=EXCLUDED.metadata,updated_at=now();
