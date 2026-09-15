-- Global Risk/Reward and Options Flow projections. Tenant memberships
-- authorize reads; they never own duplicate analytical records.

ALTER TABLE subscriber_global_marketops_evidence_runs
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_runs
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check
  CHECK (evidence_kind IN ('eod_bar','feature_vector','market_state','eroc','valuation','eeom','material_event','signal_assertion','outcome','sri_snapshot','options_snapshot','risk_reward'));

ALTER TABLE subscriber_global_marketops_evidence_records
  DROP CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_records
  ADD CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check
  CHECK (evidence_kind IN ('eod_bar','feature_vector','market_state','eroc','valuation','eeom','material_event','signal_assertion','outcome','sri_snapshot','options_snapshot','risk_reward'));

ALTER TABLE subscriber_global_marketops_legacy_parity_manifest_entries
  DROP CONSTRAINT subscriber_global_marketops_legacy_parity_m_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_legacy_parity_manifest_entries
  ADD CONSTRAINT subscriber_global_marketops_legacy_parity_m_evidence_kind_check
  CHECK (evidence_kind IN ('feature_vector','market_state','valuation','eeom','signal_assertion','outcome','options_snapshot','risk_reward'));

CREATE VIEW subscriber_global_marketops_legacy_parity_source_v2 WITH (security_barrier = true) AS
SELECT * FROM subscriber_global_marketops_legacy_parity_source
UNION ALL
SELECT 'options_snapshot'::text,
  concat('options:',distribution.symbol,':',distribution.trade_date::text,':',distribution.window_name,':',distribution.updated_at::text),
  distribution.symbol, distribution.trade_date, 'marketops.options_distribution', 'v1',
  CASE WHEN distribution.contract_count > 0 AND distribution.total_call_volume + distribution.total_put_volume > 0 THEN 'usable' ELSE 'partial' END,
  jsonb_build_object(
    'window_name',distribution.window_name,'source_id',distribution.source_id,'provider',distribution.provider,
    'trade_days',distribution.trade_days,'contract_count',distribution.contract_count,
    'call_contract_count',distribution.call_contract_count,'put_contract_count',distribution.put_contract_count,
    'total_call_open_interest',distribution.total_call_open_interest,'total_put_open_interest',distribution.total_put_open_interest,
    'total_call_volume',distribution.total_call_volume,'total_put_volume',distribution.total_put_volume,
    'missing_open_interest_count',distribution.missing_open_interest_count,
    'call_put_open_interest_ratio',distribution.call_put_open_interest_ratio,'call_put_volume_ratio',distribution.call_put_volume_ratio,
    'ratio_delta',distribution.ratio_delta,'ratio_change_pct',distribution.ratio_change_pct,
    'ratio_zscore',distribution.ratio_zscore,'change_point_score',distribution.change_point_score,'confidence',distribution.confidence,
    'moneyness_distribution',distribution.moneyness_distribution,'expiration_distribution',distribution.expiration_distribution,
    'metrics',distribution.metrics,'provenance',distribution.provenance,'source_trade_dates',to_jsonb(distribution.source_trade_dates)),
  distribution.updated_at
FROM marketops_options_distribution_daily distribution
WHERE distribution.tenant_id='tenant-local'
UNION ALL
SELECT 'risk_reward'::text, snapshot.snapshot_id, snapshot.symbol, snapshot.session_date,
  'signalops.algorithms.risk_reward_temporal_v1', 'v1',
  CASE WHEN snapshot.eligible AND snapshot.usable_input_count >= 5 THEN 'usable' ELSE 'partial' END,
  jsonb_build_object('algorithm_result_id',snapshot.algorithm_result_id,
    'execution_request_id',snapshot.execution_request_id,'technical_score',snapshot.technical_score,
    'technical_direction',snapshot.technical_direction,'risk_level',snapshot.risk_level,
    'confidence',snapshot.confidence,'usable_input_count',snapshot.usable_input_count,
    'required_input_count',snapshot.required_input_count,'eligible',snapshot.eligible,
    'result_payload',snapshot.result_payload,'input_snapshot',snapshot.input_snapshot,
    'observed_at',snapshot.observed_at),
  snapshot.created_at
FROM marketops_risk_reward_snapshots snapshot
WHERE snapshot.tenant_id='tenant-local';

