-- Subscriber global analytical-data-plane gate: an append-only manifest over
-- tenant-local source rows. The source view is fixed to tenant-local and is
-- the only legacy analytical surface granted to the controlled worker. It
-- reports mapping and materialization status; it never imports a result.

CREATE TABLE subscriber_global_marketops_legacy_parity_runs (
  parity_run_id text PRIMARY KEY,
  parity_version text NOT NULL,
  source_tenant_id text NOT NULL CHECK (source_tenant_id = 'tenant-local'),
  execution_mode text NOT NULL CHECK (execution_mode = 'shadow_read_only'),
  requested_evidence_kinds text[] NOT NULL,
  selected_record_count integer NOT NULL CHECK (selected_record_count >= 0),
  mapped_record_count integer NOT NULL CHECK (mapped_record_count >= 0),
  unmapped_record_count integer NOT NULL CHECK (unmapped_record_count >= 0),
  ambiguous_record_count integer NOT NULL CHECK (ambiguous_record_count >= 0),
  manifest_fingerprint text NOT NULL,
  report jsonb NOT NULL DEFAULT '{}'::jsonb,
  recorded_by text NOT NULL CHECK (recorded_by = 'subscriber-global-eod-reconciler'),
  correlation_id text NOT NULL DEFAULT '',
  source_read_at timestamptz NOT NULL,
  recorded_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (selected_record_count = mapped_record_count + unmapped_record_count + ambiguous_record_count)
);

CREATE TABLE subscriber_global_marketops_legacy_parity_manifest_entries (
  parity_entry_id text PRIMARY KEY,
  parity_run_id text NOT NULL REFERENCES subscriber_global_marketops_legacy_parity_runs(parity_run_id) ON DELETE RESTRICT,
  evidence_kind text NOT NULL CHECK (evidence_kind IN ('feature_vector', 'market_state', 'valuation', 'eeom', 'signal_assertion', 'outcome')),
  legacy_record_id text NOT NULL,
  legacy_symbol text NOT NULL,
  legacy_session_date date NOT NULL,
  legacy_algorithm_id text NOT NULL,
  legacy_algorithm_version text NOT NULL,
  legacy_quality_state text NOT NULL,
  legacy_fingerprint text NOT NULL,
  global_asset_id text REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  mapping_status text NOT NULL CHECK (mapping_status IN ('mapped', 'unmapped', 'ambiguous')),
  global_materialization_status text NOT NULL CHECK (global_materialization_status IN ('pending_global_materialization', 'not_mappable')),
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  manifested_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (parity_run_id, evidence_kind, legacy_record_id),
  CHECK ((mapping_status = 'mapped' AND global_asset_id IS NOT NULL AND global_materialization_status = 'pending_global_materialization')
      OR (mapping_status IN ('unmapped', 'ambiguous') AND global_asset_id IS NULL AND global_materialization_status = 'not_mappable'))
);
CREATE INDEX idx_subscriber_global_marketops_legacy_parity_entries_asset
  ON subscriber_global_marketops_legacy_parity_manifest_entries(global_asset_id, evidence_kind, legacy_session_date DESC);
CREATE INDEX idx_subscriber_global_marketops_legacy_parity_entries_run
  ON subscriber_global_marketops_legacy_parity_manifest_entries(parity_run_id, mapping_status, evidence_kind);

-- SECURITY BARRIER prevents predicate pushdown from changing the fixed source
-- scope. The view deliberately exposes only record metadata and canonical
-- payload text required to calculate a source fingerprint.
CREATE VIEW subscriber_global_marketops_legacy_parity_source WITH (security_barrier = true) AS
SELECT 'feature_vector'::text AS evidence_kind, feature.feature_observation_id AS legacy_record_id,
  feature.symbol AS legacy_symbol, feature.session_date AS legacy_session_date,
  feature.feature_key AS legacy_algorithm_id, feature.feature_version AS legacy_algorithm_version,
  feature.quality_state AS legacy_quality_state,
  jsonb_build_object('asset_id',feature.asset_id,'as_of_time',feature.as_of_time,'dimensions',feature.dimensions,
    'numeric_value',feature.numeric_value,'text_value',feature.text_value,'boolean_value',feature.boolean_value,
    'quality_score',feature.quality_score,'quality_details',feature.quality_details,'source_event_ids',feature.source_event_ids,
    'source_artifact_ids',feature.source_artifact_ids,'calculation_run_id',feature.calculation_run_id,'deterministic_key',feature.deterministic_key) AS legacy_payload,
  feature.created_at AS legacy_created_at
FROM marketops_feature_observations feature WHERE feature.tenant_id='tenant-local'
UNION ALL
SELECT 'market_state', state.market_state_id, state.symbol, state.session_date,
  'market_state', state.state_schema_version, state.quality_state,
  jsonb_build_object('asset_id',state.asset_id,'as_of_time',state.as_of_time,'state_payload',state.state_payload,
    'feature_observation_ids',state.feature_observation_ids,'feature_count',state.feature_count,
    'required_feature_count',state.required_feature_count,'completeness_ratio',state.completeness_ratio,
    'quality_score',state.quality_score,'quality_summary',state.quality_summary,'eligible_hypotheses',state.eligible_hypotheses,
    'build_run_id',state.build_run_id,'deterministic_key',state.deterministic_key), state.created_at
