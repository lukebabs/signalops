CREATE TABLE IF NOT EXISTS marketops_options_capture_sessions (
  capture_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  provider text NOT NULL DEFAULT 'massive',
  source_id text NOT NULL DEFAULT '',
  run_id text NOT NULL,
  status text NOT NULL CHECK (status IN ('analytics_ready', 'partial', 'no_data', 'failed')),
  analytics_ready boolean NOT NULL DEFAULT false,
  contract_count integer NOT NULL DEFAULT 0 CHECK (contract_count >= 0),
  usable_iv_count integer NOT NULL DEFAULT 0 CHECK (usable_iv_count >= 0),
  usable_greeks_count integer NOT NULL DEFAULT 0 CHECK (usable_greeks_count >= 0),
  open_interest_count integer NOT NULL DEFAULT 0 CHECK (open_interest_count >= 0),
  required_surface_cells integer NOT NULL DEFAULT 0 CHECK (required_surface_cells BETWEEN 0 AND 5),
  quality_reasons jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(quality_reasons) = 'array'),
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metrics) = 'object'),
  error_message text NOT NULL DEFAULT '',
  attempt_count integer NOT NULL DEFAULT 1 CHECK (attempt_count > 0),
  started_at timestamptz NOT NULL,
  completed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, symbol, session_date, provider)
);

CREATE INDEX IF NOT EXISTS idx_marketops_options_capture_tenant_date
  ON marketops_options_capture_sessions (tenant_id, session_date DESC, symbol);

CREATE INDEX IF NOT EXISTS idx_marketops_options_capture_readiness
  ON marketops_options_capture_sessions (tenant_id, analytics_ready, session_date DESC);
