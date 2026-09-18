-- Subscriber S4: a narrow gateway read projection for current MarketOps context.
-- The gateway receives no direct grant on global catalog or revision tables.

CREATE VIEW subscriber_gateway_current_eod_context WITH (security_barrier = true) AS
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
  resolved.as_of_time
FROM subscriber_global_eod_resolved_observations resolved
JOIN subscriber_global_assets asset ON asset.global_asset_id = resolved.global_asset_id
WHERE resolved.usage_context = 'current_market_context';

ALTER VIEW subscriber_gateway_current_eod_context OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_current_eod_context FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_current_eod_context TO signalops_subscriber_gateway;
