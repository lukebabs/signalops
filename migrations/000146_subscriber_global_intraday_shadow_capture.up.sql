-- Central intraday provider capture remains shadow-only until a separate
-- reader/cutover approval.  These records contain global symbols and aggregate
-- parity results only; no tenant, subject, or watchlist identity is retained.

CREATE TABLE subscriber_global_intraday_shadow_capture_runs (
  capture_run_id text PRIMARY KEY,
  selector_version text NOT NULL,
  execution_mode text NOT NULL CHECK (execution_mode = 'shadow_provider_capture'),
  market_session text NOT NULL CHECK (market_session IN ('regular', 'extended')),
  selected_count integer NOT NULL CHECK (selected_count >= 0),
  provider_success_count integer NOT NULL CHECK (provider_success_count >= 0),
  provider_failure_count integer NOT NULL CHECK (provider_failure_count >= 0),
  legacy_overlap_count integer NOT NULL CHECK (legacy_overlap_count >= 0),
  freshness_match_count integer NOT NULL CHECK (freshness_match_count >= 0),
  freshness_mismatch_count integer NOT NULL CHECK (freshness_mismatch_count >= 0),
  result_status text NOT NULL CHECK (result_status IN ('complete', 'degraded')),
  validation_contract_ref text NOT NULL,
  immutable_baseline_ref text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  recorded_by text NOT NULL CHECK (recorded_by = 'subscriber-global-eod-reconciler'),
  recorded_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (provider_success_count + provider_failure_count = selected_count),
  CHECK (freshness_match_count + freshness_mismatch_count <= legacy_overlap_count)
);

CREATE TABLE subscriber_global_intraday_shadow_capture_entries (
  capture_run_id text NOT NULL REFERENCES subscriber_global_intraday_shadow_capture_runs(capture_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  canonical_symbol text NOT NULL,
  central_evidence_id text,
  central_as_of_time timestamptz,
  legacy_snapshot_id text NOT NULL DEFAULT '',
  legacy_as_of_time timestamptz,
  comparison_status text NOT NULL CHECK (comparison_status IN ('freshness_match', 'freshness_mismatch', 'legacy_missing', 'provider_failure')),
  failure_class text NOT NULL DEFAULT '',
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY (capture_run_id, global_asset_id),
  CHECK ((comparison_status = 'provider_failure') = (central_evidence_id IS NULL)),
  CHECK ((comparison_status = 'provider_failure') = (failure_class <> ''))
);
CREATE INDEX idx_subscriber_global_intraday_shadow_capture_runs_recorded
  ON subscriber_global_intraday_shadow_capture_runs(recorded_at DESC);

CREATE FUNCTION subscriber_global_intraday_shadow_capture_immutable_guard()
RETURNS trigger LANGUAGE plpgsql AS $guard$
BEGIN
  RAISE EXCEPTION $$subscriber global intraday shadow capture is append-only$$;
END;
$guard$;
CREATE TRIGGER trg_subscriber_global_intraday_shadow_capture_runs_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_intraday_shadow_capture_runs
FOR EACH ROW EXECUTE FUNCTION subscriber_global_intraday_shadow_capture_immutable_guard();
CREATE TRIGGER trg_subscriber_global_intraday_shadow_capture_entries_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_intraday_shadow_capture_entries
FOR EACH ROW EXECUTE FUNCTION subscriber_global_intraday_shadow_capture_immutable_guard();

ALTER TABLE subscriber_global_intraday_shadow_capture_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_intraday_shadow_capture_entries OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_intraday_shadow_capture_runs, subscriber_global_intraday_shadow_capture_entries FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_intraday_shadow_capture_runs, subscriber_global_intraday_shadow_capture_entries TO signalops_subscriber_global_eod;
REVOKE ALL ON FUNCTION subscriber_global_intraday_shadow_capture_immutable_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_intraday_shadow_capture_immutable_guard() TO signalops_subscriber_global_eod;
