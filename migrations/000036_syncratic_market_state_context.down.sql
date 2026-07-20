ALTER TABLE syncratic_context_windows
  DROP CONSTRAINT IF EXISTS syncratic_context_windows_market_state_cardinality_check;

DROP INDEX IF EXISTS idx_syncratic_context_windows_market_state_ids;

ALTER TABLE syncratic_context_windows
  DROP COLUMN IF EXISTS lineage_refs,
  DROP COLUMN IF EXISTS quality_warnings,
  DROP COLUMN IF EXISTS calibration_summary_ids,
  DROP COLUMN IF EXISTS outcome_ids,
  DROP COLUMN IF EXISTS opportunity_ids,
  DROP COLUMN IF EXISTS hypothesis_evaluation_ids,
  DROP COLUMN IF EXISTS marketops_evidence_ids,
  DROP COLUMN IF EXISTS state_transition_ids,
  DROP COLUMN IF EXISTS market_state_ids,
  DROP COLUMN IF EXISTS context_payload_version;
