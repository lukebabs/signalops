DO $preflight$
DECLARE
  legacy_symbols integer;
  canonically_resolved integer;
BEGIN
  WITH legacy AS (
    SELECT DISTINCT ticker
    FROM marketops_universal_assets
    WHERE tenant_id = 'tenant-local' AND is_active
  ), mappings AS (
    SELECT legacy.ticker,
      count(DISTINCT COALESCE(resolution.canonical_global_asset_id, link.global_asset_id)) AS canonical_ids
    FROM legacy
    JOIN subscriber_global_asset_source_links AS link
      ON link.source_tenant_id = 'tenant-local'
     AND link.source_ticker = legacy.ticker
    LEFT JOIN subscriber_global_asset_identity_resolutions AS resolution
      ON resolution.source_global_asset_id = link.global_asset_id
    GROUP BY legacy.ticker
  )
  SELECT count(*), count(*) FILTER (WHERE canonical_ids = 1)
    INTO legacy_symbols, canonically_resolved
  FROM mappings;

  IF legacy_symbols = 0 OR legacy_symbols <> canonically_resolved THEN
    RAISE EXCEPTION 'tenant-local legacy-default import requires exactly one canonical global identity per active symbol (symbols %, resolved %)', legacy_symbols, canonically_resolved;
  END IF;
END;
$preflight$;

-- Preserve the current tenant-local 132-symbol MarketOps universe as that tenant's
-- durable default list. This migration is additive: it does not alter legacy
-- ownership rows, historical observations, schedules, or subscriber feature flags.
--
-- Membership order is derived from the legacy universe priority/rank and retained
-- in the import timestamp and provenance for stable default-list presentation.

WITH legacy_source_rows AS (
  SELECT DISTINCT ON (asset.ticker)
    asset.ticker, asset.universe_priority, asset.rank
  FROM marketops_universal_assets AS asset
  WHERE asset.tenant_id = 'tenant-local' AND asset.is_active
  ORDER BY asset.ticker, asset.universe_priority, asset.rank
),
legacy_universe AS (
  SELECT
    source.ticker, source.universe_priority, source.rank,
    row_number() OVER (ORDER BY source.universe_priority, source.rank, source.ticker) AS legacy_order
  FROM legacy_source_rows AS source
),
legacy_members AS (
  SELECT DISTINCT ON (legacy.ticker)
    legacy.ticker, legacy.universe_priority, legacy.rank, legacy.legacy_order,
    COALESCE(resolution.canonical_global_asset_id, link.global_asset_id) AS global_asset_id
  FROM legacy_universe AS legacy
  JOIN subscriber_global_asset_source_links AS link
    ON link.source_tenant_id = 'tenant-local'
   AND link.source_ticker = legacy.ticker
  LEFT JOIN subscriber_global_asset_identity_resolutions AS resolution
    ON resolution.source_global_asset_id = link.global_asset_id
  ORDER BY legacy.ticker, COALESCE(resolution.canonical_global_asset_id, link.global_asset_id)
),
inserted_list AS (
  INSERT INTO subscriber_watchlists
    (list_id, tenant_id, list_kind, owner_subject, list_name, created_by_subject, updated_by_subject, provenance)
  VALUES
    ('sublist-tenant-local-legacy-default', 'tenant-local', 'tenant_default', '', 'MarketOps Legacy Default',
     'subscriber-legacy-default-import', 'subscriber-legacy-default-import',
     jsonb_build_object(
       'schema_version', 'subscriber.watchlist.legacy-default.v1',
       'source_scope', 'tenant-local',
       'source_table', 'marketops_universal_assets',
       'selection_policy', 'active_distinct_ticker_ordered_by_universe_priority_rank',
       'preservation_policy', 'legacy-default-membership-preserved'
     ))
  ON CONFLICT (tenant_id) WHERE (list_kind = 'tenant_default') DO NOTHING
  RETURNING list_id
),
default_list AS (
  SELECT list_id FROM inserted_list
  UNION ALL
  SELECT list_id FROM subscriber_watchlists
  WHERE tenant_id = 'tenant-local' AND list_kind = 'tenant_default'
  LIMIT 1
),
inserted_members AS (
  INSERT INTO subscriber_watchlist_memberships
    (tenant_id, list_id, global_asset_id, added_by_subject, provenance, added_at, updated_at)
  SELECT
    'tenant-local', default_list.list_id, member.global_asset_id, 'subscriber-legacy-default-import',
    jsonb_build_object(
      'schema_version', 'subscriber.watchlist.legacy-default.v1',
      'source_scope', 'tenant-local',
      'source_ticker', member.ticker,
      'source_universe_priority', member.universe_priority,
      'source_rank', member.rank,
      'legacy_order', member.legacy_order,
      'preservation_policy', 'legacy-default-membership-preserved'
    ),
    now() + (member.legacy_order * interval '1 microsecond'),
    now() + (member.legacy_order * interval '1 microsecond')
  FROM legacy_members AS member
  CROSS JOIN default_list
  ON CONFLICT (list_id, global_asset_id) DO NOTHING
  RETURNING list_id, global_asset_id
),
list_audit AS (
  INSERT INTO subscriber_watchlist_audit
    (audit_id, tenant_id, list_id, actor_subject, mutation, global_asset_id, before_value, after_value, correlation_id, occurred_at)
  SELECT
    'sublistaudit-' || md5('tenant-local-legacy-default-create-v1'),
    'tenant-local', list_id, 'subscriber-legacy-default-import', 'create_list', '',
    '{}'::jsonb, jsonb_build_object('list_name', 'MarketOps Legacy Default', 'source_scope', 'tenant-local'),
    'subscriber-legacy-default-132-v1', now()
  FROM inserted_list
  ON CONFLICT (audit_id) DO NOTHING
)
INSERT INTO subscriber_watchlist_audit
  (audit_id, tenant_id, list_id, actor_subject, mutation, global_asset_id, before_value, after_value, correlation_id, occurred_at)
SELECT
  'sublistaudit-' || md5('tenant-local-legacy-default-member-v1:' || member.global_asset_id),
  'tenant-local', member.list_id, 'subscriber-legacy-default-import', 'add_asset', member.global_asset_id,
  '{}'::jsonb, jsonb_build_object('global_asset_id', member.global_asset_id, 'source_scope', 'tenant-local'),
  'subscriber-legacy-default-132-v1', now()
FROM inserted_members AS member
ON CONFLICT (audit_id) DO NOTHING;
