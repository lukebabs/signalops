CREATE TABLE IF NOT EXISTS marketops_fmp_daily_budgets (
  tenant_id text NOT NULL,
  provider_day date NOT NULL,
  max_calls integer NOT NULL CHECK (max_calls >= 0),
  reserved_calls integer NOT NULL DEFAULT 0 CHECK (reserved_calls >= 0),
  completed_calls integer NOT NULL DEFAULT 0 CHECK (completed_calls >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, provider_day)
);
CREATE TABLE IF NOT EXISTS marketops_fmp_poll_states (
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  status text NOT NULL DEFAULT 'pending',
  last_success_at timestamptz,
  next_eligible_at timestamptz,
  attempt_count integer NOT NULL DEFAULT 0,
  last_provider_status integer,
  last_error text NOT NULL DEFAULT '',
  financial_snapshot_id text,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, symbol)
);
CREATE INDEX IF NOT EXISTS idx_marketops_fmp_poll_states_queue ON marketops_fmp_poll_states (tenant_id, status, next_eligible_at, updated_at);
