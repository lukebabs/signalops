-- Subscriber Project S4: deterministic as-of selection. Historical assurance
-- always selects the immutable initial capture; current market context selects
-- the latest verified global re-observation. Both remain visible as versions.

CREATE TABLE subscriber_global_eod_revision_selection_policies (
  selection_policy_id text PRIMARY KEY,
  usage_context text NOT NULL UNIQUE CHECK (usage_context IN ('historical_assurance', 'current_market_context')),
  selected_observation_role text NOT NULL CHECK (selected_observation_role IN ('initial_tenant_local_capture', 'global_reobservation')),
  policy_version text NOT NULL,
  rationale text NOT NULL,
  effective_at timestamptz NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((usage_context = 'historical_assurance' AND selected_observation_role = 'initial_tenant_local_capture') OR
         (usage_context = 'current_market_context' AND selected_observation_role = 'global_reobservation'))
);

INSERT INTO subscriber_global_eod_revision_selection_policies
  (selection_policy_id,usage_context,selected_observation_role,policy_version,rationale,effective_at,provenance)
VALUES
  ('subeodrevselect_historical_v1','historical_assurance','initial_tenant_local_capture','s4-as-of-selection-v1','Reproducible point-in-time evidence for Signal Assurance, outcomes, and backtests.',now(),'{"as_of_policy":"initial_capture"}'::jsonb),
  ('subeodrevselect_current_v1','current_market_context','global_reobservation','s4-as-of-selection-v1','Latest verified provider response for current MarketOps context and new calculations.',now(),'{"as_of_policy":"latest_verified_provider_revision"}'::jsonb);

CREATE VIEW subscriber_global_eod_resolved_observations AS
SELECT
  policy.selection_policy_id,
  policy.usage_context,
  policy.selected_observation_role,
  policy.policy_version AS selection_policy_version,
  policy.rationale AS selection_rationale,
  observation.revision_observation_id,
  observation.global_asset_id,
  observation.session_date,
  observation.provider,
  observation.observation_role,
  observation.source_event_id,
  observation.source_run_id,
  observation.payload,
  observation.payload_fingerprint,
  observation.algorithm_version,
  observation.quality_state,
  observation.provenance AS observation_provenance,
  observation.observed_at AS as_of_time
FROM subscriber_global_eod_revision_selection_policies policy
JOIN LATERAL (
  SELECT DISTINCT ON (candidate.global_asset_id, candidate.session_date) candidate.*
  FROM subscriber_global_eod_provider_revision_observations candidate
  WHERE candidate.observation_role = policy.selected_observation_role
    AND candidate.quality_state = 'usable'
  ORDER BY candidate.global_asset_id, candidate.session_date, candidate.observed_at DESC, candidate.revision_observation_id DESC
) observation ON true;

ALTER TABLE subscriber_global_eod_revision_selection_policies OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_global_eod_resolved_observations OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_revision_selection_policies, subscriber_global_eod_resolved_observations FROM PUBLIC;
GRANT SELECT ON subscriber_global_eod_revision_selection_policies, subscriber_global_eod_resolved_observations TO signalops_subscriber_global_eod;
