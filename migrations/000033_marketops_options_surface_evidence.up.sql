ALTER TABLE marketops_options_chain_daily
  ADD COLUMN IF NOT EXISTS bid double precision,
  ADD COLUMN IF NOT EXISTS ask double precision,
  ADD COLUMN IF NOT EXISTS quote_timestamp timestamptz,
  ADD COLUMN IF NOT EXISTS exercise_style text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS shares_per_contract bigint,
  ADD COLUMN IF NOT EXISTS selection_cell text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS selection_policy_version text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS selection_score double precision;

CREATE INDEX IF NOT EXISTS idx_marketops_options_chain_surface_cell
  ON marketops_options_chain_daily (tenant_id, symbol, trade_date DESC, selection_cell)
  WHERE selection_cell <> '';
