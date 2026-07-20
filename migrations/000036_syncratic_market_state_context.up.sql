ALTER TABLE syncratic_context_windows
  ADD COLUMN IF NOT EXISTS context_payload_version text NOT NULL DEFAULT 'signalops.syncratic.context_payload.v1',
  ADD COLUMN IF NOT EXISTS market_state_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS state_transition_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS marketops_evidence_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS hypothesis_evaluation_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS opportunity_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS outcome_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS calibration_summary_ids text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS quality_warnings jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS lineage_refs jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_syncratic_context_windows_market_state_ids
  ON syncratic_context_windows USING gin (market_state_ids);

ALTER TABLE syncratic_context_windows
  ADD CONSTRAINT syncratic_context_windows_market_state_cardinality_check
  CHECK (context_strategy <> 'market_state_session_v1' OR cardinality(market_state_ids) = 1)
  NOT VALID;
