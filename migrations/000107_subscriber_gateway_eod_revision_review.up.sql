-- Subscriber S4: narrow, immutable delta projection for analyst review.
-- This exposes comparison evidence only; it grants no direct global-table access.

CREATE VIEW subscriber_gateway_eod_revision_review WITH (security_barrier = true) AS
SELECT
  asset.canonical_symbol,
  delta.session_date,
  delta.field_name,
  (delta.initial_value #>> '{}')::double precision AS initial_value,
  (delta.revised_value #>> '{}')::double precision AS revised_value,
  delta.delta_class,
  delta.materiality,
  initial_observation.observed_at AS initial_observed_at,
  revised_observation.observed_at AS revised_observed_at,
  initial_observation.source_event_id AS initial_source_event_id,
  revised_observation.source_event_id AS revised_source_event_id,
  initial_observation.source_run_id AS initial_source_run_id,
  revised_observation.source_run_id AS revised_source_run_id,
  initial_observation.payload_fingerprint AS initial_payload_fingerprint,
  revised_observation.payload_fingerprint AS revised_payload_fingerprint,
  initial_observation.algorithm_version AS initial_algorithm_version,
  revised_observation.algorithm_version AS revised_algorithm_version
FROM subscriber_global_eod_provider_revision_field_deltas delta
JOIN subscriber_global_assets asset ON asset.global_asset_id = delta.global_asset_id
JOIN subscriber_global_eod_provider_revision_observations initial_observation
  ON initial_observation.revision_observation_id = delta.initial_revision_observation_id
JOIN subscriber_global_eod_provider_revision_observations revised_observation
  ON revised_observation.revision_observation_id = delta.revised_revision_observation_id;

ALTER VIEW subscriber_gateway_eod_revision_review OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_eod_revision_review FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_eod_revision_review TO signalops_subscriber_gateway;
