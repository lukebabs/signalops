CREATE TABLE IF NOT EXISTS algorithm_signal_proposals (
  tenant_id text NOT NULL,
  proposal_id text NOT NULL,
  algorithm_result_id text NOT NULL,
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  execution_request_id text NOT NULL,
  proposed_signal_type text NOT NULL,
  status text NOT NULL CHECK (status IN ('proposed', 'reviewed', 'rejected', 'superseded')),
  score double precision NOT NULL CHECK (score >= 0),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  proposal_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  rationale jsonb NOT NULL DEFAULT '{}'::jsonb,
  source_event_ids text[] NOT NULL DEFAULT '{}',
  evidence_refs text[] NOT NULL DEFAULT '{}',
  correlation_id text NOT NULL,
  created_by text NOT NULL DEFAULT 'operator-local',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, proposal_id),
  UNIQUE (tenant_id, algorithm_result_id, proposed_signal_type)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_signal_proposals_algorithm_created
  ON algorithm_signal_proposals (tenant_id, algorithm_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algorithm_signal_proposals_execution
  ON algorithm_signal_proposals (tenant_id, execution_request_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algorithm_signal_proposals_status
  ON algorithm_signal_proposals (tenant_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_algorithm_signal_proposals_result
  ON algorithm_signal_proposals (tenant_id, algorithm_result_id);
