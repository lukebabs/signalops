-- Expose implemented MarketOps cross-signal analytics in the Administration Workbench.
INSERT INTO algorithm_definitions (
  tenant_id, algorithm_id, name, description, algorithm_type, runtime_type,
  input_features, input_event_types, output_schema, config_schema,
  default_config, version, status, metadata
) VALUES
(
  'tenant-local', 'signalops.algorithms.options_flow_extreme_v1', 'Options-flow Extremes',
  'Daily descriptive options-flow extreme detector. It identifies put/call contract-volume extremes using completed-session aggregate options data; it is corroboration only and cannot infer trade intent.',
  'anomaly_detection', 'container_plugin',
  ARRAY['put_call_volume_ratio','options_contract_volume'],
  ARRAY['marketops_options_distribution'],
  '{"type":"object","required":["put_call_ratio","extreme_direction","contract_volume","interpretation"]}'::jsonb,
  '{"type":"object"}'::jsonb,
  '{"call_volume_extreme_lt":0.30,"put_volume_extreme_gt":1.20,"minimum_contract_volume":1000}'::jsonb,
  'v1', 'active',
  '{"research_only":true,"marketops_role":"options_flow_extreme","schedule":"daily_postclose","does_not_infer_trade_intent":true}'::jsonb
),
(
  'tenant-local', 'signalops.algorithms.convergence_opportunity_v2', 'Convergence Opportunity Builder',
  'Daily cross-signal research queue builder. It requires same-session agreement from at least two independent MarketOps sources and presents material disagreement as non-directional mixed conviction.',
  'trend_detection', 'container_plugin',
  ARRAY['risk_reward_temporal_v1','eroc_v6','tactical_market_posture_v1','options_flow_extreme_v1'],
  ARRAY['marketops_algorithm_result','marketops_options_distribution'],
  '{"type":"object","required":["agreement_sources","conflict_sources","score_formula","research_only"]}'::jsonb,
  '{"type":"object"}'::jsonb,
  '{"minimum_agreement_sources":2,"material_conflict_strength":0.20}'::jsonb,
  'v2', 'active',
  '{"research_only":true,"marketops_role":"convergence_opportunity","schedule":"daily_postclose","requires_independent_evidence":true}'::jsonb
)
ON CONFLICT (tenant_id, algorithm_id) DO UPDATE SET
  name=EXCLUDED.name, description=EXCLUDED.description, algorithm_type=EXCLUDED.algorithm_type,
  runtime_type=EXCLUDED.runtime_type, input_features=EXCLUDED.input_features,
  input_event_types=EXCLUDED.input_event_types, output_schema=EXCLUDED.output_schema,
  config_schema=EXCLUDED.config_schema, default_config=EXCLUDED.default_config,
  version=EXCLUDED.version, status=EXCLUDED.status, metadata=EXCLUDED.metadata, updated_at=now();
