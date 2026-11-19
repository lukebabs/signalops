-- Dedicated platform-global task control for annual FMP capture. It is
-- deliberately separate from tenant MarketOps task tables: the global worker
-- cannot read or mutate any tenant task, workflow, list, or asset record.

CREATE TABLE subscriber_global_annual_financial_workflows (
  workflow_id text PRIMARY KEY,
  session_date date NOT NULL UNIQUE,
  status text NOT NULL CHECK (status IN ('queued','running','succeeded','degraded','failed')),
  schedule_job_id text NOT NULL DEFAULT 'marketops-fmp-annual-financial',
  coverage jsonb NOT NULL DEFAULT '{}'::jsonb,
  failure_class text NOT NULL DEFAULT '',
  error_message text NOT NULL DEFAULT '',
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_global_annual_financial_tasks (
  task_id text PRIMARY KEY,
  workflow_id text NOT NULL REFERENCES subscriber_global_annual_financial_workflows(workflow_id) ON DELETE CASCADE,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  symbol text NOT NULL,
  status text NOT NULL CHECK (status IN ('queued','running','retry_scheduled','succeeded','skipped_no_data','blocked_entitlement','deferred_quota','failed_terminal')),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  max_attempts integer NOT NULL DEFAULT 3 CHECK (max_attempts BETWEEN 1 AND 10),
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  lease_expires_at timestamptz,
  failure_class text NOT NULL DEFAULT '',
  provider_status integer,
  error_message text NOT NULL DEFAULT '',
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (workflow_id, global_asset_id)
);
CREATE INDEX idx_subscriber_global_annual_financial_tasks_due ON subscriber_global_annual_financial_tasks(status,next_attempt_at,workflow_id);

CREATE VIEW subscriber_gateway_global_annual_financial_tasks WITH (security_barrier=true) AS
SELECT task.task_id,task.workflow_id,'platform-global'::text AS tenant_id,workflow.session_date,
  'subscriber_global_annual_financial'::text AS task_type,task.symbol,task.status,task.attempt_count,
  task.max_attempts,task.next_attempt_at,task.lease_expires_at,'fmp'::text AS provider,
  task.failure_class,task.provider_status,task.error_message,task.result,task.completed_at,
  task.created_at,task.updated_at
FROM subscriber_global_annual_financial_tasks task
JOIN subscriber_global_annual_financial_workflows workflow ON workflow.workflow_id=task.workflow_id;

ALTER TABLE subscriber_global_annual_financial_workflows OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_annual_financial_tasks OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_annual_financial_tasks OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_annual_financial_workflows,subscriber_global_annual_financial_tasks FROM PUBLIC;
REVOKE ALL ON subscriber_gateway_global_annual_financial_tasks FROM PUBLIC;
GRANT SELECT,INSERT,UPDATE ON subscriber_global_annual_financial_workflows,subscriber_global_annual_financial_tasks TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_gateway_global_annual_financial_tasks TO signalops_subscriber_gateway;
