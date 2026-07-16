CREATE TABLE IF NOT EXISTS algorithm_signal_materializations (
  materialization_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  proposal_id TEXT NOT NULL,
  algorithm_result_id TEXT NOT NULL,
  execution_request_id TEXT NOT NULL,
  algorithm_id TEXT NOT NULL,
  algorithm_version TEXT NOT NULL,
  proposed_signal_type TEXT NOT NULL,
  signal_id TEXT,
  materialization_status TEXT NOT NULL,
  materialization_policy_version TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  duplicate_of_signal_id TEXT,
  requested_by TEXT NOT NULL,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  failed_at TIMESTAMPTZ,
  error_code TEXT,
  error_message TEXT,
  request_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  preflight_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  signal_payload_preview JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_signal_materializations_status
  ON algorithm_signal_materializations (tenant_id, materialization_status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_algorithm_signal_materializations_proposal
  ON algorithm_signal_materializations (tenant_id, proposal_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_algorithm_signal_materializations_signal
  ON algorithm_signal_materializations (tenant_id, signal_id)
  WHERE signal_id IS NOT NULL;
