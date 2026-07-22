CREATE TABLE IF NOT EXISTS marketops_intraday_condition_snapshots (
  snapshot_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  universe_group text NOT NULL,
  symbol text NOT NULL,
  as_of_time timestamptz NOT NULL,
  market_status text NOT NULL,
  stale boolean NOT NULL DEFAULT false,
  conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
  source_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, universe_group, symbol, as_of_time)
);
CREATE INDEX IF NOT EXISTS marketops_intraday_condition_snapshots_latest_idx ON marketops_intraday_condition_snapshots (tenant_id, universe_group, symbol, as_of_time DESC);
