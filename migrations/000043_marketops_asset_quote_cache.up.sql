CREATE TABLE IF NOT EXISTS marketops_asset_quote_cache (
 tenant_id text NOT NULL,
 universe_group text NOT NULL,
 ticker text NOT NULL,
 price double precision NOT NULL,
 quote_timestamp timestamptz NOT NULL,
 market_status text NOT NULL CHECK (market_status IN ('regular','extended','end_of_day')),
 stale boolean NOT NULL DEFAULT false,
 previous_close double precision,
 change_value double precision,
 change_percent double precision,
 week52_low double precision,
 week52_high double precision,
 refreshed_at timestamptz NOT NULL,
 provider text NOT NULL DEFAULT 'massive',
 PRIMARY KEY (tenant_id, universe_group, ticker)
);
CREATE INDEX IF NOT EXISTS marketops_asset_quote_cache_group_idx ON marketops_asset_quote_cache (tenant_id, universe_group, refreshed_at DESC);
INSERT INTO marketops_asset_quote_cache (tenant_id,universe_group,ticker,price,quote_timestamp,market_status,stale,previous_close,change_percent,week52_low,week52_high,refreshed_at,provider)
SELECT DISTINCT ON (tenant_id, universe_group, symbol)
 tenant_id, universe_group, symbol,
 COALESCE((source_payload->>'price')::double precision, 0),
 COALESCE(NULLIF(source_payload->>'quote_timestamp','')::timestamptz, as_of_time),
 'end_of_day', true,
 NULLIF(source_payload->>'previous_close','')::double precision,
 NULLIF(source_payload->>'change_percent','')::double precision,
 NULLIF(source_payload->>'week52_low','')::double precision,
 NULLIF(source_payload->>'week52_high','')::double precision,
 as_of_time, 'massive'
FROM marketops_intraday_condition_snapshots
WHERE COALESCE(source_payload->>'price','') <> ''
ORDER BY tenant_id, universe_group, symbol, as_of_time DESC
ON CONFLICT (tenant_id,universe_group,ticker) DO NOTHING;
