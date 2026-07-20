DROP INDEX IF EXISTS idx_marketops_options_chain_surface_cell;

ALTER TABLE marketops_options_chain_daily
  DROP COLUMN IF EXISTS selection_score,
  DROP COLUMN IF EXISTS selection_policy_version,
  DROP COLUMN IF EXISTS selection_cell,
  DROP COLUMN IF EXISTS shares_per_contract,
  DROP COLUMN IF EXISTS exercise_style,
  DROP COLUMN IF EXISTS quote_timestamp,
  DROP COLUMN IF EXISTS ask,
  DROP COLUMN IF EXISTS bid;
