-- SignalOps TimescaleDB temporal foundation.
-- PostgreSQL remains the relational system of record. TimescaleDB owns
-- replayable time-series ledgers and market-data history.

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS raw_event_ledger (
  event_id text NOT NULL,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  idempotency_key text NOT NULL,
  observation_time timestamptz NOT NULL,
  processing_time timestamptz NOT NULL,
  broker_topic text,
  broker_partition integer,
  broker_offset bigint,
  payload jsonb NOT NULL,
  entity_hints jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (event_id, observation_time)
);
SELECT create_hypertable('raw_event_ledger', 'observation_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_event_id ON raw_event_ledger (event_id, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_source_time ON raw_event_ledger (tenant_id, source_id, source_adapter, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_dataset_time ON raw_event_ledger (dataset, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_idempotency_time ON raw_event_ledger (tenant_id, source_id, idempotency_key, observation_time DESC);

CREATE TABLE IF NOT EXISTS normalized_event_ledger (
  event_id text NOT NULL,
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
  PRIMARY KEY (event_id, observation_time)
);
SELECT create_hypertable('normalized_event_ledger', 'observation_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_normalized_event_event_id ON normalized_event_ledger (event_id, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_normalized_event_source_time ON normalized_event_ledger (tenant_id, source_id, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_normalized_event_dataset_time ON normalized_event_ledger (dataset, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_normalized_event_raw_position_time ON normalized_event_ledger (raw_topic, raw_partition, raw_offset, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_normalized_event_idempotency_time ON normalized_event_ledger (tenant_id, source_id, idempotency_key, observation_time DESC);

CREATE TABLE IF NOT EXISTS signal_ledger (
  signal_id text NOT NULL,
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
  PRIMARY KEY (signal_id, signal_time)
);
SELECT create_hypertable('signal_ledger', 'signal_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_signal_id ON signal_ledger (signal_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_tenant_time ON signal_ledger (tenant_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_source_time ON signal_ledger (tenant_id, source_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_detector_time ON signal_ledger (tenant_id, detector_id, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_severity_time ON signal_ledger (tenant_id, severity, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_broker_position_time ON signal_ledger (broker_topic, broker_partition, broker_offset, signal_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_ledger_event_ids ON signal_ledger USING gin (event_ids);

CREATE TABLE IF NOT EXISTS marketdata_equity_eod_prices (
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  provider text NOT NULL,
  symbol text NOT NULL,
  observation_date date NOT NULL,
  observation_time timestamptz NOT NULL,
  open numeric,
  high numeric,
  low numeric,
  close numeric,
  volume bigint,
  vwap numeric,
  raw_event_id text,
  raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, source_id, provider, symbol, observation_time)
);
SELECT create_hypertable('marketdata_equity_eod_prices', 'observation_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_marketdata_equity_eod_symbol_time ON marketdata_equity_eod_prices (symbol, observation_time DESC);

CREATE TABLE IF NOT EXISTS marketdata_option_contracts_daily (
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  provider text NOT NULL,
  option_ticker text NOT NULL,
  underlying_symbol text NOT NULL,
  contract_type text NOT NULL,
  expiration_date date NOT NULL,
  strike_price numeric NOT NULL,
  observation_date date NOT NULL,
  observation_time timestamptz NOT NULL,
  open numeric,
  high numeric,
  low numeric,
  close numeric,
  volume bigint,
  open_interest bigint,
  vwap numeric,
  raw_event_id text,
  raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, source_id, provider, option_ticker, observation_time)
);
SELECT create_hypertable('marketdata_option_contracts_daily', 'observation_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_marketdata_option_contracts_underlying_time ON marketdata_option_contracts_daily (underlying_symbol, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_marketdata_option_contracts_expiration ON marketdata_option_contracts_daily (expiration_date, underlying_symbol);
