-- Restore pre-000164 gateway global projection view definitions.
-- Immutable evidence remains untouched.

CREATE OR REPLACE VIEW subscriber_gateway_global_marketops_evidence_coverage WITH (security_barrier = true) AS
SELECT record.global_asset_id,
    asset.canonical_symbol,
    record.evidence_kind,
    max(record.session_date) AS latest_session_date,
    max(record.observed_at) AS latest_observed_at,
    count(*) FILTER (WHERE record.quality_state = 'usable'::text) AS usable_record_count,
    count(*) FILTER (WHERE record.quality_state = 'partial'::text) AS partial_record_count,
    count(*) FILTER (WHERE record.quality_state = 'invalid'::text) AS invalid_record_count
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE run.source_scope = ANY (ARRAY['global_provider_capture'::text, 'legacy_materialization'::text])
  GROUP BY record.global_asset_id, asset.canonical_symbol, record.evidence_kind;;

ALTER VIEW subscriber_gateway_global_marketops_evidence_coverage OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_marketops_evidence_coverage FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_marketops_evidence_coverage TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_market_states WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.session_date) record.source_event_id AS market_state_id,
    record.global_asset_id,
    asset.canonical_symbol AS symbol,
    record.session_date,
    record.observed_at AS as_of_time,
    record.algorithm_version AS state_schema_version,
    record.quality_state,
    record.payload,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'market_state'::text AND (run.source_scope = ANY (ARRAY['global_provider_capture'::text, 'legacy_materialization'::text]))
  ORDER BY record.global_asset_id, record.session_date, record.observed_at DESC, record.global_evidence_id DESC;;

ALTER VIEW subscriber_gateway_global_market_states OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_market_states FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_market_states TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_eroc_results WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.session_date) record.source_event_id AS result_id,
    COALESCE(record.payload ->> 'snapshot_id'::text, ''::text) AS snapshot_id,
    record.global_asset_id,
    asset.canonical_symbol AS symbol,
    record.session_date,
    record.algorithm_id,
    record.algorithm_version AS model_version,
    COALESCE((record.payload ->> 'score'::text)::double precision, 0::double precision) AS score,
    COALESCE((record.payload ->> 'fair_value'::text)::double precision, 0::double precision) AS fair_value,
    COALESCE(record.payload ->> 'classification'::text, ''::text) AS classification,
    COALESCE((record.payload ->> 'confidence'::text)::integer, 0) AS confidence,
    COALESCE(record.payload ->> 'confidence_label'::text, ''::text) AS confidence_label,
    COALESCE(record.payload ->> 'evaluation_status'::text, ''::text) AS evaluation_status,
    COALESCE((record.payload ->> 'eligible'::text)::boolean, false) AS eligible,
    COALESCE(record.payload -> 'result_json'::text, '{}'::jsonb) AS result_json,
    record.observed_at AS created_at,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'valuation'::text AND record.algorithm_id = 'signalops.algorithms.eroc_v6'::text AND (run.source_scope = ANY (ARRAY['global_provider_capture'::text, 'legacy_materialization'::text]))
  ORDER BY record.global_asset_id, record.session_date, record.observed_at DESC, record.global_evidence_id DESC;;

ALTER VIEW subscriber_gateway_global_eroc_results OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_eroc_results FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_eroc_results TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_eeom_results WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, (COALESCE(record.payload ->> 'earnings_event_id'::text, ''::text))) record.source_event_id AS result_id,
    record.global_asset_id,
    asset.canonical_symbol AS symbol,
    COALESCE(record.payload ->> 'earnings_event_id'::text, ''::text) AS earnings_event_id,
    COALESCE((record.payload ->> 'earnings_date'::text)::date, record.session_date) AS earnings_date,
    record.session_date,
    record.algorithm_version AS model_version,
    COALESCE((record.payload ->> 'score'::text)::double precision, 0::double precision) AS score,
    COALESCE(record.payload ->> 'posture'::text, ''::text) AS posture,
    COALESCE(record.payload ->> 'classification'::text, ''::text) AS classification,
    record.quality_state AS evidence_quality,
    COALESCE((record.payload ->> 'eligible'::text)::boolean, false) AS eligible,
    COALESCE(record.payload -> 'result_json'::text, '{}'::jsonb) AS result_json,
    record.observed_at AS created_at,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'eeom'::text AND record.algorithm_id = 'earnings_event_opportunity'::text AND (run.source_scope = ANY (ARRAY['global_provider_capture'::text, 'legacy_materialization'::text]))
  ORDER BY record.global_asset_id, (COALESCE(record.payload ->> 'earnings_event_id'::text, ''::text)), record.session_date DESC, record.observed_at DESC, record.global_evidence_id DESC;;

