CREATE TABLE IF NOT EXISTS marketops_financial_statements (
  statement_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  statement_type text NOT NULL,
  fiscal_period_end date NOT NULL,
  accepted_at timestamptz NOT NULL,
  fiscal_period text NOT NULL DEFAULT '',
  normalized_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  raw_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,symbol,statement_type,fiscal_period_end,accepted_at)
);
CREATE INDEX IF NOT EXISTS idx_marketops_financial_statements_lookup ON marketops_financial_statements (tenant_id,symbol,statement_type,fiscal_period_end DESC,accepted_at DESC);

CREATE TABLE IF NOT EXISTS marketops_financial_snapshots (
  financial_snapshot_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  snapshot_version text NOT NULL,
  evaluation_date date NOT NULL,
  available_at timestamptz NOT NULL,
  statement_ids text[] NOT NULL DEFAULT '{}',
  input_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  derived_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id,symbol,snapshot_version,evaluation_date,available_at)
);
CREATE INDEX IF NOT EXISTS idx_marketops_financial_snapshots_lookup ON marketops_financial_snapshots (tenant_id,symbol,created_at DESC);
