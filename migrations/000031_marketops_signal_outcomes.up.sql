-- G140: immutable forward outcomes for MarketOps research sources.

CREATE TABLE IF NOT EXISTS marketops_signal_outcomes (
  outcome_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  source_type text NOT NULL CHECK (source_type IN ('hypothesis_evaluation','opportunity','signal')),
  source_id text NOT NULL,
  hypothesis_key text,
  hypothesis_version text,
  asset_id text NOT NULL,
  symbol text NOT NULL,
  direction text NOT NULL CHECK (direction IN ('upside','downside','non_directional')),
  origin_session_date date NOT NULL,
  horizon_sessions integer NOT NULL CHECK (horizon_sessions IN (1,5,10,20)),
  matured_session_date date,
  outcome_status text NOT NULL CHECK (outcome_status IN ('pending','matured','missing_price')),
  forward_return double precision,
  max_favorable_excursion double precision,
  max_adverse_excursion double precision,
  maximum_drawdown double precision,
  realized_vol_change double precision,
  directional_hit boolean,
  threshold_hit boolean,
  days_to_threshold integer CHECK (days_to_threshold IS NULL OR days_to_threshold > 0),
  origin_event_id text,
  outcome_event_ids text[] NOT NULL DEFAULT '{}',
  outcome_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(outcome_payload) = 'object'),
  calculation_version text NOT NULL,
  calculation_run_id text NOT NULL,
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key),
  CHECK (
    (outcome_status = 'matured' AND matured_session_date IS NOT NULL AND forward_return IS NOT NULL)
    OR (
      outcome_status <> 'matured' AND matured_session_date IS NULL
      AND forward_return IS NULL AND max_favorable_excursion IS NULL
      AND max_adverse_excursion IS NULL AND maximum_drawdown IS NULL
      AND realized_vol_change IS NULL AND directional_hit IS NULL
      AND threshold_hit IS NULL AND days_to_threshold IS NULL
    )
  )
);

CREATE INDEX IF NOT EXISTS idx_marketops_signal_outcomes_source
  ON marketops_signal_outcomes (tenant_id, source_type, source_id, horizon_sessions);
CREATE INDEX IF NOT EXISTS idx_marketops_signal_outcomes_symbol_origin
  ON marketops_signal_outcomes (tenant_id, symbol, origin_session_date DESC, horizon_sessions);
CREATE INDEX IF NOT EXISTS idx_marketops_signal_outcomes_status
  ON marketops_signal_outcomes (tenant_id, outcome_status, horizon_sessions, origin_session_date DESC);
