-- SignalOps storage foundation.
-- PostgreSQL owns operational metadata, run audit, provider usage, idempotency,
-- and initial market-data snapshots. Timescale hypertables can be added later.

CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scheduler_runs (
  run_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  datasets text[] NOT NULL DEFAULT '{}',
  observation_date date NOT NULL,
  dry_run boolean NOT NULL,
  status text NOT NULL CHECK (status IN ('started', 'succeeded', 'failed', 'canceled')),
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  events_built integer NOT NULL DEFAULT 0 CHECK (events_built >= 0),
  events_published integer NOT NULL DEFAULT 0 CHECK (events_published >= 0),
  provider_requests integer NOT NULL DEFAULT 0 CHECK (provider_requests >= 0),
  provider_retries integer NOT NULL DEFAULT 0 CHECK (provider_retries >= 0),
  failures integer NOT NULL DEFAULT 0 CHECK (failures >= 0),
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  report jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scheduler_runs_started_at ON scheduler_runs (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduler_runs_source ON scheduler_runs (tenant_id, source_id, source_adapter, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduler_runs_status ON scheduler_runs (status, started_at DESC);

CREATE TABLE IF NOT EXISTS provider_usage_runs (
  usage_id text PRIMARY KEY,
  run_id text NOT NULL REFERENCES scheduler_runs(run_id) ON DELETE CASCADE,
  provider text NOT NULL,
  dataset text NOT NULL,
  request_count integer NOT NULL DEFAULT 0 CHECK (request_count >= 0),
  retry_count integer NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
  event_count integer NOT NULL DEFAULT 0 CHECK (event_count >= 0),
  budget jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_provider_usage_runs_run ON provider_usage_runs (run_id);
CREATE INDEX IF NOT EXISTS idx_provider_usage_runs_provider_dataset ON provider_usage_runs (provider, dataset, created_at DESC);

CREATE TABLE IF NOT EXISTS idempotency_records (
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  idempotency_key text NOT NULL,
  event_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  topic text,
  partition integer,
  offset_value bigint,
  payload_hash text,
  status text NOT NULL CHECK (status IN ('accepted', 'published', 'processed', 'failed', 'duplicate')),
  first_seen_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (tenant_id, source_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_records_event ON idempotency_records (event_id);
CREATE INDEX IF NOT EXISTS idx_idempotency_records_status ON idempotency_records (status, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS raw_event_ledger (
  event_id text PRIMARY KEY,
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
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_source_time ON raw_event_ledger (tenant_id, source_id, source_adapter, observation_time DESC);
CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_dataset_time ON raw_event_ledger (dataset, observation_time DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_raw_event_ledger_idempotency ON raw_event_ledger (tenant_id, source_id, idempotency_key);

CREATE TABLE IF NOT EXISTS marketdata_equity_eod_prices (
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  provider text NOT NULL,
  symbol text NOT NULL,
  observation_date date NOT NULL,
  open numeric,
  high numeric,
  low numeric,
  close numeric,
  volume bigint,
  vwap numeric,
  raw_event_id text REFERENCES raw_event_ledger(event_id) ON DELETE SET NULL,
  raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, source_id, provider, symbol, observation_date)
);

CREATE INDEX IF NOT EXISTS idx_marketdata_equity_eod_symbol_date ON marketdata_equity_eod_prices (symbol, observation_date DESC);

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
  open numeric,
  high numeric,
  low numeric,
  close numeric,
  volume bigint,
  open_interest bigint,
  vwap numeric,
  raw_event_id text REFERENCES raw_event_ledger(event_id) ON DELETE SET NULL,
  raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, source_id, provider, option_ticker, observation_date)
);

CREATE INDEX IF NOT EXISTS idx_marketdata_option_contracts_underlying_date ON marketdata_option_contracts_daily (underlying_symbol, observation_date DESC);
CREATE INDEX IF NOT EXISTS idx_marketdata_option_contracts_expiration ON marketdata_option_contracts_daily (expiration_date, underlying_symbol);
