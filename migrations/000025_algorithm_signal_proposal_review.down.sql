DROP INDEX IF EXISTS idx_algorithm_signal_proposals_decided;
ALTER TABLE algorithm_signal_proposals
  DROP COLUMN IF EXISTS decision_metadata,
  DROP COLUMN IF EXISTS decided_at,
  DROP COLUMN IF EXISTS decision_note,
  DROP COLUMN IF EXISTS reviewed_by;
