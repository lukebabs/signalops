-- Subscriber global analytical-data-plane: make the existing narrow current
-- EOD context capable of serving immutable platform-global history bars. A
-- newer verified global re-observation remains preferred for its session; no
-- tenant-local source is consulted by this Gateway projection.

CREATE OR REPLACE VIEW subscriber_gateway_current_eod_context WITH (security_barrier = true) AS
WITH global_reobservations AS (
  SELECT
    resolved.global_asset_id,
    asset.canonical_symbol,
    resolved.session_date,
    (resolved.payload->>'open')::double precision AS open,
    (resolved.payload->>'high')::double precision AS high,
    (resolved.payload->>'low')::double precision AS low,
    (resolved.payload->>'close')::double precision AS close,
    (resolved.payload->>'volume')::bigint AS volume,
    (resolved.payload->>'vwap')::double precision AS vwap,
    resolved.provider,
    resolved.selected_observation_role,
    resolved.selection_policy_version,
    resolved.payload_fingerprint,
    resolved.source_event_id,
    resolved.source_run_id,
    resolved.algorithm_version,
    resolved.quality_state,
    resolved.as_of_time,
    1 AS source_priority
  FROM subscriber_global_eod_resolved_observations resolved
  JOIN subscriber_global_assets asset ON asset.global_asset_id = resolved.global_asset_id
  WHERE resolved.usage_context = 'current_market_context'
), global_history AS (
  SELECT
    record.global_asset_id,
    asset.canonical_symbol,
    record.session_date,
    (record.payload->>'open')::double precision AS open,
    (record.payload->>'high')::double precision AS high,
    (record.payload->>'low')::double precision AS low,
    (record.payload->>'close')::double precision AS close,
    (record.payload->>'volume')::bigint AS volume,
    (record.payload->>'vwap')::double precision AS vwap,
    record.source_system AS provider,
    'initial_global_evidence_capture'::text AS selected_observation_role,
    'global-eod-history-reader-v1'::text AS selection_policy_version,
    record.evidence_fingerprint AS payload_fingerprint,
    record.source_event_id,
    record.evidence_run_id AS source_run_id,
    record.algorithm_version,
    record.quality_state,
    record.observed_at AS as_of_time,
    2 AS source_priority
  FROM subscriber_global_marketops_evidence_records record
  JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
  JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'eod_bar'
    AND record.algorithm_id = 'marketops.equity_eod.initial_capture'
    AND record.algorithm_version = 'v1'
    AND record.quality_state = 'usable'
    AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
), candidates AS (
  SELECT * FROM global_reobservations
  UNION ALL
  SELECT * FROM global_history
)
SELECT DISTINCT ON (global_asset_id)
  global_asset_id, canonical_symbol, session_date, open, high, low, close,
  volume, vwap, provider, selected_observation_role,
  selection_policy_version, payload_fingerprint, source_event_id,
  source_run_id, algorithm_version, quality_state, as_of_time
FROM candidates
ORDER BY global_asset_id, session_date DESC, source_priority, as_of_time DESC,
  source_event_id DESC;

ALTER VIEW subscriber_gateway_current_eod_context OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_current_eod_context FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_current_eod_context TO signalops_subscriber_gateway;
GRANT SELECT ON subscriber_gateway_current_eod_context TO signalops;
