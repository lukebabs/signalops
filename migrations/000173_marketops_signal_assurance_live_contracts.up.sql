-- Seed prospective LIVE Signal Assurance validation contracts for the current
-- governed algorithm-signal materialization family. This is additive policy
-- metadata only; it does not create assertions, evaluations, outcomes, or
-- provider requests.
INSERT INTO signal_validation_contracts (
  contract_id,
  signal_type,
  contract_version,
  algorithm,
  algorithm_version,
  direction,
  primary_metric,
  comparison_operator,
  threshold,
  evaluation_windows,
  max_horizon_trading_days,
  materialization_policy,
  invalidation_policy,
  config,
  active,
  contract_scope_key
) VALUES
  (
    'saf_contract_live_change_point_candidate_v1_bullish',
    'signalops.algorithm.change_point_candidate',
    'live-default-v1',
    'signalops.algorithms.ruptures_change_point_v1',
    'v1',
    'bullish',
    'absolute_return',
    '>=',
    0.05,
    '[1,5,10,20]'::jsonb,
    20,
    'threshold_multi_horizon',
    'adverse_excursion_warning',
    '{"research_only":false,"operational_scope":"prospective_only","mfe_threshold":0.05,"adverse_warning_threshold":-0.10,"metric_definition_version":"saf_usefulness.v1","notes":"Seed contract for future directional algorithm materializations; no historical assertion backfill."}'::jsonb,
    true,
    'signalops.algorithms.ruptures_change_point_v1|v1'
  ),
  (
    'saf_contract_live_change_point_candidate_v1_bearish',
    'signalops.algorithm.change_point_candidate',
    'live-default-v1',
    'signalops.algorithms.ruptures_change_point_v1',
    'v1',
    'bearish',
    'absolute_return',
    '<=',
    -0.05,
    '[1,5,10,20]'::jsonb,
    20,
    'threshold_multi_horizon',
    'adverse_excursion_warning',
    '{"research_only":false,"operational_scope":"prospective_only","mfe_threshold":0.05,"adverse_warning_threshold":-0.10,"metric_definition_version":"saf_usefulness.v1","notes":"Seed contract for future directional algorithm materializations; no historical assertion backfill."}'::jsonb,
    true,
    'signalops.algorithms.ruptures_change_point_v1|v1'
  )
ON CONFLICT (contract_id) DO NOTHING;

INSERT INTO schema_migrations (version)
VALUES ('000173_marketops_signal_assurance_live_contracts')
ON CONFLICT (version) DO NOTHING;
