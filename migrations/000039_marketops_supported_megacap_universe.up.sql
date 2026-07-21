-- Replace five Massive-incompatible exchange listings with the next five
-- provider-validated megacaps while preserving a 50-asset active universe.

UPDATE marketops_asset_universe
SET rank = rank + 100,
    is_active = false,
    updated_at = now()
WHERE tenant_id = 'tenant-local'
  AND universe_group = 'top50_megacap';

DELETE FROM marketops_asset_universe
WHERE tenant_id = 'tenant-local'
  AND universe_group = 'top50_megacap'
  AND ticker IN ('2222.SR', '005930.KS', '000660.KS', '601939.SS', 'RO.SW');

WITH desired(ticker, rank) AS (
  VALUES
    ('NVDA', 1), ('AAPL', 2), ('GOOGL', 3), ('MSFT', 4), ('AMZN', 5),
    ('TSM', 6), ('SPCX', 7), ('AVGO', 8), ('TSLA', 9), ('META', 10),
    ('MU', 11), ('BRK.B', 12), ('LLY', 13), ('JPM', 14), ('AMD', 15),
    ('WMT', 16), ('ASML', 17), ('V', 18), ('JNJ', 19), ('INTC', 20),
    ('XOM', 21), ('TCEHY', 22), ('MA', 23), ('AMAT', 24), ('ABBV', 25),
    ('CSCO', 26), ('CAT', 27), ('LRCX', 28), ('BAC', 29), ('COST', 30),
    ('ORCL', 31), ('GE', 32), ('UNH', 33), ('KO', 34), ('MS', 35),
    ('HD', 36), ('PG', 37), ('ARM', 38), ('HSBC', 39), ('CVX', 40),
    ('NFLX', 41), ('PLTR', 42), ('MRK', 43), ('GS', 44), ('GEV', 45)
)
UPDATE marketops_asset_universe AS asset
SET rank = desired.rank,
    is_active = true,
    metadata = asset.metadata || '{"universe_revision":"2026-07-21-provider-supported-top50"}'::jsonb,
    updated_at = now()
FROM desired
WHERE asset.tenant_id = 'tenant-local'
  AND asset.universe_group = 'top50_megacap'
  AND asset.ticker = desired.ticker;

INSERT INTO marketops_asset_universe (
  tenant_id, app_id, domain, use_case, source_id, universe_group, rank, ticker, ticker_key,
  company, company_key, asset_type, exchange, sector, sector_key, industry, industry_key, metadata
) VALUES
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 46, 'PM', 'pm', 'Philip Morris International', 'philip_morris_international', 'equity', '', 'Consumer Staples', 'consumer_staples', 'Tobacco', 'tobacco', '{"seed":"top50megacap.normalized.csv","provider":"massive","selection_source":"companiesmarketcap.com","selection_date":"2026-07-21","provider_validation_date":"2026-07-20","universe_revision":"2026-07-21-provider-supported-top50"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 47, 'RY', 'ry', 'Royal Bank of Canada', 'royal_bank_of_canada', 'equity', '', 'Financials', 'financials', 'Banking', 'banking', '{"seed":"top50megacap.normalized.csv","provider":"massive","selection_source":"companiesmarketcap.com","selection_date":"2026-07-21","provider_validation_date":"2026-07-20","universe_revision":"2026-07-21-provider-supported-top50"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 48, 'BABA', 'baba', 'Alibaba', 'alibaba', 'equity', '', 'Consumer Discretionary', 'consumer_discretionary', 'E-Commerce', 'e_commerce', '{"seed":"top50megacap.normalized.csv","provider":"massive","selection_source":"companiesmarketcap.com","selection_date":"2026-07-21","provider_validation_date":"2026-07-20","universe_revision":"2026-07-21-provider-supported-top50"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 49, 'NVS', 'nvs', 'Novartis', 'novartis', 'equity', '', 'Healthcare', 'healthcare', 'Pharmaceuticals', 'pharmaceuticals', '{"seed":"top50megacap.normalized.csv","provider":"massive","selection_source":"companiesmarketcap.com","selection_date":"2026-07-21","provider_validation_date":"2026-07-20","universe_revision":"2026-07-21-provider-supported-top50"}'::jsonb),
  ('tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-massive', 'top50_megacap', 50, 'PANW', 'panw', 'Palo Alto Networks', 'palo_alto_networks', 'equity', '', 'Technology', 'technology', 'Cybersecurity', 'cybersecurity', '{"seed":"top50megacap.normalized.csv","provider":"massive","selection_source":"companiesmarketcap.com","selection_date":"2026-07-21","provider_validation_date":"2026-07-20","universe_revision":"2026-07-21-provider-supported-top50"}'::jsonb)
ON CONFLICT (tenant_id, universe_group, ticker) DO UPDATE SET
  rank = EXCLUDED.rank,
  ticker_key = EXCLUDED.ticker_key,
  company = EXCLUDED.company,
  company_key = EXCLUDED.company_key,
  sector = EXCLUDED.sector,
  sector_key = EXCLUDED.sector_key,
  industry = EXCLUDED.industry,
  industry_key = EXCLUDED.industry_key,
  is_active = true,
  metadata = EXCLUDED.metadata,
  updated_at = now();

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
    RAISE EXCEPTION 'expected 50 active top50_megacap assets, found %', active_count;
  END IF;
END
$$;
