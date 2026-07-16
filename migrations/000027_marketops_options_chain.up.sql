CREATE TABLE IF NOT EXISTS marketops_options_chain_daily (
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  trade_date date NOT NULL,
  option_ticker text NOT NULL,
  provider text NOT NULL DEFAULT 'massive',
  source_id text NOT NULL DEFAULT '',
  ingestion_run_id text NOT NULL DEFAULT '',
  contract_type text NOT NULL CHECK (contract_type IN ('call', 'put')),
  expiration_date date NOT NULL,
  strike_price double precision NOT NULL CHECK (strike_price > 0),
  underlying_close double precision,
  moneyness double precision,
  open double precision,
  high double precision,
  low double precision,
  close double precision,
  vwap double precision,
  volume bigint,
  open_interest bigint,
  implied_volatility double precision,
  delta double precision,
  gamma double precision,
  theta double precision,
  vega double precision,
  provider_request_id text NOT NULL DEFAULT '',
  payload_hash text NOT NULL DEFAULT '',
  raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, symbol, trade_date, option_ticker)
);

CREATE INDEX IF NOT EXISTS idx_marketops_options_chain_symbol_date
  ON marketops_options_chain_daily (tenant_id, symbol, trade_date DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_options_chain_expiration_strike
  ON marketops_options_chain_daily (tenant_id, symbol, expiration_date, strike_price);

CREATE TABLE IF NOT EXISTS marketops_options_distribution_daily (
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  trade_date date NOT NULL,
  window_name text NOT NULL DEFAULT '10_trade_days',
  source_id text NOT NULL DEFAULT '',
  provider text NOT NULL DEFAULT 'massive',
  trade_days integer NOT NULL DEFAULT 0,
  contract_count integer NOT NULL DEFAULT 0,
  call_contract_count integer NOT NULL DEFAULT 0,
  put_contract_count integer NOT NULL DEFAULT 0,
  total_call_open_interest bigint NOT NULL DEFAULT 0,
  total_put_open_interest bigint NOT NULL DEFAULT 0,
  total_call_volume bigint NOT NULL DEFAULT 0,
  total_put_volume bigint NOT NULL DEFAULT 0,
  missing_open_interest_count integer NOT NULL DEFAULT 0,
  call_put_open_interest_ratio double precision NOT NULL DEFAULT 0,
  call_put_volume_ratio double precision NOT NULL DEFAULT 0,
  ratio_delta double precision NOT NULL DEFAULT 0,
  ratio_change_pct double precision NOT NULL DEFAULT 0,
  ratio_zscore double precision NOT NULL DEFAULT 0,
  change_point_score double precision NOT NULL DEFAULT 0,
  confidence double precision NOT NULL DEFAULT 0,
  moneyness_distribution jsonb NOT NULL DEFAULT '{}'::jsonb,
  expiration_distribution jsonb NOT NULL DEFAULT '{}'::jsonb,
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  source_trade_dates date[] NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, symbol, trade_date, window_name)
);

CREATE INDEX IF NOT EXISTS idx_marketops_options_distribution_symbol_date
  ON marketops_options_distribution_daily (tenant_id, symbol, trade_date DESC);
