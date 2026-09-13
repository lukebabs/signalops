-- Subscriber Project S4 follow-up: preserve provider revisions rather than
-- overwriting initial capture evidence. The default policy intentionally holds
-- the existing tenant-local result authoritative until a later review changes
-- the canonical-selection policy.

CREATE TABLE subscriber_global_eod_provider_revision_policies (
  policy_id text PRIMARY KEY,
  policy_version text NOT NULL,
  provider text NOT NULL CHECK (provider = 'massive'),
  initial_capture_treatment text NOT NULL CHECK (initial_capture_treatment = 'immutable'),
  revised_capture_treatment text NOT NULL CHECK (revised_capture_treatment = 'immutable_version'),
  canonical_selection text NOT NULL CHECK (canonical_selection = 'hold_initial_pending_review'),
  effective_at timestamptz NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_global_eod_provider_revision_observations (
  revision_observation_id text PRIMARY KEY,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  session_date date NOT NULL,
  provider text NOT NULL CHECK (provider = 'massive'),
  observation_role text NOT NULL CHECK (observation_role IN ('initial_tenant_local_capture', 'global_reobservation')),
  source_event_id text NOT NULL DEFAULT '',
  source_run_id text NOT NULL DEFAULT '',
  payload jsonb NOT NULL,
  payload_fingerprint text NOT NULL,
  algorithm_version text NOT NULL,
  quality_state text NOT NULL CHECK (quality_state IN ('usable', 'invalid')),
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  observed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (global_asset_id, session_date, observation_role, source_event_id, source_run_id)
);
CREATE INDEX idx_subscriber_global_eod_provider_revision_observations_lookup
  ON subscriber_global_eod_provider_revision_observations(global_asset_id, session_date, observed_at);

CREATE TABLE subscriber_global_eod_provider_revision_field_deltas (
  revision_delta_id text PRIMARY KEY,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  session_date date NOT NULL,
  initial_revision_observation_id text NOT NULL REFERENCES subscriber_global_eod_provider_revision_observations(revision_observation_id) ON DELETE RESTRICT,
  revised_revision_observation_id text NOT NULL REFERENCES subscriber_global_eod_provider_revision_observations(revision_observation_id) ON DELETE RESTRICT,
  field_name text NOT NULL CHECK (field_name IN ('open', 'high', 'low', 'close', 'volume', 'vwap')),
  initial_value jsonb NOT NULL,
  revised_value jsonb NOT NULL,
  delta_class text NOT NULL CHECK (delta_class IN ('unchanged', 'provider_revision')),
  materiality text NOT NULL CHECK (materiality IN ('informational', 'review_required')),
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  observed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (initial_revision_observation_id, revised_revision_observation_id, field_name)
);

ALTER TABLE subscriber_global_eod_provider_revision_policies OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_provider_revision_observations OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_provider_revision_field_deltas OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_provider_revision_policies, subscriber_global_eod_provider_revision_observations, subscriber_global_eod_provider_revision_field_deltas FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_eod_provider_revision_observations, subscriber_global_eod_provider_revision_field_deltas TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_eod_provider_revision_policies TO signalops_subscriber_global_eod;