FROM marketops_market_states state WHERE state.tenant_id='tenant-local'
UNION ALL
SELECT 'valuation', value.result_id, value.symbol, value.session_date,
  value.algorithm_id, value.model_version, value.evaluation_status,
  jsonb_build_object('snapshot_id',value.snapshot_id,'score',value.score,'fair_value',value.fair_value,
    'classification',value.classification,'confidence',value.confidence,'confidence_label',value.confidence_label,
    'eligible',value.eligible,'result_json',value.result_json), value.created_at
FROM marketops_valuation_results value WHERE value.tenant_id='tenant-local'
UNION ALL
SELECT 'eeom', eeom.result_id, eeom.symbol, eeom.session_date,
  'earnings_event_opportunity', eeom.model_version, eeom.evidence_quality,
  jsonb_build_object('earnings_event_id',eeom.earnings_event_id,'earnings_date',eeom.earnings_date,'score',eeom.score,
    'posture',eeom.posture,'classification',eeom.classification,'eligible',eeom.eligible,'result_json',eeom.result_json), eeom.created_at
FROM marketops_eeom_results eeom WHERE eeom.tenant_id='tenant-local'
UNION ALL
SELECT 'signal_assertion', assertion.assertion_id, assertion.symbol, assertion.confirmed_at::date,
  assertion.algorithm, assertion.algorithm_version, assertion.state,
  jsonb_build_object('asset_id',assertion.asset_id,'signal_id',assertion.signal_id,'source_ledger_signal_id',assertion.source_ledger_signal_id,
    'signal_type',assertion.signal_type,'signal_direction',assertion.signal_direction,'signal_score',assertion.signal_score,
    'confirmed_at',assertion.confirmed_at,'state',assertion.state,'evaluation_mode',assertion.evaluation_mode,
    'evaluation_run_id',assertion.evaluation_run_id,'validation_contract_id',assertion.validation_contract_id,
    'validation_contract_version',assertion.validation_contract_version,'validation_contract',assertion.validation_contract,
    'evaluation_engine_version',assertion.evaluation_engine_version,'baseline_snapshot',assertion.baseline_snapshot,
    'baseline_provenance',assertion.baseline_provenance), assertion.created_at
FROM signal_assertions assertion WHERE assertion.tenant_id='tenant-local'
UNION ALL
SELECT 'outcome', outcome.outcome_id, outcome.symbol, outcome.origin_session_date,
  outcome.source_type, outcome.calculation_version, outcome.outcome_status,
  jsonb_build_object('asset_id',outcome.asset_id,'source_id',outcome.source_id,'hypothesis_key',outcome.hypothesis_key,
    'hypothesis_version',outcome.hypothesis_version,'direction',outcome.direction,'horizon_sessions',outcome.horizon_sessions,
    'matured_session_date',outcome.matured_session_date,'forward_return',outcome.forward_return,
    'max_favorable_excursion',outcome.max_favorable_excursion,'max_adverse_excursion',outcome.max_adverse_excursion,
    'maximum_drawdown',outcome.maximum_drawdown,'realized_vol_change',outcome.realized_vol_change,
    'directional_hit',outcome.directional_hit,'threshold_hit',outcome.threshold_hit,'days_to_threshold',outcome.days_to_threshold,
    'origin_event_id',outcome.origin_event_id,'outcome_event_ids',outcome.outcome_event_ids,
    'outcome_payload',outcome.outcome_payload,'calculation_run_id',outcome.calculation_run_id,'deterministic_key',outcome.deterministic_key), outcome.created_at
FROM marketops_signal_outcomes outcome WHERE outcome.tenant_id='tenant-local';

ALTER TABLE subscriber_global_marketops_legacy_parity_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_marketops_legacy_parity_manifest_entries OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_global_marketops_legacy_parity_source OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_marketops_legacy_parity_runs, subscriber_global_marketops_legacy_parity_manifest_entries FROM PUBLIC;
REVOKE ALL ON subscriber_global_marketops_legacy_parity_source FROM PUBLIC;
-- A standard PostgreSQL view checks its owner's access to underlying tables.
-- The migrator is NOLOGIN, so this does not expose legacy rows to any workload.
GRANT SELECT ON marketops_feature_observations, marketops_market_states, marketops_valuation_results,
  marketops_eeom_results, signal_assertions, marketops_signal_outcomes TO signalops_subscriber_migrator;
GRANT SELECT, INSERT ON subscriber_global_marketops_legacy_parity_runs, subscriber_global_marketops_legacy_parity_manifest_entries TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_marketops_legacy_parity_source TO signalops_subscriber_global_eod;
