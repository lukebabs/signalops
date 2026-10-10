-- Current issuer-published ETF holdings snapshots for SRI representation.
CREATE TABLE sri_etf_holdings_snapshots (
 snapshot_id text PRIMARY KEY, tenant_id text NOT NULL, etf_symbol text NOT NULL,
 fund_name text NOT NULL DEFAULT '', effective_date date NOT NULL, retrieved_at timestamptz NOT NULL,
 source text NOT NULL, source_url text NOT NULL, content_hash text NOT NULL,
 holdings_count integer NOT NULL CHECK (holdings_count >= 0), total_weight double precision NOT NULL,
 top_ten_weight double precision NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (tenant_id, etf_symbol, effective_date, source, content_hash)
);
CREATE INDEX idx_sri_etf_holdings_snapshots_latest ON sri_etf_holdings_snapshots (tenant_id, etf_symbol, effective_date DESC, retrieved_at DESC);
CREATE TABLE sri_etf_holdings (
 snapshot_id text NOT NULL REFERENCES sri_etf_holdings_snapshots(snapshot_id) ON DELETE RESTRICT,
 holding_key text NOT NULL, holding_rank integer NOT NULL CHECK (holding_rank > 0), ticker text NOT NULL DEFAULT '',
 name text NOT NULL, identifier text NOT NULL DEFAULT '', sedol text NOT NULL DEFAULT '', sector text NOT NULL DEFAULT '',
 currency text NOT NULL DEFAULT '', weight double precision NOT NULL, shares_held double precision NOT NULL,
 PRIMARY KEY (snapshot_id, holding_key), UNIQUE (snapshot_id, holding_rank)
);