ALTER VIEW subscriber_gateway_global_eeom_results OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_eeom_results FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_eeom_results TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_material_events WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.source_event_id) record.global_asset_id,
    asset.canonical_symbol AS symbol,
    record.source_event_id AS event_id,
    record.algorithm_id,
    record.algorithm_version,
    record.session_date,
    record.observed_at,
    record.quality_state,
    record.payload,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'material_event'::text AND record.algorithm_id = 'market_data.fmp.earnings_calendar'::text AND run.source_scope = 'global_provider_capture'::text
  ORDER BY record.global_asset_id, record.source_event_id, record.observed_at DESC, record.global_evidence_id DESC;;

ALTER VIEW subscriber_gateway_global_material_events OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_material_events FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_material_events TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_options_distributions WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id, record.session_date, (record.payload ->> 'window_name'::text)) asset.canonical_symbol AS symbol,
    record.session_date AS trade_date,
    COALESCE(record.payload ->> 'window_name'::text, '10_trade_days'::text) AS window_name,
    COALESCE(record.payload ->> 'source_id'::text, ''::text) AS source_id,
    COALESCE(record.payload ->> 'provider'::text, 'massive'::text) AS provider,
    COALESCE((record.payload ->> 'trade_days'::text)::integer, 0) AS trade_days,
    COALESCE((record.payload ->> 'contract_count'::text)::integer, 0) AS contract_count,
    COALESCE((record.payload ->> 'call_contract_count'::text)::integer, 0) AS call_contract_count,
    COALESCE((record.payload ->> 'put_contract_count'::text)::integer, 0) AS put_contract_count,
    COALESCE((record.payload ->> 'total_call_open_interest'::text)::bigint, 0::bigint) AS total_call_open_interest,
    COALESCE((record.payload ->> 'total_put_open_interest'::text)::bigint, 0::bigint) AS total_put_open_interest,
    COALESCE((record.payload ->> 'total_call_volume'::text)::bigint, 0::bigint) AS total_call_volume,
    COALESCE((record.payload ->> 'total_put_volume'::text)::bigint, 0::bigint) AS total_put_volume,
    COALESCE((record.payload ->> 'missing_open_interest_count'::text)::integer, 0) AS missing_open_interest_count,
    COALESCE((record.payload ->> 'call_put_open_interest_ratio'::text)::double precision, 0::double precision) AS call_put_open_interest_ratio,
    COALESCE((record.payload ->> 'call_put_volume_ratio'::text)::double precision, 0::double precision) AS call_put_volume_ratio,
    COALESCE((record.payload ->> 'ratio_delta'::text)::double precision, 0::double precision) AS ratio_delta,
    COALESCE((record.payload ->> 'ratio_change_pct'::text)::double precision, 0::double precision) AS ratio_change_pct,
    COALESCE((record.payload ->> 'ratio_zscore'::text)::double precision, 0::double precision) AS ratio_zscore,
    COALESCE((record.payload ->> 'change_point_score'::text)::double precision, 0::double precision) AS change_point_score,
    COALESCE((record.payload ->> 'confidence'::text)::double precision, 0::double precision) AS confidence,
    COALESCE(record.payload -> 'moneyness_distribution'::text, '{}'::jsonb) AS moneyness_distribution,
    COALESCE(record.payload -> 'expiration_distribution'::text, '{}'::jsonb) AS expiration_distribution,
    COALESCE(record.payload -> 'metrics'::text, '{}'::jsonb) AS metrics,
    COALESCE(record.payload -> 'provenance'::text, '{}'::jsonb) AS provenance,
    COALESCE(record.payload -> 'source_trade_dates'::text, '[]'::jsonb)::text AS source_trade_dates,
    record.observed_at
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'options_snapshot'::text AND (run.source_scope = ANY (ARRAY['global_provider_capture'::text, 'legacy_materialization'::text]))
  ORDER BY record.global_asset_id, record.session_date, (record.payload ->> 'window_name'::text), record.observed_at DESC;;

