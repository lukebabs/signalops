-- Platform-global annual financial evidence for the parallel VC/DOSM v4
-- profile. The payload remains immutable provider evidence; a later reader
-- and scorer decide which locally-derived ratios are usable.

ALTER TABLE subscriber_global_marketops_evidence_runs
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check,
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_execution_mode_check;
ALTER TABLE subscriber_global_marketops_evidence_runs
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check
  CHECK (evidence_kind IN ('eod_bar','feature_vector','market_state','eroc','valuation','eeom','material_event','signal_assertion','outcome','sri_snapshot','options_snapshot','risk_reward','fundamental_annual')),
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_execution_mode_check
  CHECK (execution_mode IN ('shadow_read_only','legacy_materialized','provider_capture'));

ALTER TABLE subscriber_global_marketops_evidence_records
  DROP CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_records
  ADD CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check
  CHECK (evidence_kind IN ('eod_bar','feature_vector','market_state','eroc','valuation','eeom','material_event','signal_assertion','outcome','sri_snapshot','options_snapshot','risk_reward','fundamental_annual'));
