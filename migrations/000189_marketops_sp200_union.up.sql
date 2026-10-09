-- Promote the governed ranked S&P-200 selection into the tenant-local
-- operational universe without deleting or replacing the preserved legacy
-- cohort. The current ranking contains 200 rows, of which only eligible
-- assets are admitted to the provider-backed operational surface.

DO $preflight$
DECLARE
  ranked_count integer;
  eligible_count integer;
BEGIN
  SELECT count(DISTINCT entry.global_asset_id),
         count(DISTINCT entry.global_asset_id) FILTER (WHERE asset.eligibility_status = 'eligible')
    INTO ranked_count, eligible_count
  FROM subscriber_global_ranking_snapshot_entries entry
  JOIN subscriber_global_ranking_snapshots snapshot
    ON snapshot.ranking_snapshot_id = entry.ranking_snapshot_id
   AND snapshot.is_current
  JOIN subscriber_global_assets asset ON asset.global_asset_id = entry.global_asset_id
  WHERE entry.source_rank <= 200;

  IF ranked_count < 200 THEN
    RAISE EXCEPTION 'S&P 200 promotion requires a current ranked cohort of at least 200 assets; found %', ranked_count;
  END IF;
  IF eligible_count < 1 THEN
    RAISE EXCEPTION 'S&P 200 promotion found no eligible assets';
  END IF;
END;
$preflight$;

-- Keep the canonical MarketOps view as the single source for scheduled jobs,
-- now including the governed S&P-200 projection at the lowest priority. The
-- existing top50/analyst/S&P100 rows remain preferred when symbols overlap.
CREATE OR REPLACE VIEW marketops_universal_assets AS
SELECT DISTINCT ON (tenant_id, ticker) *
FROM (
  SELECT marketops_asset_universe.*,
    CASE universe_group
      WHEN 'top50_megacap' THEN 1
      WHEN 'analyst_watchlist' THEN 2
      WHEN 'sp100' THEN 3
      WHEN 'snp500_top200' THEN 4
      ELSE 99
    END AS universe_priority
  FROM marketops_asset_universe
  WHERE universe_group IN ('top50_megacap', 'analyst_watchlist', 'sp100', 'snp500_top200')
    AND is_active = true
) ranked
ORDER BY tenant_id, ticker, universe_priority, rank;

WITH ranked AS (
  SELECT DISTINCT ON (asset.global_asset_id)
    asset.global_asset_id,
    asset.canonical_symbol,
    asset.company_name,
    asset.asset_type,
    asset.exchange,
    asset.sector,
    asset.industry,
    entry.source_rank,
    row_number() OVER (ORDER BY entry.source_rank, asset.canonical_symbol, asset.global_asset_id) AS operational_rank
  FROM subscriber_global_ranking_snapshot_entries entry
  JOIN subscriber_global_ranking_snapshots snapshot
    ON snapshot.ranking_snapshot_id = entry.ranking_snapshot_id
   AND snapshot.is_current
  JOIN subscriber_global_assets asset ON asset.global_asset_id = entry.global_asset_id
  WHERE entry.source_rank <= 200
    AND asset.eligibility_status = 'eligible'
  ORDER BY asset.global_asset_id, entry.source_rank
)
INSERT INTO marketops_asset_universe (
  tenant_id, app_id, domain, use_case, source_id, universe_group, rank,
  ticker, ticker_key, company, company_key, asset_type, exchange, sector,
  sector_key, industry, industry_key, is_active, metadata
)
SELECT
  'tenant-local', 'marketops', 'market_data', 'daily_market_surveillance',
  'src-subscriber-global-ranking', 'snp500_top200', ranked.operational_rank,
  ranked.canonical_symbol,
  lower(regexp_replace(ranked.canonical_symbol, '[^A-Za-z0-9]+', '_', 'g')),
  ranked.company_name,
  lower(regexp_replace(ranked.company_name, '[^A-Za-z0-9]+', '_', 'g')),
  COALESCE(NULLIF(ranked.asset_type, ''), 'equity'), COALESCE(ranked.exchange, ''),
  COALESCE(ranked.sector, ''), lower(regexp_replace(COALESCE(ranked.sector, ''), '[^A-Za-z0-9]+', '_', 'g')),
  COALESCE(ranked.industry, ''), lower(regexp_replace(COALESCE(ranked.industry, ''), '[^A-Za-z0-9]+', '_', 'g')),
  true,
  jsonb_build_object(
    'selection', 'governed_ranked_snp500_top200',
    'ranking_source_rank', ranked.source_rank,
    'eligibility_policy', 'eligible_only',
    'captured_at', now()
  )
