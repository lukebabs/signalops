CREATE TABLE IF NOT EXISTS marketops_asset_backfill_jobs (
  backfill_job_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  universe_group text NOT NULL DEFAULT 'analyst_watchlist' CHECK (universe_group = 'analyst_watchlist'),
  start_date date NOT NULL,
  end_date date NOT NULL,
  status text NOT NULL CHECK (status IN ('queued', 'running', 'awaiting_normalization', 'succeeded', 'partial', 'failed')),
  requested_by text NOT NULL,
  requested_sessions integer NOT NULL DEFAULT 0 CHECK (requested_sessions >= 0),
  completed_sessions integer NOT NULL DEFAULT 0 CHECK (completed_sessions >= 0),
  failed_sessions integer NOT NULL DEFAULT 0 CHECK (failed_sessions >= 0),
  provider_requests integer NOT NULL DEFAULT 0 CHECK (provider_requests >= 0),
  error_message text,
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (end_date >= start_date)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_marketops_asset_backfill_active ON marketops_asset_backfill_jobs (tenant_id, symbol) WHERE status IN ('queued', 'running', 'awaiting_normalization');
CREATE INDEX IF NOT EXISTS idx_marketops_asset_backfill_tenant_created ON marketops_asset_backfill_jobs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_asset_backfill_due ON marketops_asset_backfill_jobs (status, created_at) WHERE status IN ('queued', 'running', 'awaiting_normalization');
