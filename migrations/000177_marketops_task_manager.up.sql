CREATE TABLE IF NOT EXISTS marketops_job_dependency_definitions (
  dependency_id text PRIMARY KEY, job_id text NOT NULL, upstream_job_id text NOT NULL,
  required_statuses text[] NOT NULL DEFAULT ARRAY['succeeded'], max_age interval,
  version text NOT NULL DEFAULT 'task-manager.v1', active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(job_id, upstream_job_id, version)
);
CREATE TABLE IF NOT EXISTS marketops_job_dependency_evaluations (
  evaluation_id text PRIMARY KEY, session_date date NOT NULL, job_id text NOT NULL,
  state text NOT NULL, reason text NOT NULL DEFAULT '', dependencies jsonb NOT NULL DEFAULT '[]',
  correlation_id text NOT NULL, evaluated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS marketops_job_dependency_evaluations_session_idx ON marketops_job_dependency_evaluations(session_date, evaluated_at DESC);
CREATE TABLE IF NOT EXISTS marketops_task_retry_decisions (
  decision_id text PRIMARY KEY, job_id text NOT NULL, session_date date,
  decision text NOT NULL, reason text NOT NULL DEFAULT '', attempt integer NOT NULL DEFAULT 0,
  next_attempt_at timestamptz, actor text NOT NULL DEFAULT '', correlation_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