FROM ranked
ON CONFLICT (tenant_id, universe_group, ticker) DO UPDATE SET
  rank = EXCLUDED.rank,
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

-- Expand the tenant-local default list by unioning the eligible ranked cohort.
-- Existing legacy memberships are retained verbatim, preserving historical
-- SAF evidence and its immutable baseline.
WITH ranked AS (
  SELECT DISTINCT ON (asset.global_asset_id) asset.global_asset_id, entry.source_rank
  FROM subscriber_global_ranking_snapshot_entries entry
  JOIN subscriber_global_ranking_snapshots snapshot
    ON snapshot.ranking_snapshot_id = entry.ranking_snapshot_id
   AND snapshot.is_current
  JOIN subscriber_global_assets asset ON asset.global_asset_id = entry.global_asset_id
  WHERE entry.source_rank <= 200 AND asset.eligibility_status = 'eligible'
  ORDER BY asset.global_asset_id, entry.source_rank
), default_list AS (
  SELECT list_id
  FROM subscriber_watchlists
  WHERE tenant_id = 'tenant-local' AND list_kind = 'tenant_default'
), added AS (
  INSERT INTO subscriber_watchlist_memberships
    (tenant_id, list_id, global_asset_id, added_by_subject, provenance)
  SELECT 'tenant-local', default_list.list_id, ranked.global_asset_id,
    'subscriber-sp200-union-000189',
    jsonb_build_object(
      'schema_version', 'subscriber.watchlist.sp200-union.v1',
      'selection', 'governed_ranked_snp500_top200',
      'ranking_source_rank', ranked.source_rank,
      'preservation_policy', 'legacy-memberships-retained'
    )
  FROM ranked CROSS JOIN default_list
  ON CONFLICT (list_id, global_asset_id) DO NOTHING
  RETURNING list_id, global_asset_id
)
INSERT INTO subscriber_watchlist_audit
  (audit_id, tenant_id, list_id, actor_subject, mutation, global_asset_id,
   before_value, after_value, correlation_id)
SELECT 'sublistaudit-' || md5('sp200-union-000189:' || added.list_id || ':' || added.global_asset_id),
  'tenant-local', added.list_id, 'subscriber-sp200-union-000189', 'add_asset', added.global_asset_id,
  '{}'::jsonb,
  jsonb_build_object('promotion', 'operational_primary', 'selection', 'snp500_top200'),
  'subscriber-sp200-union-000189'
FROM added
ON CONFLICT (audit_id) DO NOTHING;

-- Queue global EOD activation for newly selected assets. Existing active rows
-- are idempotently reconciled and no provider request is made by this migration.
INSERT INTO subscriber_global_coverage_activation_requests
  (activation_request_id, global_asset_id, request_key, request_state,
   request_reason, requester_kind, requester_tenant_id, requester_subject,
   requester_list_id, policy_version, provenance, requested_at)
SELECT 'subactivation-' || md5('tenant-local:' || membership.global_asset_id),
  membership.global_asset_id,
  'subscriber-tenant-default-hot-v1:tenant-local:' || membership.global_asset_id,
  'queued', 'subscriber_tenant_default_sp200_union', 'tenant_default_list',
  'tenant-local', 'subscriber-sp200-union-000189', membership.list_id,
  'subscriber-tenant-default-sp200-v1',
  jsonb_build_object('surface', 'subscriber.sp200.union', 'migration', '000189'), now()
FROM subscriber_watchlist_memberships membership
JOIN subscriber_watchlists list ON list.list_id = membership.list_id
WHERE membership.tenant_id = 'tenant-local' AND list.list_kind = 'tenant_default'
  AND membership.provenance->>'selection' = 'governed_ranked_snp500_top200'
ON CONFLICT (request_key) DO UPDATE SET updated_at = now();

DO $verify$
DECLARE
  universal_count integer;
  default_count integer;
BEGIN
  SELECT count(*) INTO universal_count
  FROM marketops_universal_assets
  WHERE tenant_id = 'tenant-local' AND is_active;
  SELECT count(*) INTO default_count
  FROM subscriber_watchlist_memberships membership
  JOIN subscriber_watchlists list ON list.list_id = membership.list_id
  WHERE membership.tenant_id = 'tenant-local' AND list.list_kind = 'tenant_default';
  IF universal_count < 197 OR default_count < 197 THEN
    RAISE EXCEPTION 'S&P 200 union expected at least 197 active assets/default memberships; universal %, default %', universal_count, default_count;
  END IF;
END;
$verify$;
