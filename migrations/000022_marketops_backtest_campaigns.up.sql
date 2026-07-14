CREATE TABLE IF NOT EXISTS marketops_backtest_campaigns (
  campaign_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL DEFAULT '',
  source_adapter text NOT NULL DEFAULT 'market_data.massive',
  detector_id text NOT NULL,
  detector_version text NOT NULL DEFAULT '',
  requested_by text NOT NULL DEFAULT 'operator-local',
  universe_group text NOT NULL DEFAULT '',
  dataset_scope text[] NOT NULL DEFAULT '{}',
  symbols text[] NOT NULL DEFAULT '{}',
  window_start timestamptz NOT NULL,
  window_end timestamptz NOT NULL,
  window_step_days integer NOT NULL DEFAULT 1 CHECK (window_step_days >= 1),
  max_symbols integer NOT NULL DEFAULT 5 CHECK (max_symbols >= 1 AND max_symbols <= 50),
  max_windows integer NOT NULL DEFAULT 5 CHECK (max_windows >= 1 AND max_windows <= 60),
  max_runs integer NOT NULL DEFAULT 25 CHECK (max_runs >= 1 AND max_runs <= 250),
  max_records integer NOT NULL DEFAULT 50 CHECK (max_records >= 1 AND max_records <= 1000),
  batch_size integer NOT NULL DEFAULT 50 CHECK (batch_size >= 1 AND batch_size <= 1000),
  status text NOT NULL CHECK (status IN ('started', 'succeeded', 'failed', 'canceled')),
  child_run_ids text[] NOT NULL DEFAULT '{}',
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text NOT NULL DEFAULT '',
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (window_end > window_start)
);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_campaigns_tenant_time
  ON marketops_backtest_campaigns (tenant_id, app_id, domain, use_case, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_campaigns_status_time
  ON marketops_backtest_campaigns (tenant_id, status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_backtest_campaigns_universe
  ON marketops_backtest_campaigns (tenant_id, universe_group, started_at DESC);
