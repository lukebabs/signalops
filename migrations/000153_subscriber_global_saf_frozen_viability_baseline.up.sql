-- SAF-V3a: immutable, research-only viability-policy and baseline evidence.
-- This does not alter outcomes, algorithms, subscriptions, or schedules.

CREATE TABLE subscriber_global_saf_viability_policies (
  policy_id text PRIMARY KEY,
  policy_version text NOT NULL UNIQUE,
  cohort_list_id text NOT NULL,
  benchmark_calculation_version text NOT NULL,
  rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object'),
  policy_document_ref text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_global_saf_viability_baselines (
  baseline_id text PRIMARY KEY,
  policy_id text NOT NULL REFERENCES subscriber_global_saf_viability_policies(policy_id) ON DELETE RESTRICT,
  cohort_list_id text NOT NULL,
  member_count integer NOT NULL CHECK (member_count > 0),
  observation_count integer NOT NULL CHECK (observation_count >= 0),
  cutoff_matured_session_date date NOT NULL,
  assessment_state text NOT NULL CHECK (assessment_state IN ('not_demonstrated','research_supported_in_sample')),
  assessment_reasons jsonb NOT NULL CHECK (jsonb_typeof(assessment_reasons) = 'array'),
  metrics jsonb NOT NULL CHECK (jsonb_typeof(metrics) = 'object'),
  selection_provenance jsonb NOT NULL CHECK (jsonb_typeof(selection_provenance) = 'object'),
  frozen_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (policy_id, cohort_list_id, cutoff_matured_session_date)
);

CREATE FUNCTION subscriber_global_saf_viability_baseline_immutable_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'SAF viability policies and baselines are append-only; apply a new policy or baseline version';
END;
$$;
CREATE TRIGGER trg_subscriber_global_saf_viability_policy_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_saf_viability_policies
FOR EACH ROW EXECUTE FUNCTION subscriber_global_saf_viability_baseline_immutable_guard();
CREATE TRIGGER trg_subscriber_global_saf_viability_baseline_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_saf_viability_baselines
FOR EACH ROW EXECUTE FUNCTION subscriber_global_saf_viability_baseline_immutable_guard();

INSERT INTO subscriber_global_saf_viability_policies (policy_id,policy_version,cohort_list_id,benchmark_calculation_version,rules,policy_document_ref)
VALUES ('safpolicy_legacy_v1','saf_viability.v1','sublist-tenant-local-legacy-default','saf_benchmark.v4',
  '{"minimum_matured_observations":30,"directional_accuracy_lower_bound_must_exceed":0.5,"average_directional_benchmark_excess_must_be_positive":true,"average_mfe_must_exceed_absolute_average_mae":true,"research_only":true}'::jsonb,
  'docs/projects/signal_assurance_viability/README.md')
ON CONFLICT (policy_version) DO NOTHING;

INSERT INTO subscriber_global_saf_viability_baselines (baseline_id,policy_id,cohort_list_id,member_count,observation_count,cutoff_matured_session_date,assessment_state,assessment_reasons,metrics,selection_provenance)
VALUES ('safbaseline_legacy_v1_20260814','safpolicy_legacy_v1','sublist-tenant-local-legacy-default',132,92,DATE '2026-08-14','not_demonstrated',
  '["Directional accuracy is 46/92 (50.0%); its 95% lower bound does not exceed the 50% reference.","Mean favorable excursion (2.310%) does not exceed absolute mean adverse excursion (3.858%)."]'::jsonb,
  '{"directional_hits":46,"directional_accuracy":0.5,"average_directional_broad_market_excess":0.000410,"average_directional_sector_excess":0.000610,"average_mfe":0.023097,"average_mae":-0.038575,"benchmark_coverage":{"broad_market_matched":92,"sector_matched":92}}'::jsonb,
  '{"selection_policy":"historical_assurance_initial_capture.v1","benchmark_calculation_version":"saf_benchmark.v4","outcomes_immutable":true,"provider_requests":0,"research_only":true}')
ON CONFLICT (policy_id,cohort_list_id,cutoff_matured_session_date) DO NOTHING;

ALTER TABLE subscriber_global_saf_viability_policies OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_saf_viability_baselines OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_saf_viability_policies, subscriber_global_saf_viability_baselines FROM PUBLIC;
GRANT SELECT ON subscriber_global_saf_viability_policies, subscriber_global_saf_viability_baselines TO signalops_subscriber_gateway;
REVOKE ALL ON FUNCTION subscriber_global_saf_viability_baseline_immutable_guard() FROM PUBLIC;
