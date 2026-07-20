-- Generalize the existing reviewed graph-proposal ledger for persisted MarketOps intelligence records.

ALTER TABLE marketops_dsm_graph_proposals
  ADD COLUMN IF NOT EXISTS proposal_source text NOT NULL DEFAULT 'dsm_signal',
  ADD COLUMN IF NOT EXISTS source_record_type text NOT NULL DEFAULT 'dsm_signal',
  ADD COLUMN IF NOT EXISTS source_record_id text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS source_record_version text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS source_refs jsonb NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS lineage_refs jsonb NOT NULL DEFAULT '{}'::jsonb;

UPDATE marketops_dsm_graph_proposals
SET proposal_source = 'dsm_signal', source_record_type = 'dsm_signal', source_record_id = signal_id,
    source_record_version = detector_id,
    source_refs = jsonb_build_object('artifact_id', artifact_id, 'signal_id', signal_id),
    lineage_refs = jsonb_build_object('event_ids', to_jsonb(event_ids))
WHERE source_record_id = '';

ALTER TABLE marketops_dsm_graph_proposals
  ALTER COLUMN artifact_id DROP NOT NULL, ALTER COLUMN signal_id DROP NOT NULL,
  ALTER COLUMN signal_type DROP NOT NULL, ALTER COLUMN detector_id DROP NOT NULL,
  ALTER COLUMN severity DROP NOT NULL, ALTER COLUMN confidence DROP NOT NULL;

ALTER TABLE marketops_dsm_graph_proposals
  ADD CONSTRAINT marketops_dsm_graph_proposals_proposal_source_check CHECK (
    proposal_source IN ('dsm_signal','market_state','state_transition','hypothesis_definition','hypothesis_evaluation','opportunity','outcome')
  ),
  ADD CONSTRAINT marketops_dsm_graph_proposals_source_identity_check CHECK (
    (proposal_source = 'dsm_signal' AND artifact_id IS NOT NULL AND artifact_id <> '' AND signal_id IS NOT NULL AND signal_id <> ''
      AND signal_type IS NOT NULL AND signal_type <> '' AND detector_id IS NOT NULL AND detector_id <> ''
      AND severity IS NOT NULL AND confidence IS NOT NULL)
    OR (proposal_source <> 'dsm_signal' AND source_record_type <> '' AND source_record_id <> '')
  );

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_source_status
  ON marketops_dsm_graph_proposals (tenant_id, proposal_source, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_source_record
  ON marketops_dsm_graph_proposals (tenant_id, source_record_type, source_record_id);
