ALTER TABLE algorithm_signal_proposals
  ADD COLUMN IF NOT EXISTS reviewed_by text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS decision_note text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS decided_at timestamptz,
  ADD COLUMN IF NOT EXISTS decision_metadata jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_algorithm_signal_proposals_decided
  ON algorithm_signal_proposals (tenant_id, status, decided_at DESC)
  WHERE decided_at IS NOT NULL;
