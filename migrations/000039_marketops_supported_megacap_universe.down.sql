-- Restore the original five exchange listings and original rank ordering.

UPDATE marketops_asset_universe
SET rank = rank + 200,
    is_active = false,
    updated_at = now()
WHERE tenant_id = 'tenant-local'
  AND universe_group = 'top50_megacap';

DELETE FROM marketops_asset_universe
WHERE tenant_id = 'tenant-local'
  AND universe_group = 'top50_megacap'
  AND ticker IN ('PM', 'RY', 'BABA', 'NVS', 'PANW');

WITH desired(ticker, rank) AS (
  VALUES
    ('NVDA', 1), ('AAPL', 2), ('GOOGL', 3), ('MSFT', 4), ('AMZN', 5),
    ('TSM', 6), ('SPCX', 7), ('AVGO', 8), ('TSLA', 10), ('META', 11),
    ('MU', 13), ('BRK.B', 14), ('LLY', 15), ('JPM', 17), ('AMD', 18),
    ('WMT', 19), ('ASML', 20), ('V', 21), ('JNJ', 22), ('INTC', 23),
    ('XOM', 24), ('TCEHY', 25), ('MA', 26), ('AMAT', 27), ('ABBV', 28),
    ('CSCO', 29), ('CAT', 30), ('LRCX', 31), ('BAC', 32), ('COST', 33),
    ('ORCL', 34), ('GE', 35), ('UNH', 36), ('KO', 38), ('MS', 39),
    ('HD', 40), ('PG', 41), ('ARM', 42), ('HSBC', 43), ('CVX', 44),
    ('NFLX', 46), ('PLTR', 47), ('MRK', 48), ('GS', 49), ('GEV', 50)
)
UPDATE marketops_asset_universe AS asset
SET rank = desired.rank,
    is_active = true,
    metadata = asset.metadata - 'universe_revision',
    updated_at = now()
FROM desired
WHERE asset.tenant_id = 'tenant-local'
  AND asset.universe_group = 'top50_megacap'
  AND asset.ticker = desired.ticker;

INSERT INTO marketops_asset_universe (
  tenant_id, app_id, domain, use_case, source_id, universe_group, rank, ticker, ticker_key,
  company, company_key, asset_type, exchange, sector, sector_key, industry, industry_key, metadata
) VALUES
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 9, '2222.SR', '2222_sr', 'Saudi Aramco', 'saudi_aramco', 'equity', '', 'Energy', 'energy', 'Oil & Gas', 'oil_gas', '{"seed":"top50megacap.normalized.csv","provider":"massive"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 12, '005930.KS', '005930_ks', 'Samsung Electronics', 'samsung_electronics', 'equity', '', 'Technology Hardware', 'technology_hardware', '', '', '{"seed":"top50megacap.normalized.csv","provider":"massive"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 16, '000660.KS', '000660_ks', 'SK Hynix', 'sk_hynix', 'equity', '', 'Technology', 'technology', 'Semiconductors', 'semiconductors', '{"seed":"top50megacap.normalized.csv","provider":"massive"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 37, '601939.SS', '601939_ss', 'China Construction Bank', 'china_construction_bank', 'equity', '', 'Financials', 'financials', 'Banking', 'banking', '{"seed":"top50megacap.normalized.csv","provider":"massive"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 45, 'RO.SW', 'ro_sw', 'Roche', 'roche', 'equity', '', 'Healthcare', 'healthcare', '', '', '{"seed":"top50megacap.normalized.csv","provider":"massive"}'::jsonb);

DO $$
DECLARE
  active_count integer;
BEGIN
  SELECT count(*) INTO active_count
  FROM marketops_asset_universe
  WHERE tenant_id = 'tenant-local'
    AND universe_group = 'top50_megacap'
    AND is_active;

  IF active_count <> 50 THEN
    RAISE EXCEPTION 'expected 50 restored top50_megacap assets, found %', active_count;
  END IF;
END
$$;
