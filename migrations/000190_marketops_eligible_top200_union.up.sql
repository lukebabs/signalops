-- Correct the operational selection rule: choose the first 200 eligible
-- assets from the governed 1,000-row ranking, rather than taking source ranks
-- 1..200 and then filtering (which produced only 165 eligible assets).
-- The existing tenant-local operational union is retained and expanded.

CREATE TEMP TABLE marketops_ranked_eligible_top200 AS
SELECT DISTINCT ON (asset.global_asset_id)
  asset.global_asset_id,
  asset.canonical_symbol,
  asset.company_name,
  asset.asset_type,
  asset.exchange,
  asset.sector,
  asset.industry,
  entry.source_rank,
  row_number() OVER (ORDER BY entry.source_rank, asset.canonical_symbol, asset.global_asset_id)::integer AS operational_rank
FROM subscriber_global_ranking_snapshot_entries entry
JOIN subscriber_global_ranking_snapshots snapshot
  ON snapshot.ranking_snapshot_id = entry.ranking_snapshot_id
 AND snapshot.is_current
JOIN subscriber_global_assets asset ON asset.global_asset_id = entry.global_asset_id
WHERE asset.eligibility_status = 'eligible'
  AND entry.selection_rank <= 1000
ORDER BY asset.global_asset_id, entry.source_rank;

DELETE FROM marketops_ranked_eligible_top200
WHERE operational_rank > 200;

DO $preflight$
DECLARE
  selected_count integer;
BEGIN
  SELECT count(*) INTO selected_count FROM marketops_ranked_eligible_top200;
  IF selected_count <> 200 THEN
    RAISE EXCEPTION 'eligible top-200 promotion requires exactly 200 assets; found %', selected_count;
  END IF;
END;
$preflight$;

-- Avoid transient rank collisions while the current projection is reordered.
UPDATE marketops_asset_universe
SET rank = rank + 1000, updated_at = now()
WHERE tenant_id = 'tenant-local' AND universe_group = 'snp500_top200';

UPDATE marketops_asset_universe AS target
SET rank = desired.operational_rank,
    company = desired.company_name,
    company_key = lower(regexp_replace(desired.company_name, '[^A-Za-z0-9]+', '_', 'g')),
    asset_type = COALESCE(NULLIF(desired.asset_type, ''), 'equity'),
    exchange = COALESCE(desired.exchange, ''),
    sector = COALESCE(desired.sector, ''),
    sector_key = lower(regexp_replace(COALESCE(desired.sector, ''), '[^A-Za-z0-9]+', '_', 'g')),
    industry = COALESCE(desired.industry, ''),
    industry_key = lower(regexp_replace(COALESCE(desired.industry, ''), '[^A-Za-z0-9]+', '_', 'g')),
    is_active = true,
    metadata = target.metadata || jsonb_build_object(
      'selection', 'governed_ranked_eligible_top200',
      'ranking_source_rank', desired.source_rank,
      'eligibility_policy', 'eligible_only',
      'selection_revision', '000190'
    ),
    updated_at = now()
FROM marketops_ranked_eligible_top200 desired
WHERE target.tenant_id = 'tenant-local'
  AND target.universe_group = 'snp500_top200'
  AND target.ticker = desired.canonical_symbol;

UPDATE marketops_asset_universe AS target
SET is_active = false, updated_at = now(),
    metadata = target.metadata || jsonb_build_object('superseded_by', '000190')
WHERE target.tenant_id = 'tenant-local'
  AND target.universe_group = 'snp500_top200'
  AND NOT EXISTS (
    SELECT 1 FROM marketops_ranked_eligible_top200 desired
    WHERE desired.canonical_symbol = target.ticker
  );