ALTER VIEW subscriber_gateway_global_options_distributions OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_options_distributions FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_options_distributions TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_risk_reward_snapshots WITH (security_barrier = true) AS
SELECT record.global_asset_id,
    asset.canonical_symbol AS symbol,
    record.source_event_id AS snapshot_id,
    record.session_date,
    record.observed_at,
    COALESCE((record.payload ->> 'usable_input_count'::text)::integer, 0) AS usable_input_count,
    COALESCE((record.payload ->> 'required_input_count'::text)::integer, 0) AS required_input_count,
    COALESCE((record.payload ->> 'eligible'::text)::boolean, false) AS eligible,
    COALESCE(record.payload -> 'result_payload'::text, '{}'::jsonb) AS result_payload,
    record.algorithm_version,
    record.quality_state,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'risk_reward'::text AND (run.source_scope = ANY (ARRAY['global_provider_capture'::text, 'legacy_materialization'::text]));;

ALTER VIEW subscriber_gateway_global_risk_reward_snapshots OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_risk_reward_snapshots FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_risk_reward_snapshots TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_intraday_current_states WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.global_asset_id) record.global_asset_id,
    asset.canonical_symbol AS symbol,
    record.source_event_id AS snapshot_id,
    (record.payload ->> 'as_of_time'::text)::timestamp with time zone AS as_of_time,
    record.session_date,
    COALESCE(record.payload ->> 'universe_group'::text, 'all_active'::text) AS universe_group,
    COALESCE(record.payload ->> 'market_status'::text, ''::text) AS market_status,
    COALESCE((record.payload ->> 'stale'::text)::boolean, true) AS stale,
    COALESCE(record.payload -> 'conditions'::text, '[]'::jsonb) AS conditions,
    COALESCE(record.payload -> 'source_payload'::text, '{}'::jsonb) AS source_payload,
    true AS current_only_source,
    record.algorithm_id,
    record.algorithm_version,
    record.quality_state,
    record.evidence_fingerprint,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
  WHERE record.evidence_kind = 'intraday_snapshot'::text AND run.source_scope = 'legacy_materialization'::text AND COALESCE((record.payload ->> 'current_only_source'::text)::boolean, false) AND NULLIF(record.payload ->> 'as_of_time'::text, ''::text) IS NOT NULL
  ORDER BY record.global_asset_id, ((record.payload ->> 'as_of_time'::text)::timestamp with time zone) DESC, record.evidence_fingerprint DESC;;

ALTER VIEW subscriber_gateway_global_intraday_current_states OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_intraday_current_states FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_intraday_current_states TO signalops_subscriber_gateway;

