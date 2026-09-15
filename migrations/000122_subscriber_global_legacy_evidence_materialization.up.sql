-- Subscriber global analytical-data-plane: permit only a parity-gated,
-- append-only materialization of records already mapped in an immutable
-- tenant-local parity manifest.  This makes no provider request and does not
-- alter a legacy record or make any projection selectable by itself.

ALTER TABLE subscriber_global_marketops_evidence_runs
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_execution_mode_check,
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_source_scope_check;

ALTER TABLE subscriber_global_marketops_evidence_runs
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_execution_mode_check
    CHECK (execution_mode IN ('shadow_read_only', 'legacy_materialized')),
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_source_scope_check
    CHECK (source_scope IN ('global_provider_capture', 'legacy_parity_review', 'legacy_materialization'));

-- The coverage view remains metadata only.  It may now report materialized
-- legacy evidence, but no score, state, signal, or recommendation can be read
-- through it until its type-specific projection has its own parity/UX gate.
CREATE OR REPLACE VIEW subscriber_gateway_global_marketops_evidence_coverage WITH (security_barrier = true) AS
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
WHERE run.source_scope IN ('global_provider_capture', 'legacy_materialization')
GROUP BY record.global_asset_id, asset.canonical_symbol, record.evidence_kind;
