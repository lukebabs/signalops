-- SAF-2: versioned read-model policy for multi-horizon signal usefulness.
-- This migration records the policy contract only. It does not rewrite SAF
-- assertions, historical outcomes, benchmark observations, or algorithm output.

INSERT INTO subscriber_global_saf_viability_policies (
  policy_id,
  policy_version,
  cohort_list_id,
  benchmark_calculation_version,
  rules,
  policy_document_ref
)
VALUES (
  'safpolicy_usefulness_v1',
  'saf_usefulness.v1',
  'sublist-tenant-local-legacy-default',
  'saf_benchmark.v4',
  '{
    "research_only": true,
    "operational_cutoff_date": "2026-08-20",
    "default_trading_day_window": 10,
    "maximum_standard_trading_day_window": 20,
    "evaluation_horizons": [1, 5, 10, 20],
    "lifecycle_states": ["confirmed", "developing", "materialized", "outperformed", "adverse_warning", "invalidated", "expired", "censored"],
    "score_components": {
      "directional_resolution": 0.25,
      "materialization_strength": 0.25,
      "adverse_excursion_control": 0.20,
      "benchmark_relative_performance": 0.20,
      "timeliness_persistence": 0.10
    },
    "one_day_adverse_move_is_not_automatic_miss": true,
    "missing_benchmark_or_price_is_explicit_coverage_gap": true,
    "immutable_baseline_required": true,
    "provider_polling_required": false
  }'::jsonb,
  'docs/projects/subscriber_project/saf_multi_horizon_usefulness_sprint.md'
)
ON CONFLICT (policy_version) DO NOTHING;
