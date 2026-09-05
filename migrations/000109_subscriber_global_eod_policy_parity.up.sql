-- Subscriber Project S4 remediation: record parity against the declared
-- immutable EOD selection context rather than comparing two valid revisions.
CREATE TABLE subscriber_global_eod_canary_policy_parity_reports (
  policy_parity_report_id text PRIMARY KEY,
  live_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_live_runs(live_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  usage_context text NOT NULL CHECK (usage_context IN ('historical_assurance', 'current_market_context')),
  selected_observation_role text NOT NULL CHECK (selected_observation_role IN ('initial_tenant_local_capture', 'global_reobservation')),
  selection_policy_version text NOT NULL,
  comparison_source text NOT NULL CHECK (comparison_source IN ('tenant_local_initial_capture', 'global_canary_baseline')),
  comparison_event_id text NOT NULL DEFAULT '',
  selected_fingerprint text NOT NULL,
  comparison_fingerprint text NOT NULL DEFAULT '',
  parity_status text NOT NULL CHECK (parity_status IN ('matched', 'mismatched', 'missing')),
  mismatch_reason text NOT NULL DEFAULT '',
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  compared_at timestamptz NOT NULL,
  UNIQUE (live_run_id, global_asset_id, usage_context)
);

CREATE INDEX idx_subscriber_global_eod_canary_policy_parity_reports_run
  ON subscriber_global_eod_canary_policy_parity_reports(live_run_id, usage_context);

ALTER TABLE subscriber_global_eod_canary_policy_parity_reports OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_canary_policy_parity_reports FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_eod_canary_policy_parity_reports TO signalops_subscriber_global_eod;