CREATE VIEW subscriber_gateway_global_options_distributions WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id,record.session_date,record.payload->>'window_name')
  asset.canonical_symbol AS symbol, record.session_date AS trade_date,
  COALESCE(record.payload->>'window_name','10_trade_days') AS window_name,
  COALESCE(record.payload->>'source_id','') AS source_id, COALESCE(record.payload->>'provider','massive') AS provider,
  COALESCE((record.payload->>'trade_days')::integer,0) AS trade_days,
  COALESCE((record.payload->>'contract_count')::integer,0) AS contract_count,
  COALESCE((record.payload->>'call_contract_count')::integer,0) AS call_contract_count,
  COALESCE((record.payload->>'put_contract_count')::integer,0) AS put_contract_count,
  COALESCE((record.payload->>'total_call_open_interest')::bigint,0) AS total_call_open_interest,
  COALESCE((record.payload->>'total_put_open_interest')::bigint,0) AS total_put_open_interest,
  COALESCE((record.payload->>'total_call_volume')::bigint,0) AS total_call_volume,
  COALESCE((record.payload->>'total_put_volume')::bigint,0) AS total_put_volume,
  COALESCE((record.payload->>'missing_open_interest_count')::integer,0) AS missing_open_interest_count,
  COALESCE((record.payload->>'call_put_open_interest_ratio')::double precision,0) AS call_put_open_interest_ratio,
  COALESCE((record.payload->>'call_put_volume_ratio')::double precision,0) AS call_put_volume_ratio,
  COALESCE((record.payload->>'ratio_delta')::double precision,0) AS ratio_delta,
  COALESCE((record.payload->>'ratio_change_pct')::double precision,0) AS ratio_change_pct,
  COALESCE((record.payload->>'ratio_zscore')::double precision,0) AS ratio_zscore,
  COALESCE((record.payload->>'change_point_score')::double precision,0) AS change_point_score,
  COALESCE((record.payload->>'confidence')::double precision,0) AS confidence,
  COALESCE(record.payload->'moneyness_distribution','{}'::jsonb) AS moneyness_distribution,
  COALESCE(record.payload->'expiration_distribution','{}'::jsonb) AS expiration_distribution,
  COALESCE(record.payload->'metrics','{}'::jsonb) AS metrics,
  COALESCE(record.payload->'provenance','{}'::jsonb) AS provenance,
  COALESCE(record.payload->'source_trade_dates','[]'::jsonb)::text AS source_trade_dates,
  record.observed_at
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id=record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id=record.global_asset_id
WHERE record.evidence_kind='options_snapshot'
  AND run.source_scope IN ('global_provider_capture','legacy_materialization')
ORDER BY record.global_asset_id,record.session_date,record.payload->>'window_name',record.observed_at DESC;

CREATE VIEW subscriber_gateway_global_risk_reward_snapshots WITH (security_barrier = true) AS
SELECT record.global_asset_id, asset.canonical_symbol AS symbol, record.source_event_id AS snapshot_id,
  record.session_date, record.observed_at,
  COALESCE((record.payload->>'usable_input_count')::integer,0) AS usable_input_count,
  COALESCE((record.payload->>'required_input_count')::integer,0) AS required_input_count,
  COALESCE((record.payload->>'eligible')::boolean,false) AS eligible,
  COALESCE(record.payload->'result_payload','{}'::jsonb) AS result_payload,
  record.algorithm_version, record.quality_state, record.evidence_fingerprint,
  record.validation_contract_ref, record.immutable_baseline_ref, record.provenance
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id=record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id=record.global_asset_id
WHERE record.evidence_kind='risk_reward'
  AND run.source_scope IN ('global_provider_capture','legacy_materialization');

ALTER VIEW subscriber_global_marketops_legacy_parity_source_v2 OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_options_distributions OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_risk_reward_snapshots OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_marketops_legacy_parity_source_v2 FROM PUBLIC;
REVOKE ALL ON subscriber_gateway_global_options_distributions FROM PUBLIC;
REVOKE ALL ON subscriber_gateway_global_risk_reward_snapshots FROM PUBLIC;
GRANT SELECT ON marketops_options_distribution_daily, marketops_risk_reward_snapshots TO signalops_subscriber_migrator;
GRANT SELECT ON subscriber_global_marketops_legacy_parity_source_v2 TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_gateway_global_options_distributions, subscriber_gateway_global_risk_reward_snapshots TO signalops_subscriber_gateway;
