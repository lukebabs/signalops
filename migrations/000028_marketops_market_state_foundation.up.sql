-- G136: first-class MarketOps feature, state, transition, and evidence ledgers.

CREATE TABLE IF NOT EXISTS marketops_feature_definitions (
  tenant_id text NOT NULL,
  feature_key text NOT NULL,
  feature_version text NOT NULL,
  domain text NOT NULL,
  title text NOT NULL,
  description text NOT NULL DEFAULT '',
  value_type text NOT NULL CHECK (value_type IN ('numeric', 'text', 'boolean')),
  unit text,
  calculation_spec jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(calculation_spec) = 'object'),
  required_inputs jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(required_inputs) = 'array'),
  quality_policy jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(quality_policy) = 'object'),
  status text NOT NULL CHECK (status IN ('draft', 'active', 'disabled', 'deprecated')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, feature_key, feature_version)
);

CREATE INDEX IF NOT EXISTS idx_marketops_feature_definitions_domain_status
  ON marketops_feature_definitions (tenant_id, domain, status, feature_key);

CREATE TABLE IF NOT EXISTS marketops_feature_observations (
  feature_observation_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  asset_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  as_of_time timestamptz NOT NULL,
  feature_key text NOT NULL,
  feature_version text NOT NULL,
  dimensions jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(dimensions) = 'object'),
  numeric_value double precision,
  text_value text,
  boolean_value boolean,
  quality_state text NOT NULL CHECK (quality_state IN ('usable', 'usable_with_warning', 'partial', 'sparse', 'stale', 'invalid', 'missing', 'not_applicable')),
  quality_score double precision CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1)),
  quality_details jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(quality_details) = 'object'),
  source_event_ids text[] NOT NULL DEFAULT '{}',
  source_artifact_ids text[] NOT NULL DEFAULT '{}',
  calculation_run_id text NOT NULL,
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key),
  FOREIGN KEY (tenant_id, feature_key, feature_version)
    REFERENCES marketops_feature_definitions (tenant_id, feature_key, feature_version),
  CHECK (num_nonnulls(numeric_value, text_value, boolean_value) <= 1),
  CONSTRAINT marketops_feature_observations_usable_value_check CHECK (
    quality_state NOT IN ('usable', 'usable_with_warning')
    OR num_nonnulls(numeric_value, text_value, boolean_value) = 1
  )
);

CREATE INDEX IF NOT EXISTS idx_marketops_feature_observations_symbol_session
  ON marketops_feature_observations (tenant_id, symbol, session_date DESC, as_of_time DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_feature_observations_feature_session
  ON marketops_feature_observations (tenant_id, feature_key, feature_version, session_date DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_feature_observations_quality
  ON marketops_feature_observations (tenant_id, quality_state, session_date DESC);

CREATE TABLE IF NOT EXISTS marketops_market_states (
  market_state_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  asset_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  as_of_time timestamptz NOT NULL,
  state_schema_version text NOT NULL,
  state_payload jsonb NOT NULL CHECK (jsonb_typeof(state_payload) = 'object'),
  feature_observation_ids text[] NOT NULL DEFAULT '{}',
  feature_count integer NOT NULL CHECK (feature_count >= 0),
  required_feature_count integer NOT NULL CHECK (required_feature_count >= 0),
  completeness_ratio double precision NOT NULL CHECK (completeness_ratio >= 0 AND completeness_ratio <= 1),
  quality_state text NOT NULL CHECK (quality_state IN ('usable', 'usable_with_warning', 'partial', 'sparse', 'stale', 'invalid', 'missing', 'not_applicable')),
  quality_score double precision CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1)),
  quality_summary jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(quality_summary) = 'object'),
  eligible_hypotheses text[] NOT NULL DEFAULT '{}',
  build_run_id text NOT NULL,
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key),
  CONSTRAINT marketops_market_states_tenant_state_key UNIQUE (tenant_id, market_state_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_market_states_symbol_session
  ON marketops_market_states (tenant_id, symbol, session_date DESC, as_of_time DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_market_states_quality
  ON marketops_market_states (tenant_id, quality_state, session_date DESC);

CREATE TABLE IF NOT EXISTS marketops_state_transitions (
  transition_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  asset_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  as_of_time timestamptz NOT NULL,
  current_state_id text NOT NULL,
  baseline_state_id text,
  feature_key text NOT NULL,
  feature_version text NOT NULL,
  dimensions jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(dimensions) = 'object'),
  transition_type text NOT NULL,
  lookback_sessions integer CHECK (lookback_sessions IS NULL OR lookback_sessions >= 0),
  current_value double precision,
  baseline_value double precision,
  transition_value double precision,
  zscore double precision,
  percentile double precision CHECK (percentile IS NULL OR (percentile >= 0 AND percentile <= 1)),
  persistence_sessions integer CHECK (persistence_sessions IS NULL OR persistence_sessions >= 0),
  direction text,
  quality_state text NOT NULL CHECK (quality_state IN ('usable', 'usable_with_warning', 'partial', 'sparse', 'stale', 'invalid', 'missing', 'not_applicable')),
  transition_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(transition_payload) = 'object'),
  calculation_run_id text NOT NULL,
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key),
  CONSTRAINT marketops_state_transitions_tenant_current_state_fkey FOREIGN KEY (tenant_id, current_state_id)
    REFERENCES marketops_market_states (tenant_id, market_state_id),
  CONSTRAINT marketops_state_transitions_tenant_baseline_state_fkey FOREIGN KEY (tenant_id, baseline_state_id)
    REFERENCES marketops_market_states (tenant_id, market_state_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_state_transitions_symbol_session
  ON marketops_state_transitions (tenant_id, symbol, session_date DESC, as_of_time DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_state_transitions_state
  ON marketops_state_transitions (tenant_id, current_state_id, transition_type);
CREATE INDEX IF NOT EXISTS idx_marketops_state_transitions_feature
  ON marketops_state_transitions (tenant_id, feature_key, feature_version, session_date DESC);

CREATE TABLE IF NOT EXISTS marketops_evidence (
  evidence_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  asset_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  as_of_time timestamptz NOT NULL,
  evidence_type text NOT NULL,
  evidence_version text NOT NULL,
  domain text NOT NULL,
  direction text,
  magnitude double precision,
  rarity_score double precision CHECK (rarity_score IS NULL OR (rarity_score >= 0 AND rarity_score <= 1)),
  persistence_score double precision CHECK (persistence_score IS NULL OR (persistence_score >= 0 AND persistence_score <= 1)),
  quality_score double precision CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1)),
  statement text NOT NULL,
  evidence_payload jsonb NOT NULL CHECK (jsonb_typeof(evidence_payload) = 'object'),
  source_feature_ids text[] NOT NULL DEFAULT '{}',
  source_transition_ids text[] NOT NULL DEFAULT '{}',
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key)
);

CREATE INDEX IF NOT EXISTS idx_marketops_evidence_symbol_session
  ON marketops_evidence (tenant_id, symbol, session_date DESC, as_of_time DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_evidence_domain_type
  ON marketops_evidence (tenant_id, domain, evidence_type, session_date DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_evidence_source_features
  ON marketops_evidence USING gin (source_feature_ids);
CREATE INDEX IF NOT EXISTS idx_marketops_evidence_source_transitions
  ON marketops_evidence USING gin (source_transition_ids);
