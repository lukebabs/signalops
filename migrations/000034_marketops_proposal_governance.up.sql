-- G146: governed hypothesis proposal bridge and append-only opportunity disposition ledger.

ALTER TABLE algorithm_signal_proposals
  ADD COLUMN IF NOT EXISTS proposal_source text NOT NULL DEFAULT 'algorithm_result',
  ADD COLUMN IF NOT EXISTS hypothesis_evaluation_id text,
  ADD COLUMN IF NOT EXISTS hypothesis_key text,
  ADD COLUMN IF NOT EXISTS hypothesis_version text,
  ADD COLUMN IF NOT EXISTS hypothesis_lifecycle_status text,
  ADD COLUMN IF NOT EXISTS research_only boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS materialization_eligible boolean NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS eligibility_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE algorithm_signal_proposals
  ALTER COLUMN algorithm_result_id DROP NOT NULL,
  ALTER COLUMN algorithm_id DROP NOT NULL,
  ALTER COLUMN algorithm_version DROP NOT NULL,
  ALTER COLUMN execution_request_id DROP NOT NULL;

ALTER TABLE algorithm_signal_proposals
  DROP CONSTRAINT IF EXISTS algorithm_signal_proposals_tenant_id_algorithm_result_id_pr_key,
  DROP CONSTRAINT IF EXISTS algorithm_signal_proposals_result_signal_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_proposals_algorithm_source
  ON algorithm_signal_proposals (tenant_id, algorithm_result_id, proposed_signal_type)
  WHERE proposal_source = 'algorithm_result';

CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_proposals_hypothesis_source
  ON algorithm_signal_proposals (tenant_id, hypothesis_evaluation_id, proposed_signal_type)
  WHERE proposal_source = 'hypothesis_evaluation';

CREATE INDEX IF NOT EXISTS idx_signal_proposals_hypothesis_queue
  ON algorithm_signal_proposals (tenant_id, hypothesis_key, hypothesis_version, status, created_at DESC)
  WHERE proposal_source = 'hypothesis_evaluation';

ALTER TABLE algorithm_signal_proposals
  DROP CONSTRAINT IF EXISTS algorithm_signal_proposals_source_check,
  ADD CONSTRAINT algorithm_signal_proposals_source_check CHECK (
    (proposal_source = 'algorithm_result'
      AND algorithm_result_id IS NOT NULL AND algorithm_id IS NOT NULL
      AND algorithm_version IS NOT NULL AND execution_request_id IS NOT NULL
      AND hypothesis_evaluation_id IS NULL)
    OR
    (proposal_source = 'hypothesis_evaluation'
      AND hypothesis_evaluation_id IS NOT NULL AND hypothesis_key IS NOT NULL
      AND hypothesis_version IS NOT NULL AND hypothesis_lifecycle_status IN ('candidate','approved')
      AND algorithm_result_id IS NULL AND algorithm_id IS NULL
      AND algorithm_version IS NULL AND execution_request_id IS NULL)
  ),
  DROP CONSTRAINT IF EXISTS algorithm_signal_proposals_materialization_check,
  ADD CONSTRAINT algorithm_signal_proposals_materialization_check CHECK (
    NOT materialization_eligible
    OR (NOT research_only AND (proposal_source = 'algorithm_result' OR hypothesis_lifecycle_status = 'approved'))
  );

CREATE TABLE IF NOT EXISTS marketops_opportunity_dispositions (
  disposition_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  opportunity_id text NOT NULL,
  disposition text NOT NULL CHECK (disposition IN ('watch','advance','needs_more_evidence','dismiss','resolved')),
  actor text NOT NULL,
  note text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (opportunity_id) REFERENCES marketops_opportunities (opportunity_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_opportunity_dispositions_queue
  ON marketops_opportunity_dispositions (tenant_id, opportunity_id, created_at DESC);
