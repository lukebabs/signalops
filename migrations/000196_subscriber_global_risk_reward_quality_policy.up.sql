-- Keep global Risk/Reward quality classification aligned with the active
-- adaptive contract: four core technical inputs are sufficient for a
-- deterministic, confidence-qualified posture.

CREATE VIEW subscriber_global_marketops_legacy_parity_source_v4
  WITH (security_barrier = true) AS
SELECT
  source.evidence_kind,
  source.legacy_record_id,
  source.legacy_symbol,
  source.legacy_session_date,
  source.legacy_algorithm_id,
  source.legacy_algorithm_version,
  CASE
    WHEN source.evidence_kind = 'risk_reward'
      AND COALESCE((source.legacy_payload->>'eligible')::boolean, false)
      AND COALESCE((source.legacy_payload->>'usable_input_count')::integer, 0) >= 4
      THEN 'usable'
    ELSE source.legacy_quality_state
  END AS legacy_quality_state,
  source.legacy_payload,
  source.legacy_created_at
FROM subscriber_global_marketops_legacy_parity_source_v3 source;

ALTER VIEW subscriber_global_marketops_legacy_parity_source_v4 OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_marketops_legacy_parity_source_v4 FROM PUBLIC;
GRANT SELECT ON subscriber_global_marketops_legacy_parity_source_v3 TO signalops_subscriber_migrator;
GRANT SELECT ON subscriber_global_marketops_legacy_parity_source_v4 TO signalops_subscriber_global_eod;
