DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM marketops_dsm_graph_proposals WHERE proposal_source <> 'dsm_signal') THEN
    RAISE EXCEPTION 'cannot roll back source-aware graph proposals while non-signal proposals exist';
  END IF;
END $$;

DROP INDEX IF EXISTS idx_marketops_dsm_graph_proposals_source_record;
DROP INDEX IF EXISTS idx_marketops_dsm_graph_proposals_source_status;
ALTER TABLE marketops_dsm_graph_proposals
  DROP CONSTRAINT IF EXISTS marketops_dsm_graph_proposals_source_identity_check,
  DROP CONSTRAINT IF EXISTS marketops_dsm_graph_proposals_proposal_source_check;
ALTER TABLE marketops_dsm_graph_proposals
  ALTER COLUMN artifact_id SET NOT NULL, ALTER COLUMN signal_id SET NOT NULL,
  ALTER COLUMN signal_type SET NOT NULL, ALTER COLUMN detector_id SET NOT NULL,
  ALTER COLUMN severity SET NOT NULL, ALTER COLUMN confidence SET NOT NULL,
  DROP COLUMN IF EXISTS lineage_refs, DROP COLUMN IF EXISTS source_refs,
  DROP COLUMN IF EXISTS source_record_version, DROP COLUMN IF EXISTS source_record_id,
  DROP COLUMN IF EXISTS source_record_type, DROP COLUMN IF EXISTS proposal_source;
