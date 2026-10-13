-- Promote the latest governed primary-EOD intraday capture into the global
-- gateway projection. This is read-only over the append-only capture table;
-- it performs no provider call and does not rewrite evidence.
CREATE OR REPLACE VIEW subscriber_gateway_global_intraday_current_states
  WITH (security_barrier = true) AS
WITH evidence_states AS (
  SELECT canonical.global_asset_id,
    canonical.canonical_symbol AS symbol,
    record.source_event_id AS snapshot_id,
    (record.payload->>'as_of_time')::timestamptz AS as_of_time,
    record.session_date,
    COALESCE(record.payload->>'universe_group', 'all_active') AS universe_group,
    COALESCE(record.payload->>'market_status', '') AS market_status,
    COALESCE((record.payload->>'stale')::boolean, true) AS stale,
    COALESCE(record.payload->'conditions', '[]'::jsonb) AS conditions,
    COALESCE(record.payload->'source_payload', '{}'::jsonb) AS source_payload,
    true AS current_only_source,
    record.algorithm_id,
    record.algorithm_version,
    record.quality_state,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance || jsonb_build_object(
      'source_global_asset_id', record.global_asset_id,
      'canonical_global_asset_id', canonical.global_asset_id,
      'canonical_resolution_state', canonical.canonical_resolution_state
    ) AS provenance
  FROM subscriber_global_marketops_evidence_records record
  JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
  JOIN subscriber_global_asset_identity_resolutions resolution ON resolution.source_global_asset_id = record.global_asset_id
  JOIN subscriber_gateway_global_canonical_assets canonical ON canonical.global_asset_id = resolution.canonical_global_asset_id
  WHERE record.evidence_kind = 'intraday_snapshot'
    AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
    AND COALESCE((record.payload->>'current_only_source')::boolean, false)
    AND NULLIF(record.payload->>'as_of_time', '') IS NOT NULL
), live_primary AS (
  SELECT canonical.global_asset_id,
    canonical.canonical_symbol AS symbol,
    snap.snapshot_id,
    snap.as_of_time,
    snap.as_of_time::date AS session_date,
    snap.universe_group,
    snap.market_status,
    snap.stale,
    snap.conditions,
    snap.source_payload,
    true AS current_only_source,
    'marketops.intraday_monitor'::text AS algorithm_id,
    'v1'::text AS algorithm_version,
    CASE WHEN snap.stale THEN 'partial' ELSE 'usable' END AS quality_state,
    md5(snap.snapshot_id || ':' || snap.as_of_time::text) AS evidence_fingerprint,
    'marketops-intraday/primary-eod/v1'::text AS validation_contract_ref,
    'marketops-intraday-snapshot'::text AS immutable_baseline_ref,
    jsonb_build_object(
      'source', 'marketops_intraday_condition_snapshots',
      'tenant_id', snap.tenant_id,
      'universe_group', snap.universe_group,
      'canonical_global_asset_id', canonical.global_asset_id
    ) AS provenance
  FROM marketops_intraday_condition_snapshots snap
  JOIN subscriber_gateway_global_canonical_assets canonical
    ON canonical.canonical_symbol = upper(snap.symbol)
  WHERE snap.tenant_id = 'tenant-local'
    AND snap.universe_group = 'primary_eod'
)
SELECT DISTINCT ON (global_asset_id)
  global_asset_id, symbol, snapshot_id, as_of_time, session_date,
  universe_group, market_status, stale, conditions, source_payload,
  current_only_source, algorithm_id, algorithm_version, quality_state,
  evidence_fingerprint, validation_contract_ref, immutable_baseline_ref,
  provenance
FROM (
  SELECT * FROM evidence_states
  UNION ALL
  SELECT * FROM live_primary
) states
ORDER BY global_asset_id, as_of_time DESC, evidence_fingerprint DESC;

ALTER VIEW subscriber_gateway_global_intraday_current_states OWNER TO signalops_subscriber_migrator;
GRANT SELECT ON marketops_intraday_condition_snapshots,
  subscriber_global_asset_identity_resolutions,
  subscriber_gateway_global_canonical_assets TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_intraday_current_states FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_intraday_current_states TO signalops_subscriber_gateway;
