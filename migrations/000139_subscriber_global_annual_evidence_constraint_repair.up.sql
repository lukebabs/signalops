-- Restore the annual evidence kind after the legacy 000124 repair was found
-- to have been replayed after newer migrations. This is additive/forward-only.
ALTER TABLE subscriber_global_marketops_evidence_runs
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_runs
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check
  CHECK (evidence_kind IN ('eod_bar','feature_vector','market_state','eroc','valuation','eeom','material_event','signal_assertion','outcome','sri_snapshot','options_snapshot','risk_reward','fundamental_annual'));

ALTER TABLE subscriber_global_marketops_evidence_records
  DROP CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_records
  ADD CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check
  CHECK (evidence_kind IN ('eod_bar','feature_vector','market_state','eroc','valuation','eeom','material_event','signal_assertion','outcome','sri_snapshot','options_snapshot','risk_reward','fundamental_annual'));