INSERT INTO marketops_asset_universe (
  tenant_id, app_id, domain, use_case, source_id, universe_group, rank,
  ticker, ticker_key, company, company_key, asset_type, exchange, sector,
  sector_key, industry, industry_key, is_active, metadata
)
SELECT
  'tenant-local', 'marketops', 'market_data', 'daily_market_surveillance',
  'src-subscriber-global-ranking', 'snp500_top200', desired.operational_rank,
  desired.canonical_symbol,
  lower(regexp_replace(desired.canonical_symbol, '[^A-Za-z0-9]+', '_', 'g')),
  desired.company_name,
  lower(regexp_replace(desired.company_name, '[^A-Za-z0-9]+', '_', 'g')),
  COALESCE(NULLIF(desired.asset_type, ''), 'equity'), COALESCE(desired.exchange, ''),
  COALESCE(desired.sector, ''), lower(regexp_replace(COALESCE(desired.sector, ''), '[^A-Za-z0-9]+', '_', 'g')),
  COALESCE(desired.industry, ''), lower(regexp_replace(COALESCE(desired.industry, ''), '[^A-Za-z0-9]+', '_', 'g')),
  true,
  jsonb_build_object('selection', 'governed_ranked_eligible_top200', 'ranking_source_rank', desired.source_rank, 'eligibility_policy', 'eligible_only', 'selection_revision', '000190')
FROM marketops_ranked_eligible_top200 desired
ON CONFLICT (tenant_id, universe_group, ticker) DO UPDATE SET
  rank = EXCLUDED.rank, is_active = true, metadata = EXCLUDED.metadata, updated_at = now();

WITH default_list AS (
  SELECT list_id FROM subscriber_watchlists WHERE tenant_id = 'tenant-local' AND list_kind = 'tenant_default'
), added AS (
  INSERT INTO subscriber_watchlist_memberships
    (tenant_id, list_id, global_asset_id, added_by_subject, provenance)
  SELECT 'tenant-local', default_list.list_id, desired.global_asset_id,
    'subscriber-eligible-top200-000190',
    jsonb_build_object('schema_version','subscriber.watchlist.eligible-top200.v1','selection','governed_ranked_eligible_top200','ranking_source_rank',desired.source_rank,'preservation_policy','legacy-memberships-retained')
  FROM marketops_ranked_eligible_top200 desired CROSS JOIN default_list
  ON CONFLICT (list_id, global_asset_id) DO NOTHING
  RETURNING list_id, global_asset_id
)
INSERT INTO subscriber_watchlist_audit
  (audit_id, tenant_id, list_id, actor_subject, mutation, global_asset_id, before_value, after_value, correlation_id)
SELECT 'sublistaudit-' || md5('eligible-top200-000190:' || added.list_id || ':' || added.global_asset_id),
  'tenant-local', added.list_id, 'subscriber-eligible-top200-000190', 'add_asset', added.global_asset_id,
  '{}'::jsonb, jsonb_build_object('promotion','operational_primary','selection','governed_ranked_eligible_top200'), 'subscriber-eligible-top200-000190'
FROM added ON CONFLICT (audit_id) DO NOTHING;

INSERT INTO subscriber_global_coverage_activation_requests
  (activation_request_id, global_asset_id, request_key, request_state, request_reason,
   requester_kind, requester_tenant_id, requester_subject, requester_list_id,
   policy_version, provenance, requested_at)
SELECT 'subactivation-' || md5('tenant-local:' || desired.global_asset_id), desired.global_asset_id,
  'subscriber-tenant-default-hot-v1:tenant-local:' || desired.global_asset_id,
  'queued', 'subscriber_tenant_default_eligible_top200_union', 'tenant_default_list',
  'tenant-local', 'subscriber-eligible-top200-000190', list.list_id,
  'subscriber-tenant-default-eligible-top200-v1',
  jsonb_build_object('surface','subscriber.eligible-top200.union','migration','000190'), now()
FROM marketops_ranked_eligible_top200 desired
CROSS JOIN LATERAL (SELECT list_id FROM subscriber_watchlists WHERE tenant_id='tenant-local' AND list_kind='tenant_default' LIMIT 1) list
ON CONFLICT (request_key) DO UPDATE SET updated_at = now();

DO $verify$
DECLARE
  universal_count integer;
  default_count integer;
BEGIN
  SELECT count(*) INTO universal_count FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active;
  SELECT count(*) INTO default_count
  FROM subscriber_watchlist_memberships membership
  JOIN subscriber_watchlists list ON list.list_id=membership.list_id
  WHERE membership.tenant_id='tenant-local' AND list.list_kind='tenant_default';
  IF universal_count <> 225 OR default_count <> 225 THEN
    RAISE EXCEPTION 'eligible top-200 union expected 225 assets; universal %, default %', universal_count, default_count;
  END IF;
END;
$verify$;
