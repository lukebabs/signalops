-- Generic foundational primitive definition registry.
-- Published semantic versions are append-only by contract; mutable updates are limited to
-- draft/validating rows and seeded local catalog alignment.

CREATE TABLE IF NOT EXISTS platform_primitive_definitions (
  tenant_id text NOT NULL,
  primitive_type text NOT NULL CHECK (primitive_type IN (
    'source', 'dataset', 'schema', 'pipeline', 'feature', 'state_model', 'detector',
    'algorithm', 'signal_definition', 'policy', 'artifact_type', 'proposal_type',
    'insight_type', 'outcome_definition'
  )),
  definition_id text NOT NULL,
  definition_key text NOT NULL,
  version text NOT NULL,
  app_id text NOT NULL DEFAULT '',
  domain text NOT NULL DEFAULT '',
  use_case text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('draft', 'validating', 'active', 'deprecated', 'retired')),
  schema_ref text NOT NULL DEFAULT '',
  implementation_ref text NOT NULL DEFAULT '',
  quality_policy_id text NOT NULL DEFAULT '',
  retention_policy_id text NOT NULL DEFAULT '',
  lineage_policy_id text NOT NULL DEFAULT '',
  point_in_time_required boolean NOT NULL DEFAULT true,
  contract jsonb NOT NULL DEFAULT '{}'::jsonb,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, primitive_type, definition_key, version),
  UNIQUE (tenant_id, definition_id)
);

CREATE INDEX IF NOT EXISTS idx_platform_primitive_definitions_type_status
  ON platform_primitive_definitions (tenant_id, primitive_type, status, definition_key, version);
CREATE INDEX IF NOT EXISTS idx_platform_primitive_definitions_app_domain
  ON platform_primitive_definitions (tenant_id, app_id, domain, use_case, primitive_type);

