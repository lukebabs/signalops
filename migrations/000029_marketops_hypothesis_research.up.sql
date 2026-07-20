-- G138: research-only MarketOps hypothesis registry and evaluation ledger.

CREATE TABLE IF NOT EXISTS marketops_hypothesis_definitions (
  tenant_id text NOT NULL,
  hypothesis_key text NOT NULL,
  hypothesis_version text NOT NULL,
  title text NOT NULL,
  domain text NOT NULL,
  direction text NOT NULL,
  description text NOT NULL,
  rationale text NOT NULL,
  required_features jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(required_features) = 'array'),
  required_transitions jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(required_transitions) = 'array'),
  quality_policy jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(quality_policy) = 'object'),
  eligibility_expression jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(eligibility_expression) = 'object'),
  trigger_expression jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(trigger_expression) = 'object'),
  persistence_rule jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(persistence_rule) = 'object'),
  corroboration_rule jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(corroboration_rule) = 'object'),
  invalidation_rule jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(invalidation_rule) = 'object'),
  expected_outcomes jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(expected_outcomes) = 'array'),
  scoring_config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(scoring_config) = 'object'),
  calibration_policy jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(calibration_policy) = 'object'),
  lifecycle_status text NOT NULL CHECK (lifecycle_status IN ('draft','research','backtest_ready','calibration','candidate','approved','paused','retired')),
  owner text,
  approved_by text,
  approved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, hypothesis_key, hypothesis_version)
);

CREATE INDEX IF NOT EXISTS idx_marketops_hypothesis_definitions_status
  ON marketops_hypothesis_definitions (tenant_id, lifecycle_status, domain, hypothesis_key);

CREATE TABLE IF NOT EXISTS marketops_hypothesis_evaluations (
  evaluation_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  hypothesis_key text NOT NULL,
  hypothesis_version text NOT NULL,
  market_state_id text NOT NULL,
  asset_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  as_of_time timestamptz NOT NULL,
  eligible boolean NOT NULL,
  triggered boolean NOT NULL,
  trigger_score double precision,
  confidence_score double precision,
  magnitude_score double precision,
  rarity_score double precision,
  persistence_score double precision,
  corroboration_score double precision,
  quality_score double precision,
  invalidated boolean NOT NULL DEFAULT false,
  evidence_ids text[] NOT NULL DEFAULT '{}',
  reason_codes text[] NOT NULL DEFAULT '{}',
  evaluation_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(evaluation_payload) = 'object'),
  evaluation_run_id text NOT NULL,
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key),
  FOREIGN KEY (tenant_id, hypothesis_key, hypothesis_version)
    REFERENCES marketops_hypothesis_definitions (tenant_id, hypothesis_key, hypothesis_version),
  FOREIGN KEY (tenant_id, market_state_id)
    REFERENCES marketops_market_states (tenant_id, market_state_id),
  CHECK (NOT triggered OR eligible),
  CHECK (trigger_score IS NULL OR (trigger_score >= 0 AND trigger_score <= 1)),
  CHECK (confidence_score IS NULL OR (confidence_score >= 0 AND confidence_score <= 1)),
  CHECK (magnitude_score IS NULL OR (magnitude_score >= 0 AND magnitude_score <= 1)),
  CHECK (rarity_score IS NULL OR (rarity_score >= 0 AND rarity_score <= 1)),
  CHECK (persistence_score IS NULL OR (persistence_score >= 0 AND persistence_score <= 1)),
  CHECK (corroboration_score IS NULL OR (corroboration_score >= 0 AND corroboration_score <= 1)),
  CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1))
);

CREATE INDEX IF NOT EXISTS idx_marketops_hypothesis_evaluations_state
  ON marketops_hypothesis_evaluations (tenant_id, market_state_id, hypothesis_key);
CREATE INDEX IF NOT EXISTS idx_marketops_hypothesis_evaluations_symbol_session
  ON marketops_hypothesis_evaluations (tenant_id, symbol, session_date DESC, hypothesis_key);
CREATE INDEX IF NOT EXISTS idx_marketops_hypothesis_evaluations_outcome
  ON marketops_hypothesis_evaluations (tenant_id, eligible, triggered, invalidated, session_date DESC);
