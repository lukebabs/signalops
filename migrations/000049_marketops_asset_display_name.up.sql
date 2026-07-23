ALTER TABLE marketops_asset_universe
  ADD COLUMN IF NOT EXISTS display_name text NOT NULL DEFAULT '';

ALTER TABLE marketops_asset_universe
  DROP CONSTRAINT IF EXISTS marketops_asset_universe_display_name_length;

ALTER TABLE marketops_asset_universe
  ADD CONSTRAINT marketops_asset_universe_display_name_length
  CHECK (char_length(display_name) <= 120);
