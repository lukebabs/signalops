CREATE TABLE IF NOT EXISTS retention_policies (
  tenant_id text NOT NULL,
  policy_id text NOT NULL,
  app_id text NOT NULL,
  domain text NOT NULL,
  data_class text NOT NULL,
  retention_days integer NOT NULL CHECK (retention_days > 0),
  mode text NOT NULL DEFAULT 'dry_run' CHECK (mode IN ('dry_run','enforced','disabled')),
  preservation_rule text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, policy_id)
);
CREATE TABLE IF NOT EXISTS retention_runs (
  run_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  policy_id text NOT NULL,
  mode text NOT NULL,
  status text NOT NULL CHECK (status IN ('running','succeeded','failed','blocked')),
  candidate_rows bigint NOT NULL DEFAULT 0,
  affected_rows bigint NOT NULL DEFAULT 0,
  oldest_candidate_at timestamptz,
  newest_candidate_at timestamptz,
  detail jsonb NOT NULL DEFAULT '{}'::jsonb,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  FOREIGN KEY (tenant_id, policy_id) REFERENCES retention_policies (tenant_id, policy_id)
);
CREATE INDEX IF NOT EXISTS idx_retention_runs_policy_time ON retention_runs (tenant_id, policy_id, started_at DESC);
CREATE TABLE IF NOT EXISTS retention_evidence_receipts (
  tenant_id text NOT NULL,
  app_id text NOT NULL,
  domain text NOT NULL,
  event_id text NOT NULL,
  source_id text NOT NULL,
  dataset text NOT NULL,
  observed_at timestamptz NOT NULL,
  payload_hash text NOT NULL,
  parser_version text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  preserved_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_retention_evidence_receipts_scope_time ON retention_evidence_receipts (tenant_id, app_id, domain, observed_at DESC);
CREATE TABLE IF NOT EXISTS cyberops_iot_daily_features (
  tenant_id text NOT NULL,
  feature_date date NOT NULL,
  device_ip inet NOT NULL,
  peer_ip inet NOT NULL,
  protocol text NOT NULL,
  destination_port integer NOT NULL,
  allowed_log_count bigint NOT NULL,
  active_hours integer NOT NULL,
  first_seen timestamptz NOT NULL,
  last_seen timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, feature_date, device_ip, peer_ip, protocol, destination_port)
);
CREATE INDEX IF NOT EXISTS idx_cyberops_iot_daily_features_device_date ON cyberops_iot_daily_features (tenant_id, device_ip, feature_date DESC);
INSERT INTO retention_policies (tenant_id,policy_id,app_id,domain,data_class,retention_days,mode,preservation_rule,description) VALUES
('tenant-local','marketops.raw_events_30d','marketops','market_data','raw_events',30,'dry_run','canonical_market_metadata','Provider payload grace period after durable normalization.'),
('tenant-local','marketops.equity_metadata_12m','marketops','market_data','equity_metadata',365,'dry_run','algorithm_ready','Canonical EOD facts, technical features, outputs, and bounded enhancement research.'),
('tenant-local','marketops.options_detail_3m','marketops','market_data','options_contract_detail',92,'dry_run','derived_options_metadata','Contract-level option detail; distributions and deterministic outputs remain twelve months.'),
('tenant-local','marketops.financial_metadata_4y','marketops','market_data','financial_metadata',1461,'dry_run','immutable_financial_facts','Quarterly normalized facts and valuation snapshots for TTM, restatement lineage, and 16-quarter CAGR.'),
('tenant-local','cyberops.raw_events_30d','cyberops','security','raw_events',30,'dry_run','finding_evidence_receipt','Firewall/Connect payloads expire after compact evidence receipt extraction.'),
('tenant-local','cyberops.high_resolution_30d','cyberops','security','high_resolution_features',30,'dry_run','daily_feature_rollup','Normalized firewall detail and hourly flow features; daily aggregates remain twelve months.'),
('tenant-local','cyberops.metadata_12m','cyberops','security','algorithm_metadata',365,'dry_run','algorithm_ready','Daily aggregates, anomaly outputs, lifecycle episodes, and compact finding evidence.'),
('tenant-local','platform.idempotency_35d','platform','platform','idempotency',35,'dry_run','none','Duplicate-delivery protection beyond the raw grace period.'),
('tenant-local','platform.broker_terminal_7d','platform','platform','broker_terminal',7,'disabled','terminal_only','Broker retention requires a separate broker-administration adapter.')
ON CONFLICT (tenant_id,policy_id) DO UPDATE SET retention_days=EXCLUDED.retention_days, preservation_rule=EXCLUDED.preservation_rule, description=EXCLUDED.description, updated_at=now();
