-- Tenant-local legacy hot-universe parity foundation.
--
-- This migration is deliberately inventory-only. It exposes the existing
-- 132-symbol all_active intraday current-state surface through a fixed
-- security-barrier source view, and permits its immutable manifesting. It
-- does not materialize global evidence, change dashboard reads, enable a
-- scheduler, or alter the tenant-local legacy default list.
--
-- The source table is current-only by design. This establishes provenance for
-- its latest states; historical intraday reconstruction remains a later,
-- explicitly approved data-reconciliation step.

ALTER TABLE subscriber_global_marketops_evidence_runs
  DROP CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_runs
  ADD CONSTRAINT subscriber_global_marketops_evidence_runs_evidence_kind_check
  CHECK (evidence_kind IN (
    $$eod_bar$$, $$feature_vector$$, $$market_state$$, $$eroc$$, $$valuation$$, $$eeom$$,
    $$material_event$$, $$signal_assertion$$, $$outcome$$, $$sri_snapshot$$,
    $$options_snapshot$$, $$risk_reward$$, $$fundamental_annual$$, $$intraday_snapshot$$
  ));

ALTER TABLE subscriber_global_marketops_evidence_records
  DROP CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_evidence_records
  ADD CONSTRAINT subscriber_global_marketops_evidence_record_evidence_kind_check
  CHECK (evidence_kind IN (
    $$eod_bar$$, $$feature_vector$$, $$market_state$$, $$eroc$$, $$valuation$$, $$eeom$$,
    $$material_event$$, $$signal_assertion$$, $$outcome$$, $$sri_snapshot$$,
    $$options_snapshot$$, $$risk_reward$$, $$fundamental_annual$$, $$intraday_snapshot$$
  ));

ALTER TABLE subscriber_global_marketops_legacy_parity_manifest_entries
  DROP CONSTRAINT subscriber_global_marketops_legacy_parity_m_evidence_kind_check;
ALTER TABLE subscriber_global_marketops_legacy_parity_manifest_entries
  ADD CONSTRAINT subscriber_global_marketops_legacy_parity_m_evidence_kind_check
  CHECK (evidence_kind IN (
    $$feature_vector$$, $$market_state$$, $$valuation$$, $$eeom$$, $$signal_assertion$$,
    $$outcome$$, $$options_snapshot$$, $$risk_reward$$, $$intraday_snapshot$$
  ));

DO $preflight$
DECLARE
  legacy_symbols integer;
  current_snapshots integer;
BEGIN
  SELECT count(DISTINCT ticker)
    INTO legacy_symbols
  FROM marketops_universal_assets
  WHERE tenant_id = $$tenant-local$$ AND is_active;

  SELECT count(*)
    INTO current_snapshots
  FROM marketops_intraday_condition_snapshots snapshot
  WHERE snapshot.tenant_id = $$tenant-local$$
    AND snapshot.universe_group = $$all_active$$
    AND EXISTS (
      SELECT 1 FROM marketops_universal_assets asset
      WHERE asset.tenant_id = $$tenant-local$$
        AND asset.is_active
        AND upper(asset.ticker) = upper(snapshot.symbol)
    );

  IF legacy_symbols <> 132 THEN
    RAISE EXCEPTION $$legacy hot parity requires exactly 132 active tenant-local symbols; found %$$, legacy_symbols;
  END IF;
  IF current_snapshots <> legacy_symbols THEN
    RAISE EXCEPTION $$legacy hot parity requires one all_active current snapshot per active legacy symbol; snapshots %, symbols %$$, current_snapshots, legacy_symbols;
  END IF;
END;
$preflight$;

CREATE VIEW subscriber_global_marketops_legacy_parity_source_v3
  WITH (security_barrier = true) AS
SELECT *
FROM subscriber_global_marketops_legacy_parity_source_v2
UNION ALL
SELECT
  $$intraday_snapshot$$::text AS evidence_kind,
  snapshot.snapshot_id AS legacy_record_id,
  snapshot.symbol AS legacy_symbol,
  snapshot.as_of_time::date AS legacy_session_date,
  $$marketops.intraday_conditions$$::text AS legacy_algorithm_id,
  $$v1$$::text AS legacy_algorithm_version,
  CASE
    WHEN snapshot.stale THEN $$partial$$
    WHEN jsonb_array_length(snapshot.conditions) > 0 THEN $$usable$$
    ELSE $$partial$$
  END AS legacy_quality_state,
  jsonb_build_object(
    $$snapshot_id$$, snapshot.snapshot_id,
    $$universe_group$$, snapshot.universe_group,
    $$as_of_time$$, snapshot.as_of_time,
    $$market_status$$, snapshot.market_status,
    $$stale$$, snapshot.stale,
    $$conditions$$, snapshot.conditions,
    $$source_payload$$, snapshot.source_payload,
    $$current_only_source$$, true
  ) AS legacy_payload,
  snapshot.created_at AS legacy_created_at
FROM marketops_intraday_condition_snapshots snapshot
WHERE snapshot.tenant_id = $$tenant-local$$
  AND snapshot.universe_group = $$all_active$$
  AND EXISTS (
    SELECT 1 FROM marketops_universal_assets asset
    WHERE asset.tenant_id = $$tenant-local$$
      AND asset.is_active
      AND upper(asset.ticker) = upper(snapshot.symbol)
  );

ALTER VIEW subscriber_global_marketops_legacy_parity_source_v3 OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_marketops_legacy_parity_source_v3 FROM PUBLIC;
GRANT SELECT ON marketops_intraday_condition_snapshots, marketops_universal_assets TO signalops_subscriber_migrator;
GRANT SELECT ON subscriber_global_marketops_legacy_parity_source_v3 TO signalops_subscriber_global_eod;