CREATE OR REPLACE VIEW subscriber_gateway_global_signal_assurance_observations WITH (security_barrier = true) AS
SELECT DISTINCT ON (record.source_event_id) record.source_event_id AS observation_id,
    record.global_asset_id,
    asset.canonical_symbol AS symbol,
    COALESCE(record.payload ->> 'source_id'::text, ''::text) AS source_id,
    COALESCE(record.payload ->> 'source_type'::text, (record.payload -> 'outcome_payload'::text) ->> 'source_type'::text, record.algorithm_id) AS source_type,
    COALESCE(record.payload ->> 'direction'::text, ''::text) AS direction,
    COALESCE((record.payload ->> 'horizon_sessions'::text)::integer, 0) AS horizon_sessions,
    record.session_date AS origin_session_date,
    NULLIF(record.payload ->> 'matured_session_date'::text, ''::text)::date AS matured_session_date,
    NULLIF(record.payload ->> 'directional_hit'::text, ''::text)::boolean AS directional_hit,
    NULLIF(record.payload ->> 'forward_return'::text, ''::text)::double precision AS forward_return,
    NULLIF(record.payload ->> 'max_favorable_excursion'::text, ''::text)::double precision AS mfe,
    NULLIF(record.payload ->> 'max_adverse_excursion'::text, ''::text)::double precision AS mae,
    record.algorithm_version AS calculation_version,
    COALESCE(record.payload ->> 'calculation_run_id'::text, ''::text) AS calculation_run_id,
    record.quality_state,
    record.validation_contract_ref,
    record.immutable_baseline_ref,
    record.provenance,
    record.observed_at,
    COALESCE(broad_v4.benchmark_symbol, broad_v3.benchmark_symbol, broad_v2.benchmark_symbol, broad_v1.benchmark_symbol) AS broad_market_benchmark_symbol,
    COALESCE(broad_v4.benchmark_relative_return, broad_v3.benchmark_relative_return, broad_v2.benchmark_relative_return, broad_v1.benchmark_relative_return) AS broad_market_relative_return,
    COALESCE(broad_v4.benchmark_resolution_state, broad_v3.benchmark_resolution_state, broad_v2.benchmark_resolution_state, broad_v1.benchmark_resolution_state) AS broad_market_benchmark_state,
    COALESCE(sector_v4.benchmark_symbol, sector_v3.benchmark_symbol, sector_v2.benchmark_symbol, sector_v1.benchmark_symbol) AS sector_benchmark_symbol,
    COALESCE(sector_v4.benchmark_segment_key, sector_v3.benchmark_segment_key, sector_v2.benchmark_segment_key, sector_v1.benchmark_segment_key) AS sector_benchmark_segment_key,
    COALESCE(sector_v4.benchmark_relative_return, sector_v3.benchmark_relative_return, sector_v2.benchmark_relative_return, sector_v1.benchmark_relative_return) AS sector_relative_return,
    COALESCE(sector_v4.benchmark_resolution_state, sector_v3.benchmark_resolution_state, sector_v2.benchmark_resolution_state, sector_v1.benchmark_resolution_state) AS sector_benchmark_state
   FROM subscriber_global_marketops_evidence_records record
     JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
     JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
     LEFT JOIN subscriber_global_saf_benchmark_observations broad_v4 ON broad_v4.source_observation_id = record.source_event_id AND broad_v4.benchmark_kind = 'broad_market'::text AND broad_v4.calculation_version = 'saf_benchmark.v4'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations broad_v3 ON broad_v3.source_observation_id = record.source_event_id AND broad_v3.benchmark_kind = 'broad_market'::text AND broad_v3.calculation_version = 'saf_benchmark.v3'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations broad_v2 ON broad_v2.source_observation_id = record.source_event_id AND broad_v2.benchmark_kind = 'broad_market'::text AND broad_v2.calculation_version = 'saf_benchmark.v2'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations broad_v1 ON broad_v1.source_observation_id = record.source_event_id AND broad_v1.benchmark_kind = 'broad_market'::text AND broad_v1.calculation_version = 'saf_benchmark.v1'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations sector_v4 ON sector_v4.source_observation_id = record.source_event_id AND sector_v4.benchmark_kind = 'sector'::text AND sector_v4.calculation_version = 'saf_benchmark.v4'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations sector_v3 ON sector_v3.source_observation_id = record.source_event_id AND sector_v3.benchmark_kind = 'sector'::text AND sector_v3.calculation_version = 'saf_benchmark.v3'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations sector_v2 ON sector_v2.source_observation_id = record.source_event_id AND sector_v2.benchmark_kind = 'sector'::text AND sector_v2.calculation_version = 'saf_benchmark.v2'::text
     LEFT JOIN subscriber_global_saf_benchmark_observations sector_v1 ON sector_v1.source_observation_id = record.source_event_id AND sector_v1.benchmark_kind = 'sector'::text AND sector_v1.calculation_version = 'saf_benchmark.v1'::text
  WHERE record.evidence_kind = 'outcome'::text AND record.algorithm_id = 'opportunity'::text AND run.source_scope = 'legacy_materialization'::text AND COALESCE(record.payload ->> 'source_type'::text, (record.payload -> 'outcome_payload'::text) ->> 'source_type'::text, record.algorithm_id) = 'opportunity'::text AND (COALESCE(record.payload ->> 'direction'::text, ''::text) = ANY (ARRAY['upside'::text, 'downside'::text])) AND jsonb_typeof(record.payload -> 'directional_hit'::text) = 'boolean'::text
  ORDER BY record.source_event_id, record.observed_at DESC, record.global_evidence_id DESC;;

ALTER VIEW subscriber_gateway_global_signal_assurance_observations OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_signal_assurance_observations FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_gateway;

GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations TO signalops_subscriber_global_eod;
