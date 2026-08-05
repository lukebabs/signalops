CREATE TABLE marketops_task_workflows (
  workflow_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  session_date date NOT NULL,
  workflow_type text NOT NULL,
  status text NOT NULL CHECK (status IN ('queued','running','succeeded','degraded','failed')),
  schedule_job_id text NOT NULL DEFAULT '',
  coverage jsonb NOT NULL DEFAULT '{}'::jsonb,
  failure_class text NOT NULL DEFAULT '',
  error_message text NOT NULL DEFAULT '',
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, session_date, workflow_type)
);

CREATE TABLE marketops_task_items (
  task_id text PRIMARY KEY,
  workflow_id text NOT NULL REFERENCES marketops_task_workflows(workflow_id) ON DELETE CASCADE,
  tenant_id text NOT NULL,
  session_date date NOT NULL,
  task_type text NOT NULL,
  symbol text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('queued','running','retry_scheduled','succeeded','skipped_no_data','blocked_entitlement','deferred_quota','failed_terminal','superseded')),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  max_attempts integer NOT NULL DEFAULT 3 CHECK (max_attempts BETWEEN 1 AND 10),
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  lease_expires_at timestamptz,
  provider text NOT NULL DEFAULT '',
  failure_class text NOT NULL DEFAULT '',
  provider_status integer,
  error_message text NOT NULL DEFAULT '',
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, session_date, task_type, symbol)
);

CREATE INDEX idx_marketops_task_items_due ON marketops_task_items (status, next_attempt_at, session_date, task_type);
CREATE INDEX idx_marketops_task_items_workflow ON marketops_task_items (workflow_id, status, symbol);
