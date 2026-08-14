-- Subscriber production-blocker remediation: platform-owned analytical
-- evidence foundation. This migration is additive and deliberately permits
-- only shadow/read-only reconciliation runs. It neither imports legacy rows,
-- changes a scheduler, nor makes a gateway projection selectable.

CREATE TABLE subscriber_global_marketops_evidence_runs (
  evidence_run_id text PRIMARY KEY,
  evidence_kind text NOT NULL CHECK (evidence_kind IN (
    'eod_bar', 'feature_vector', 'market_state', 'eroc', 'valuation', 'eeom',
    'material_event', 'signal_assertion', 'outcome', 'sri_snapshot', 'options_snapshot'
  )),
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  execution_mode text NOT NULL CHECK (execution_mode = 'shadow_read_only'),
  source_scope text NOT NULL CHECK (source_scope IN ('global_provider_capture', 'legacy_parity_review')),
  session_start_date date,
  session_end_date date,
  input_manifest_fingerprint text NOT NULL,
  validation_contract_ref text NOT NULL,
  immutable_baseline_ref text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  recorded_by text NOT NULL CHECK (recorded_by = 'subscriber-global-eod-reconciler'),
  correlation_id text NOT NULL DEFAULT '',
  recorded_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (session_start_date IS NULL OR session_end_date IS NULL OR session_start_date <= session_end_date)
);

CREATE TABLE subscriber_global_marketops_evidence_records (
  global_evidence_id text PRIMARY KEY,
  evidence_run_id text NOT NULL REFERENCES subscriber_global_marketops_evidence_runs(evidence_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  session_date date NOT NULL,
  evidence_kind text NOT NULL CHECK (evidence_kind IN (
    'eod_bar', 'feature_vector', 'market_state', 'eroc', 'valuation', 'eeom',
    'material_event', 'signal_assertion', 'outcome', 'sri_snapshot', 'options_snapshot'
  )),
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  quality_state text NOT NULL CHECK (quality_state IN ('usable', 'partial', 'invalid')),
  source_system text NOT NULL CHECK (source_system IN ('massive', 'fmp', 'state_street', 'marketops', 'legacy_parity_review')),
  source_event_id text NOT NULL DEFAULT '',
  source_run_id text NOT NULL DEFAULT '',
  evidence_fingerprint text NOT NULL,
  validation_contract_ref text NOT NULL,
  immutable_baseline_ref text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  observed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (global_asset_id, session_date, evidence_kind, algorithm_id, algorithm_version, evidence_fingerprint)
);
CREATE INDEX idx_subscriber_global_marketops_evidence_records_projection
  ON subscriber_global_marketops_evidence_records(global_asset_id, evidence_kind, session_date DESC, observed_at DESC);
CREATE INDEX idx_subscriber_global_marketops_evidence_records_run
  ON subscriber_global_marketops_evidence_records(evidence_run_id, global_asset_id);

CREATE FUNCTION subscriber_global_marketops_evidence_run_match_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  run_kind text;
  run_algorithm_id text;
  run_algorithm_version text;
BEGIN
  SELECT evidence_kind, algorithm_id, algorithm_version
    INTO run_kind, run_algorithm_id, run_algorithm_version
  FROM subscriber_global_marketops_evidence_runs
  WHERE evidence_run_id = NEW.evidence_run_id;
  IF NOT FOUND OR NEW.evidence_kind <> run_kind
    OR NEW.algorithm_id <> run_algorithm_id
    OR NEW.algorithm_version <> run_algorithm_version THEN
    RAISE EXCEPTION 'global evidence record must match its immutable run identity';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER trg_subscriber_global_marketops_evidence_run_match
BEFORE INSERT ON subscriber_global_marketops_evidence_records
FOR EACH ROW EXECUTE FUNCTION subscriber_global_marketops_evidence_run_match_guard();

-- Evidence is never edited into a new meaning. A correction is a new record
-- with a new fingerprint and explicit provenance, selected only by a later,
-- separately reviewed projection policy.
CREATE FUNCTION subscriber_global_marketops_evidence_immutable_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'subscriber global MarketOps evidence is append-only; record a new provenance version instead';
END;
$$;
CREATE TRIGGER trg_subscriber_global_marketops_evidence_runs_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_marketops_evidence_runs
FOR EACH ROW EXECUTE FUNCTION subscriber_global_marketops_evidence_immutable_guard();
CREATE TRIGGER trg_subscriber_global_marketops_evidence_records_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_marketops_evidence_records
FOR EACH ROW EXECUTE FUNCTION subscriber_global_marketops_evidence_immutable_guard();

-- This is intentionally a coverage-only gateway projection. It makes no
-- legacy result authoritative and cannot be used to render a market state,
-- score, or signal until a type-specific projection and parity gate exist.
CREATE VIEW subscriber_gateway_global_marketops_evidence_coverage WITH (security_barrier = true) AS
SELECT
  record.global_asset_id,
  asset.canonical_symbol,
  record.evidence_kind,
  max(record.session_date) AS latest_session_date,
  max(record.observed_at) AS latest_observed_at,
  count(*) FILTER (WHERE record.quality_state = 'usable')::bigint AS usable_record_count,
  count(*) FILTER (WHERE record.quality_state = 'partial')::bigint AS partial_record_count,
  count(*) FILTER (WHERE record.quality_state = 'invalid')::bigint AS invalid_record_count
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
WHERE run.source_scope = 'global_provider_capture'
GROUP BY record.global_asset_id, asset.canonical_symbol, record.evidence_kind;

ALTER TABLE subscriber_global_marketops_evidence_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_marketops_evidence_records OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_marketops_evidence_coverage OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_marketops_evidence_runs, subscriber_global_marketops_evidence_records FROM PUBLIC;
REVOKE ALL ON subscriber_gateway_global_marketops_evidence_coverage FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_marketops_evidence_runs, subscriber_global_marketops_evidence_records TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_gateway_global_marketops_evidence_coverage TO signalops_subscriber_gateway;
REVOKE ALL ON FUNCTION subscriber_global_marketops_evidence_immutable_guard() FROM PUBLIC;
REVOKE ALL ON FUNCTION subscriber_global_marketops_evidence_run_match_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_marketops_evidence_immutable_guard() TO signalops_subscriber_global_eod;
GRANT EXECUTE ON FUNCTION subscriber_global_marketops_evidence_run_match_guard() TO signalops_subscriber_global_eod;
