CREATE TABLE marketops_equity_reconciliation_tasks (
  task_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  source_id text NOT NULL,
  universe_group text NOT NULL,
  dataset text NOT NULL DEFAULT 'equity_eod_prices'
    CHECK (dataset = 'equity_eod_prices'),
  observation_date date NOT NULL,
  symbol text NOT NULL,
  universe_rank integer NOT NULL CHECK (universe_rank > 0 AND universe_rank <= 50),
  status text NOT NULL
    CHECK (status IN ('queued', 'running', 'awaiting_normalization', 'succeeded', 'failed')),
  provider_attempts integer NOT NULL DEFAULT 0 CHECK (provider_attempts >= 0),
  max_provider_attempts integer NOT NULL CHECK (max_provider_attempts BETWEEN 1 AND 3),
  replay_count integer NOT NULL DEFAULT 0 CHECK (replay_count >= 0 AND replay_count <= 1),
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  lease_expires_at timestamptz,
  raw_event_id text,
  idempotency_key text,
  last_error text,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, source_id, universe_group, dataset, observation_date, symbol)
);

CREATE INDEX idx_marketops_equity_reconciliation_due
  ON marketops_equity_reconciliation_tasks (status, next_attempt_at, observation_date, universe_rank);

CREATE INDEX idx_marketops_equity_reconciliation_session
  ON marketops_equity_reconciliation_tasks (
    tenant_id, source_id, universe_group, observation_date, status, universe_rank
  );
