-- Down migrations are for non-production rollback before any dependent
-- projection is enabled.  Preserve append-only behavior while removing only
-- records created by the legacy materialization scope.
DROP TRIGGER IF EXISTS trg_subscriber_global_marketops_evidence_records_immutable ON subscriber_global_marketops_evidence_records;
DROP TRIGGER IF EXISTS trg_subscriber_global_marketops_evidence_runs_immutable ON subscriber_global_marketops_evidence_runs;
DELETE FROM subscriber_global_marketops_evidence_records
WHERE evidence_run_id IN (
  SELECT evidence_run_id FROM subscriber_global_marketops_evidence_runs
  WHERE source_scope = 'legacy_materialization'
);
DELETE FROM subscriber_global_marketops_evidence_runs WHERE source_scope = 'legacy_materialization';
CREATE TRIGGER trg_subscriber_global_marketops_evidence_runs_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_marketops_evidence_runs
FOR EACH ROW EXECUTE FUNCTION subscriber_global_marketops_evidence_immutable_guard();
CREATE TRIGGER trg_subscriber_global_marketops_evidence_records_immutable
BEFORE UPDATE OR DELETE ON subscriber_global_marketops_evidence_records
FOR EACH ROW EXECUTE FUNCTION subscriber_global_marketops_evidence_immutable_guard();

ALTER TABLE subscriber_global_marketops_evidence_runs
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_execution_mode_check,
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_source_scope_check;
ALTER TABLE subscriber_global_marketops_evidence_runs
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_execution_mode_check
    CHECK (execution_mode = 'shadow_read_only'),
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_source_scope_check
    CHECK (source_scope IN ('global_provider_capture', 'legacy_parity_review'));

CREATE OR REPLACE VIEW subscriber_gateway_global_marketops_evidence_coverage WITH (security_barrier = true) AS
SELECT record.global_asset_id, asset.canonical_symbol, record.evidence_kind,
  max(record.session_date) AS latest_session_date, max(record.observed_at) AS latest_observed_at,
  count(*) FILTER (WHERE record.quality_state = 'usable')::bigint AS usable_record_count,
  count(*) FILTER (WHERE record.quality_state = 'partial')::bigint AS partial_record_count,
  count(*) FILTER (WHERE record.quality_state = 'invalid')::bigint AS invalid_record_count
FROM subscriber_global_marketops_evidence_records record
JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = record.global_asset_id
WHERE run.source_scope = 'global_provider_capture'
GROUP BY record.global_asset_id, asset.canonical_symbol, record.evidence_kind;
