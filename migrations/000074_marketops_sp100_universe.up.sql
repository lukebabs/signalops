-- S&P 100 public constituent reference captured 2026-08-02.
-- The index commonly contains 101 listed holdings because Alphabet has two
-- constituent share classes. Source metadata records the public reference;
-- S&P Dow Jones remains the authoritative index operator.
DROP VIEW IF EXISTS marketops_universal_assets;
CREATE VIEW marketops_universal_assets AS
SELECT DISTINCT ON (tenant_id, ticker) *
FROM (
  SELECT marketops_asset_universe.*,
    CASE universe_group
      WHEN 'top50_megacap' THEN 1
      WHEN 'analyst_watchlist' THEN 2
      WHEN 'sp100' THEN 3
      ELSE 99
    END AS universe_priority
  FROM marketops_asset_universe
  WHERE universe_group IN ('top50_megacap', 'analyst_watchlist', 'sp100')
    AND is_active = true
) ranked
ORDER BY tenant_id, ticker, universe_priority, rank;

WITH seed(rank, ticker, company, sector) AS (
  VALUES
  (1, 'AAPL', 'Apple Inc.', 'Information Technology'),
  (2, 'ABBV', 'AbbVie', 'Health Care'),
  (3, 'ABT', 'Abbott Laboratories', 'Health Care'),
  (4, 'ACN', 'Accenture', 'Information Technology'),
  (5, 'ADBE', 'Adobe Inc.', 'Information Technology'),
  (6, 'AMAT', 'Applied Materials', 'Information Technology'),
  (7, 'AMD', 'Advanced Micro Devices', 'Information Technology'),
  (8, 'AMGN', 'Amgen', 'Health Care'),
  (9, 'AMT', 'American Tower', 'Real Estate'),
  (10, 'AMZN', 'Amazon', 'Consumer Discretionary'),
  (11, 'AVGO', 'Broadcom', 'Information Technology'),
  (12, 'AXP', 'American Express', 'Financials'),
  (13, 'BA', 'Boeing', 'Industrials'),
  (14, 'BAC', 'Bank of America', 'Financials'),
  (15, 'BKNG', 'Booking Holdings', 'Consumer Discretionary'),
  (16, 'BLK', 'BlackRock', 'Financials'),
  (17, 'BMY', 'Bristol Myers Squibb', 'Health Care'),
  (18, 'BNY', 'BNY Mellon', 'Financials'),
  (19, 'BRK.B', 'Berkshire Hathaway (Class B)', 'Financials'),
  (20, 'C', 'Citigroup', 'Financials'),
  (21, 'CAT', 'Caterpillar Inc.', 'Industrials'),
  (22, 'CL', 'Colgate-Palmolive', 'Consumer Staples'),
  (23, 'CMCSA', 'Comcast', 'Communication Services'),
  (24, 'COF', 'Capital One', 'Financials'),
  (25, 'COP', 'ConocoPhillips', 'Energy'),
  (26, 'COST', 'Costco', 'Consumer Staples'),
  (27, 'CRM', 'Salesforce', 'Information Technology'),
  (28, 'CSCO', 'Cisco', 'Information Technology'),
  (29, 'CVS', 'CVS Health', 'Health Care'),
  (30, 'CVX', 'Chevron Corporation', 'Energy'),
  (31, 'DE', 'Deere & Company', 'Industrials'),
  (32, 'DHR', 'Danaher Corporation', 'Health Care'),
  (33, 'DIS', 'Walt Disney Company (The)', 'Communication Services'),
  (34, 'DUK', 'Duke Energy', 'Utilities'),
  (35, 'EMR', 'Emerson Electric', 'Industrials'),
  (36, 'FDX', 'FedEx', 'Industrials'),
  (37, 'GD', 'General Dynamics', 'Industrials'),
  (38, 'GE', 'GE Aerospace', 'Industrials'),
  (39, 'GEV', 'GE Vernova', 'Industrials'),
  (40, 'GILD', 'Gilead Sciences', 'Health Care'),
  (41, 'GM', 'General Motors', 'Consumer Discretionary'),
  (42, 'GOOG', 'Alphabet Inc. (Class C)', 'Communication Services'),
  (43, 'GOOGL', 'Alphabet Inc. (Class A)', 'Communication Services'),
  (44, 'GS', 'Goldman Sachs', 'Financials'),
  (45, 'HD', 'Home Depot', 'Consumer Discretionary'),
  (46, 'HONA', 'Honeywell Aerospace', 'Industrials'),
  (47, 'IBM', 'IBM', 'Information Technology'),
  (48, 'INTC', 'Intel', 'Information Technology'),
  (49, 'INTU', 'Intuit', 'Information Technology'),
  (50, 'ISRG', 'Intuitive Surgical', 'Health Care'),
  (51, 'JNJ', 'Johnson & Johnson', 'Health Care'),
  (52, 'JPM', 'JPMorgan Chase', 'Financials'),
  (53, 'KO', 'Coca-Cola Company (The)', 'Consumer Staples'),
  (54, 'LIN', 'Linde plc', 'Materials'),
  (55, 'LLY', 'Eli Lilly and Company', 'Health Care'),
  (56, 'LMT', 'Lockheed Martin', 'Industrials'),
  (57, 'LOW', 'Lowe''s', 'Consumer Discretionary'),
  (58, 'LRCX', 'Lam Research', 'Information Technology'),
  (59, 'MA', 'Mastercard', 'Financials'),
  (60, 'MCD', 'McDonald''s', 'Consumer Discretionary'),
  (61, 'MDLZ', 'Mondelēz International', 'Consumer Staples'),
  (62, 'MDT', 'Medtronic', 'Health Care'),
  (63, 'META', 'Meta Platforms', 'Communication Services'),
  (64, 'MMM', '3M', 'Industrials'),
  (65, 'MO', 'Altria', 'Consumer Staples'),
  (66, 'MRK', 'Merck & Co.', 'Health Care'),
  (67, 'MS', 'Morgan Stanley', 'Financials'),
  (68, 'MSFT', 'Microsoft', 'Information Technology'),
  (69, 'MU', 'Micron Technology', 'Information Technology'),
  (70, 'NEE', 'NextEra Energy', 'Utilities'),
  (71, 'NFLX', 'Netflix, Inc.', 'Communication Services'),
  (72, 'NKE', 'Nike, Inc.', 'Consumer Discretionary'),
  (73, 'NOW', 'ServiceNow', 'Information Technology'),
  (74, 'NVDA', 'Nvidia', 'Information Technology'),
  (75, 'ORCL', 'Oracle Corporation', 'Information Technology'),
  (76, 'PEP', 'PepsiCo', 'Consumer Staples'),
  (77, 'PFE', 'Pfizer', 'Health Care'),
  (78, 'PG', 'Procter & Gamble', 'Consumer Staples'),
  (79, 'PLTR', 'Palantir Technologies', 'Information Technology'),
  (80, 'PM', 'Philip Morris International', 'Consumer Staples'),
  (81, 'QCOM', 'Qualcomm', 'Information Technology'),
  (82, 'RTX', 'RTX Corporation', 'Industrials'),
  (83, 'SBUX', 'Starbucks', 'Consumer Discretionary'),
  (84, 'SCHW', 'Charles Schwab Corporation', 'Financials'),
  (85, 'SO', 'Southern Company', 'Utilities'),
  (86, 'SPG', 'Simon Property Group', 'Real Estate'),
  (87, 'T', 'AT&T', 'Communication Services'),
  (88, 'TMO', 'Thermo Fisher Scientific', 'Health Care'),
  (89, 'TMUS', 'T-Mobile US', 'Communication Services'),
  (90, 'TSLA', 'Tesla, Inc.', 'Consumer Discretionary'),
  (91, 'TXN', 'Texas Instruments', 'Information Technology'),
  (92, 'UBER', 'Uber', 'Industrials'),
  (93, 'UNH', 'UnitedHealth Group', 'Health Care'),
  (94, 'UNP', 'Union Pacific Corporation', 'Industrials'),
  (95, 'UPS', 'United Parcel Service', 'Industrials'),
  (96, 'USB', 'U.S. Bancorp', 'Financials'),
  (97, 'V', 'Visa Inc.', 'Financials'),
  (98, 'VZ', 'Verizon', 'Communication Services'),
  (99, 'WFC', 'Wells Fargo', 'Financials'),
  (100, 'WMT', 'Walmart', 'Consumer Staples'),
  (101, 'XOM', 'ExxonMobil', 'Energy')
)
INSERT INTO marketops_asset_universe (
  tenant_id, app_id, domain, use_case, source_id, universe_group, rank, ticker, ticker_key,
  company, company_key, asset_type, exchange, sector, sector_key, industry, industry_key,
  is_active, metadata
)
SELECT
  'tenant-local', 'marketops', 'market_data', 'daily_market_surveillance', 'src-spglobal', 'sp100',
  rank, ticker, lower(regexp_replace(ticker, '[^A-Za-z0-9]+', '_', 'g')),
  company, lower(regexp_replace(company, '[^A-Za-z0-9]+', '_', 'g')),
  'equity', '', sector, lower(regexp_replace(sector, '[^A-Za-z0-9]+', '_', 'g')), '', '', true,
  jsonb_build_object(
    'origin', 'sp100_public_reference',
    'index', 'S&P 100',
    'constituent_count', 101,
    'source_url', 'https://www.spglobal.com/spdji/en/indices/equity/sp-100/',
    'reference_url', 'https://en.wikipedia.org/wiki/S%26P_100',
    'source_captured_at', '2026-08-02'
  )
FROM seed
ON CONFLICT (tenant_id, universe_group, ticker) DO UPDATE SET
  app_id = EXCLUDED.app_id,
  domain = EXCLUDED.domain,
  use_case = EXCLUDED.use_case,
  source_id = EXCLUDED.source_id,
  rank = EXCLUDED.rank,
  ticker_key = EXCLUDED.ticker_key,
  company = EXCLUDED.company,
  company_key = EXCLUDED.company_key,
  asset_type = EXCLUDED.asset_type,
  exchange = EXCLUDED.exchange,
  sector = EXCLUDED.sector,
  sector_key = EXCLUDED.sector_key,
  industry = EXCLUDED.industry,
  industry_key = EXCLUDED.industry_key,
  is_active = true,
  metadata = EXCLUDED.metadata,
  updated_at = now();