INSERT INTO platform_primitive_definitions (
  tenant_id, primitive_type, definition_id, definition_key, version, app_id, domain, use_case,
  status, schema_ref, implementation_ref, quality_policy_id, retention_policy_id,
  lineage_policy_id, point_in_time_required, contract, metadata
) VALUES
  ('tenant-local', 'source', 'src-massive', 'massive-market-data-prod', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'adapter:market_data.massive', 'policy_marketops_source_quality_v1', 'policy_marketops_raw_retention_v1', 'policy_signalops_lineage_v1', true, '{"provider":"massive","source_type":"market_data_provider","auth_type":"secret_reference","allowed_dataset_keys":["market.equity.daily_bar","market.options.contracts_daily","market.options.chain_eod"]}'::jsonb, '{"catalog_source_id":"src-massive"}'::jsonb),
  ('tenant-local', 'dataset', 'dset-market-equity-daily-bar-v1', 'market.equity.daily_bar', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', 'contracts/events/normalized_signal_event.v1.schema.json', 'normalizers/market/equity_eod_prices:v1', 'policy_marketops_eod_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"record_type":"observation","entity_types":["asset"],"temporal_semantics":{"occurred_at_required":true,"session_date_supported":true},"legacy_dataset":"equity_eod_prices"}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'dataset', 'dset-market-options-contracts-daily-v1', 'market.options.contracts_daily', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', 'contracts/events/normalized_signal_event.v1.schema.json', 'normalizers/market/options_contracts_daily:v1', 'policy_marketops_options_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"record_type":"observation","entity_types":["asset","option_contract"],"temporal_semantics":{"occurred_at_required":true,"session_date_supported":true},"legacy_dataset":"option_contracts_daily"}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'dataset', 'dset-market-options-chain-eod-v1', 'market.options.chain_eod', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', 'contracts/events/normalized_signal_event.v1.schema.json', 'normalizers/market/options_chain_eod:v1', 'policy_marketops_options_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"record_type":"snapshot","entity_types":["asset","option_contract"],"temporal_semantics":{"occurred_at_required":true,"session_date_supported":true}}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'pipeline', 'pipeline-massive-raw-ingest', 'marketops.massive.raw_ingest', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'pipeline:scheduled_pull/raw_event_build/broker_publish/raw_ledger_persist', 'policy_marketops_ingest_quality_v1', 'policy_marketops_raw_retention_v1', 'policy_signalops_lineage_v1', true, '{"input_dataset_keys":["market.equity.daily_bar","market.options.contracts_daily"],"output_topics":["signalops.local.raw.v1"]}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'feature', 'featdef-market-daily-return-pct-v1', 'market.price.return_1d', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'features/market/daily_return_pct:v1', 'policy_marketops_feature_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"value_type":"decimal","unit":"ratio","required_dataset_versions":[{"dataset_key":"market.equity.daily_bar","version_range":"^1.0.0"}],"missing_value_policy":"no_observation"}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'feature', 'featdef-market-options-distribution-v1', 'market.options.distribution_daily', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'features/market/options_distribution_daily:v1', 'policy_marketops_options_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"value_type":"object","unit":"snapshot","required_dataset_versions":[{"dataset_key":"market.options.chain_eod","version_range":"^1.0.0"}],"missing_value_policy":"quality_block"}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'state_model', 'statemodel-market-asset-daily-state-v1', 'market.asset.daily_state', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'states/market/asset_daily:v1', 'policy_marketops_state_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"subject_entity_type":"asset","required_features":["market.price.return_1d","market.options.distribution_daily"],"lateness_policy":"rebuild_and_supersede"}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'algorithm', 'alg-signalops-zscore-anomaly-v1', 'signalops.algorithms.zscore_anomaly', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'signalops.algorithms.zscore_anomaly_v1', 'policy_algorithm_input_quality_v1', 'policy_algorithm_result_retention_v1', 'policy_signalops_lineage_v1', true, '{"runtime":"python","determinism":"deterministic","mode_isolation":["research","backtest","shadow","production_evaluation"]}'::jsonb, '{"legacy_algorithm_id":"signalops.algorithms.zscore_anomaly_v1"}'::jsonb),
  ('tenant-local', 'signal_definition', 'signaldef-market-algorithm-anomaly-candidate-v1', 'market.algorithm.anomaly_candidate', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'validating', '', 'signals/market/algorithm_anomaly_candidate:v1', 'policy_signal_quality_v1', 'policy_signal_lifecycle_retention_v1', 'policy_signalops_lineage_v1', true, '{"required_evidence_types":["algorithm_result"],"lifecycle":["candidate","evaluating","accepted","active","resolved","rejected","suppressed"]}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'proposal_type', 'proposaltype-marketops-graph-update-v1', 'marketops.graph_update', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'proposals/marketops/graph_update:v1', 'policy_proposal_quality_v1', 'policy_governance_retention_v1', 'policy_signalops_lineage_v1', true, '{"governed_materialization":true,"review_required":true}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'outcome_definition', 'outcomedef-market-forward-return-v1', 'market.forward_return', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'outcomes/market/forward_return:v1', 'policy_outcome_quality_v1', 'policy_marketops_temporal_retention_v1', 'policy_signalops_lineage_v1', true, '{"horizons_sessions":[1,5,10,20],"source_dataset_key":"market.equity.daily_bar","point_in_time_correct":true}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'schema', 'schema-normalized-signal-event-v1', 'signalops.normalized_event', '1.0.0', '', '', '', 'active', 'contracts/events/normalized_signal_event.v1.schema.json', 'schema_registry/jsonschema:v1', 'policy_signalops_quality_states_v1', 'policy_schema_retention_v1', 'policy_signalops_lineage_v1', true, '{"schema_format":"json_schema","contract_boundary":"normalized_event"}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'detector', 'detector-marketops-dsm-taxonomy-v1', 'marketops.dsm.taxonomy', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'validating', '', 'detectors/marketops/dsm_taxonomy:v1', 'policy_detector_input_quality_v1', 'policy_detector_result_retention_v1', 'policy_signalops_lineage_v1', true, '{"input_contract":{"normalized_events":["market.options.contracts_daily"]},"required_quality_states":["usable","degraded"]}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'artifact_type', 'artifacttype-marketops-dsm-signal-v1', 'marketops.dsm.signal_artifact', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'active', '', 'artifacts/marketops/dsm_signal:v1', 'policy_artifact_quality_v1', 'policy_artifact_retention_v1', 'policy_signalops_lineage_v1', true, '{"requires_evidence_ids":true,"governed_materialization":false}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'insight_type', 'insighttype-marketops-evidence-brief-v1', 'marketops.evidence_brief', '1.0.0', 'marketops', 'markets', 'daily_market_surveillance', 'validating', '', 'insights/marketops/evidence_brief:v1', 'policy_insight_quality_v1', 'policy_insight_retention_v1', 'policy_signalops_lineage_v1', true, '{"requires_evidence_ids":true,"external_action_authorized":false}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'policy', 'policy_signalops_quality_states_v1', 'signalops.quality_states', '1.0.0', '', '', '', 'active', '', 'policy/quality_states:v1', '', '', 'policy_signalops_lineage_v1', true, '{"states":["usable","degraded","partial","stale","missing","invalid","contradictory","not_applicable","suppressed"],"fails_closed":true}'::jsonb, '{}'::jsonb),
  ('tenant-local', 'policy', 'policy_signalops_lineage_v1', 'signalops.complete_lineage', '1.0.0', '', '', '', 'active', '', 'policy/complete_lineage:v1', '', '', '', true, '{"required_fields":["tenant_id","app_id","domain","use_case","dataset","processing_run_id","observation_time","processing_time","correlation_id","causation_id","quality_state","definition_version"]}'::jsonb, '{}'::jsonb)
ON CONFLICT (tenant_id, primitive_type, definition_key, version) DO UPDATE SET
  definition_id = EXCLUDED.definition_id,
  app_id = EXCLUDED.app_id,
  domain = EXCLUDED.domain,
  use_case = EXCLUDED.use_case,
  status = EXCLUDED.status,
  schema_ref = EXCLUDED.schema_ref,
  implementation_ref = EXCLUDED.implementation_ref,
  quality_policy_id = EXCLUDED.quality_policy_id,
  retention_policy_id = EXCLUDED.retention_policy_id,
  lineage_policy_id = EXCLUDED.lineage_policy_id,
  point_in_time_required = EXCLUDED.point_in_time_required,
  contract = EXCLUDED.contract,
  metadata = EXCLUDED.metadata,
  updated_at = now();


-- Every policy reference is itself a durable policy definition. The generic policy
-- rows keep governance dependencies resolvable even when their executable policy
-- implementation is introduced in a later platform phase.
WITH policy_references AS (
  SELECT tenant_id, quality_policy_id AS policy_id FROM platform_primitive_definitions WHERE quality_policy_id <> ''
  UNION
  SELECT tenant_id, retention_policy_id FROM platform_primitive_definitions WHERE retention_policy_id <> ''
  UNION
  SELECT tenant_id, lineage_policy_id FROM platform_primitive_definitions WHERE lineage_policy_id <> ''
)
INSERT INTO platform_primitive_definitions (
  tenant_id, primitive_type, definition_id, definition_key, version, app_id, domain, use_case,
  status, implementation_ref, point_in_time_required, contract, metadata
)
SELECT
  tenant_id,
  'policy',
  policy_id,
  CASE policy_id
    WHEN 'policy_signalops_quality_states_v1' THEN 'signalops.quality_states'
    WHEN 'policy_signalops_lineage_v1' THEN 'signalops.complete_lineage'
    ELSE substr(policy_id, length('policy_') + 1)
  END,
  '1.0.0',
  CASE WHEN policy_id LIKE 'policy_marketops_%' THEN 'marketops' ELSE '' END,
  CASE WHEN policy_id LIKE 'policy_marketops_%' THEN 'markets' ELSE '' END,
  CASE WHEN policy_id LIKE 'policy_marketops_%' THEN 'daily_market_surveillance' ELSE '' END,
  'active',
  'registry:policy_reference:v1',
  true,
  jsonb_build_object(
    'policy_id', policy_id,
    'policy_class', CASE
      WHEN policy_id LIKE '%quality%' THEN 'quality_gate'
      WHEN policy_id LIKE '%retention%' THEN 'retention'
      WHEN policy_id LIKE '%lineage%' THEN 'lineage'
      ELSE 'governance'
    END
  ),
  jsonb_build_object('registry_seed', 'platform_primitive_definitions_v1')
FROM policy_references
ON CONFLICT (tenant_id, primitive_type, definition_key, version) DO NOTHING;
