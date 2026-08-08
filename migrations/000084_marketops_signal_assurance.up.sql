-- Signal Assurance Framework v1.1: assertion lifecycle and durable evaluation history.
CREATE TABLE signal_validation_contracts (
  contract_id text PRIMARY KEY,
  signal_type text NOT NULL,
  contract_version text NOT NULL,
  algorithm text,
  algorithm_version text,
  direction text NOT NULL CHECK (direction IN ('bullish','bearish')),
  primary_metric text NOT NULL CHECK (primary_metric IN ('absolute_return','benchmark_relative_return')),
  comparison_operator text NOT NULL CHECK (comparison_operator IN ('>=','>','<=','<')),
  threshold double precision NOT NULL,
  evaluation_windows jsonb NOT NULL CHECK (jsonb_typeof(evaluation_windows) = 'array'),
  max_horizon_trading_days integer NOT NULL CHECK (max_horizon_trading_days > 0),
  materialization_policy text NOT NULL,
  invalidation_policy text NOT NULL DEFAULT '',
  config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
  active boolean NOT NULL DEFAULT true,
  contract_scope_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (signal_type, direction, contract_scope_key, contract_version)
);

CREATE TABLE signal_assertions (
  assertion_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  asset_id text NOT NULL,
  symbol text NOT NULL,
  signal_id text NOT NULL,
  source_ledger_signal_id text NOT NULL,
  signal_type text NOT NULL,
  signal_direction text NOT NULL CHECK (signal_direction IN ('bullish','bearish')),
  signal_score double precision,
  algorithm text NOT NULL,
  algorithm_version text NOT NULL,
  confirmed_at timestamptz NOT NULL,
  state text NOT NULL CHECK (state IN ('ACTIVE','MATERIALIZED','INVALIDATED','SUPERSEDED','EXPIRED','CLOSED')),
  evaluation_mode text NOT NULL CHECK (evaluation_mode IN ('LIVE','BACKTEST','RESEARCH')),
  evaluation_run_id text,
  registration_idempotency_key text NOT NULL,
  validation_contract_id text NOT NULL REFERENCES signal_validation_contracts(contract_id),
  validation_contract_version text NOT NULL,
  validation_contract jsonb NOT NULL CHECK (jsonb_typeof(validation_contract) = 'object'),
  evaluation_engine_version text NOT NULL,
  baseline_snapshot jsonb NOT NULL CHECK (jsonb_typeof(baseline_snapshot) = 'object'),
  baseline_provenance jsonb NOT NULL CHECK (jsonb_typeof(baseline_provenance) = 'object'),
  materialized_at timestamptz,
  invalidated_at timestamptz,
  superseded_at timestamptz,
  expired_at timestamptz,
  closed_at timestamptz,
  transition_sequence integer NOT NULL DEFAULT 1 CHECK (transition_sequence > 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, registration_idempotency_key),
  CHECK ((evaluation_mode = 'LIVE' AND evaluation_run_id IS NULL) OR (evaluation_mode <> 'LIVE' AND evaluation_run_id IS NOT NULL))
);

CREATE TABLE signal_assurance_registration_inbox (
  tenant_id text NOT NULL,
  eligible_event_id text NOT NULL,
  signal_ledger_id text NOT NULL,
  assertion_id text NOT NULL REFERENCES signal_assertions(assertion_id),
  payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
  received_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, eligible_event_id)
);

CREATE TABLE signal_assertion_evaluations (
  evaluation_id text PRIMARY KEY,
  assertion_id text NOT NULL REFERENCES signal_assertions(assertion_id),
  evaluated_at timestamptz NOT NULL,
  evaluation_session_date date NOT NULL,
  evaluation_mode text NOT NULL CHECK (evaluation_mode IN ('LIVE','BACKTEST','RESEARCH')),
  evaluation_run_id text,
  input_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input_snapshot) = 'object'),
  input_completeness text NOT NULL CHECK (input_completeness IN ('COMPLETE','INCOMPLETE')),
  transition_sequence integer NOT NULL DEFAULT 0,
  trading_days_active integer NOT NULL CHECK (trading_days_active >= 0),
  calendar_days_active integer NOT NULL CHECK (calendar_days_active >= 0),
  asset_price double precision,
  benchmark_price double precision,
  absolute_return double precision,
  benchmark_return double precision,
  benchmark_relative_return double precision,
  mfe double precision,
  mae double precision,
  materialization_condition_met boolean NOT NULL DEFAULT false,
  invalidation_condition_met boolean NOT NULL DEFAULT false,
  evaluation_version text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (assertion_id, evaluation_session_date, evaluation_version)
);

CREATE TABLE signal_assertion_events (
  event_id text PRIMARY KEY,
  assertion_id text NOT NULL REFERENCES signal_assertions(assertion_id),
  event_type text NOT NULL,
  previous_state text,
  new_state text,
  reason_code text NOT NULL DEFAULT '',
  details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
  occurred_at timestamptz NOT NULL,
  transition_sequence integer NOT NULL,
  evaluation_id text,
  evaluation_mode text NOT NULL CHECK (evaluation_mode IN ('LIVE','BACKTEST','RESEARCH')),
  evaluation_run_id text,
  idempotency_key text NOT NULL,
  published_at timestamptz,
  UNIQUE (assertion_id, transition_sequence, event_type),
  UNIQUE (idempotency_key)
);

CREATE TABLE signal_effectiveness_snapshots (
  snapshot_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  evaluation_mode text NOT NULL CHECK (evaluation_mode IN ('LIVE','BACKTEST','RESEARCH')),
  dimension_key text NOT NULL,
  dimension_value text NOT NULL,
  sample_size integer NOT NULL CHECK (sample_size >= 0),
  materialized_count integer NOT NULL CHECK (materialized_count >= 0),
  censored_count integer NOT NULL CHECK (censored_count >= 0),
  excluded_count integer NOT NULL CHECK (excluded_count >= 0),
  metrics jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metrics) = 'object'),
  as_of timestamptz NOT NULL,
  evaluation_engine_version text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, evaluation_mode, dimension_key, dimension_value, evaluation_engine_version)
);

CREATE INDEX idx_signal_assertions_active ON signal_assertions (tenant_id, state, confirmed_at);
CREATE INDEX idx_signal_assertion_evaluations_assertion ON signal_assertion_evaluations (assertion_id, evaluation_session_date DESC);
CREATE INDEX idx_signal_assurance_inbox_ledger ON signal_assurance_registration_inbox (tenant_id, signal_ledger_id);
