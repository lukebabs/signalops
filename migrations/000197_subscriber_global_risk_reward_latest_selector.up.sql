-- Expose one canonical Risk/Reward record per asset/session. Reconciliation
-- is append-only, so the gateway must select the newest observed/materialized
-- evidence rather than surfacing stale duplicates.

CREATE OR REPLACE VIEW subscriber_gateway_global_risk_reward_snapshots
  WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.session_date)
  record.global_asset_id,
  asset.canonical_symbol AS symbol,
  record.source_event_id AS snapshot_id,
  record.session_date,
  record.observed_at,
  COALESCE((record.payload->>'usable_input_count')::integer,0) AS usable_input_count,
  COALESCE((record.payload->>'required_input_count')::integer,0) AS required_input_count,
  COALESCE((record.payload->>'eligible')::boolean,false) AS eligible,
  COALESCE(record.payload->'result_payload','{}'::jsonb) AS result_payload,
  record.algorithm_version,
  record.quality_state,
  record.evidence_fingerprint,
  record.validation_contract_ref,
  record.immutable_baseline_ref,
  record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id=record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id=record.global_asset_id
WHERE record.evidence_kind='risk_reward'
  AND run.source_scope IN ('global_provider_capture','legacy_materialization')
ORDER BY record.global_asset_id, record.session_date, record.observed_at DESC, record.created_at DESC;

ALTER VIEW subscriber_gateway_global_risk_reward_snapshots OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_risk_reward_snapshots FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_risk_reward_snapshots TO signalops_subscriber_gateway;
