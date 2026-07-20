DROP TABLE IF EXISTS marketops_opportunity_dispositions;

DELETE FROM algorithm_signal_proposals WHERE proposal_source = 'hypothesis_evaluation';

DROP INDEX IF EXISTS idx_signal_proposals_hypothesis_queue;
DROP INDEX IF EXISTS idx_signal_proposals_hypothesis_source;
DROP INDEX IF EXISTS idx_signal_proposals_algorithm_source;

ALTER TABLE algorithm_signal_proposals
  DROP CONSTRAINT IF EXISTS algorithm_signal_proposals_materialization_check,
  DROP CONSTRAINT IF EXISTS algorithm_signal_proposals_source_check,
  DROP COLUMN IF EXISTS eligibility_snapshot,
  DROP COLUMN IF EXISTS materialization_eligible,
  DROP COLUMN IF EXISTS research_only,
  DROP COLUMN IF EXISTS hypothesis_lifecycle_status,
  DROP COLUMN IF EXISTS hypothesis_version,
  DROP COLUMN IF EXISTS hypothesis_key,
  DROP COLUMN IF EXISTS hypothesis_evaluation_id,
  DROP COLUMN IF EXISTS proposal_source;

ALTER TABLE algorithm_signal_proposals
  ALTER COLUMN algorithm_result_id SET NOT NULL,
  ALTER COLUMN algorithm_id SET NOT NULL,
  ALTER COLUMN algorithm_version SET NOT NULL,
  ALTER COLUMN execution_request_id SET NOT NULL,
  ADD CONSTRAINT algorithm_signal_proposals_result_signal_key
    UNIQUE (tenant_id, algorithm_result_id, proposed_signal_type);
