-- Register daily MarketOps algorithms so the Administration Workbench exposes
-- their runtime contract alongside the existing VC/DOSM and Risk/Reward models.
INSERT INTO algorithm_definitions (
  tenant_id, algorithm_id, name, description, algorithm_type, runtime_type,
  input_features, input_event_types, output_schema, config_schema,
  default_config, version, status, metadata
) VALUES
(
  'tenant-local',
  'signalops.algorithms.eroc_v6',
  'Exhaustive Reversal',
  'Daily deterministic reversal-readiness assessment. It combines ten-session directional persistence and price extension with options-flow and volume-exhaustion context; it prioritizes analyst review and is not a trade instruction.',
  'trend_detection',
  'container_plugin',
  ARRAY['eod_close','eod_volume','put_call_volume_ratio'],
  ARRAY['marketops_eod_event','marketops_options_distribution'],
  '{"type":"object","required":["stance_score","regime","reversal_candidate","evidence_complete","price_score","flow_score","volume_score"]}'::jsonb,
  '{"type":"object"}'::jsonb,
  '{"directional_window":10,"minimum_eod_history":21,"persistence_threshold":0.8,"extension_units":3}'::jsonb,
  'v6.1',
  'active',
  '{"research_only":true,"marketops_role":"exhaustive_reversal","schedule":"daily_postclose"}'::jsonb
),
(
  'tenant-local',
  'signalops.algorithms.tactical_market_posture_v1',
  'Tactical Market Posture',
  'Daily deterministic EOD technical overlay. RSI reversal, SMA trend, and five-day price extension produce a constructive, neutral, or caution posture; it provides current-condition context and does not alter weekly VC/DOSM strategic scores.',
  'trend_detection',
  'container_plugin',
  ARRAY['rsi_14','return_5d','distance_sma_50_pct','distance_sma_200_pct','sma_50_slope_20d_pct'],
  ARRAY['marketops_feature_observation'],
  '{"type":"object","required":["technical_overlay","posture","technical_components","feature_observation_ids"]}'::jsonb,
  '{"type":"object"}'::jsonb,
  '{"overlay_range":[-1.5,1.5],"constructive_threshold":0.5,"caution_threshold":-0.5}'::jsonb,
  'v1',
  'active',
  '{"research_only":true,"marketops_role":"tactical_market_posture","schedule":"daily_postclose","does_not_modify_strategic_scores":true}'::jsonb
)
ON CONFLICT (tenant_id, algorithm_id) DO UPDATE SET
  name=EXCLUDED.name, description=EXCLUDED.description, algorithm_type=EXCLUDED.algorithm_type,
  runtime_type=EXCLUDED.runtime_type, input_features=EXCLUDED.input_features,
  input_event_types=EXCLUDED.input_event_types, output_schema=EXCLUDED.output_schema,
  config_schema=EXCLUDED.config_schema, default_config=EXCLUDED.default_config,
  version=EXCLUDED.version, status=EXCLUDED.status, metadata=EXCLUDED.metadata, updated_at=now();
