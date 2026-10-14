-- Subscriber platform-global SRI foundation.
-- The established tenant-local SRI materialization is common market context,
-- not tenant-owned analytics. Seed a platform-owned projection once, retaining
-- explicit legacy input provenance. New SRI writers target platform-global.

INSERT INTO sri_segments (tenant_id, segment_id, segment_key, name, segment_type, parent_segment_key, active, registry_version, metadata)
SELECT 'platform-global', segment_id, segment_key, name, segment_type, parent_segment_key, active, registry_version,
  metadata || jsonb_build_object('source_scope', 'legacy_materialization', 'source_tenant_id', 'tenant-local')
FROM sri_segments WHERE tenant_id = 'tenant-local'
ON CONFLICT (tenant_id, segment_id) DO NOTHING;

INSERT INTO sri_etf_registry (tenant_id, etf_symbol, segment_id, role, benchmark_priority, active, registry_version, config)
SELECT 'platform-global', etf_symbol, segment_id, role, benchmark_priority, active, registry_version,
  config || jsonb_build_object('source_scope', 'legacy_materialization', 'source_tenant_id', 'tenant-local')
FROM sri_etf_registry WHERE tenant_id = 'tenant-local'
ON CONFLICT (tenant_id, etf_symbol, segment_id, registry_version) DO NOTHING;

INSERT INTO sri_segment_snapshots (snapshot_id, tenant_id, segment_id, session_date, as_of_time, state, composite_score, relative_strength_score, momentum_score, momentum_acceleration, rank, rank_change_5d, evidence_quality, quality_state, quality_flags, components, input_provenance, algorithm_version, configuration_version, calculation_run_id, deterministic_key)
SELECT 'sri_global_' || md5(snapshot_id), 'platform-global', segment_id, session_date, as_of_time, state,
  composite_score, relative_strength_score, momentum_score, momentum_acceleration, rank, rank_change_5d, evidence_quality, quality_state, quality_flags, components,
  input_provenance || jsonb_build_object('source_scope', 'legacy_materialization', 'source_tenant_id', 'tenant-local', 'source_snapshot_id', snapshot_id),
  algorithm_version, configuration_version, calculation_run_id,
  'platform-global|' || segment_id || '|' || session_date::text || '|' || algorithm_version
FROM sri_segment_snapshots WHERE tenant_id = 'tenant-local'
ON CONFLICT (tenant_id, segment_id, session_date, algorithm_version) DO NOTHING;

INSERT INTO sri_etf_holdings_snapshots (snapshot_id, tenant_id, etf_symbol, fund_name, effective_date, retrieved_at, source, source_url, content_hash, holdings_count, total_weight, top_ten_weight)
SELECT 'sri_global_holding_' || md5(snapshot_id), 'platform-global', etf_symbol, fund_name, effective_date, retrieved_at,
  source, source_url, content_hash, holdings_count, total_weight, top_ten_weight
FROM sri_etf_holdings_snapshots WHERE tenant_id = 'tenant-local'
ON CONFLICT (tenant_id, etf_symbol, effective_date, source, content_hash) DO NOTHING;

INSERT INTO sri_etf_holdings (snapshot_id, holding_key, holding_rank, ticker, name, identifier, sedol, sector, currency, weight, shares_held)
SELECT 'sri_global_holding_' || md5(snapshot.snapshot_id), holding.holding_key, holding.holding_rank,
  holding.ticker, holding.name, holding.identifier, holding.sedol, holding.sector, holding.currency, holding.weight, holding.shares_held
FROM sri_etf_holdings holding
JOIN sri_etf_holdings_snapshots snapshot ON snapshot.snapshot_id = holding.snapshot_id
WHERE snapshot.tenant_id = 'tenant-local'
ON CONFLICT (snapshot_id, holding_key) DO NOTHING;

CREATE OR REPLACE VIEW subscriber_gateway_global_sri_segments WITH (security_barrier = true) AS
SELECT tenant_id, segment_id, segment_key, name, segment_type, parent_segment_key, active, registry_version, metadata
FROM sri_segments WHERE tenant_id = 'platform-global';
CREATE OR REPLACE VIEW subscriber_gateway_global_sri_etf_registry WITH (security_barrier = true) AS
SELECT tenant_id, etf_symbol, segment_id, role, benchmark_priority, active, registry_version, config
FROM sri_etf_registry WHERE tenant_id = 'platform-global';
CREATE OR REPLACE VIEW subscriber_gateway_global_sri_snapshots WITH (security_barrier = true) AS
SELECT snapshot_id, tenant_id, segment_id, session_date, as_of_time, state, composite_score, relative_strength_score, momentum_score, momentum_acceleration,
  rank, rank_change_5d, evidence_quality, quality_state, quality_flags, components, input_provenance, algorithm_version, configuration_version, calculation_run_id, deterministic_key
FROM sri_segment_snapshots WHERE tenant_id = 'platform-global';
CREATE OR REPLACE VIEW subscriber_gateway_global_sri_etf_holdings_snapshots WITH (security_barrier = true) AS
SELECT snapshot_id, tenant_id, etf_symbol, fund_name, effective_date, retrieved_at, source, source_url, content_hash, holdings_count, total_weight, top_ten_weight
FROM sri_etf_holdings_snapshots WHERE tenant_id = 'platform-global';
CREATE OR REPLACE VIEW subscriber_gateway_global_sri_etf_holdings WITH (security_barrier = true) AS
SELECT holding.snapshot_id, holding.holding_key, holding.holding_rank, holding.ticker, holding.name, holding.identifier, holding.sedol, holding.sector, holding.currency, holding.weight, holding.shares_held
FROM sri_etf_holdings holding
JOIN subscriber_gateway_global_sri_etf_holdings_snapshots snapshot ON snapshot.snapshot_id = holding.snapshot_id;

ALTER VIEW subscriber_gateway_global_sri_segments OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_sri_etf_registry OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_sri_snapshots OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_sri_etf_holdings_snapshots OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_gateway_global_sri_etf_holdings OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_sri_segments, subscriber_gateway_global_sri_etf_registry, subscriber_gateway_global_sri_snapshots, subscriber_gateway_global_sri_etf_holdings_snapshots, subscriber_gateway_global_sri_etf_holdings FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_sri_segments, subscriber_gateway_global_sri_etf_registry, subscriber_gateway_global_sri_snapshots, subscriber_gateway_global_sri_etf_holdings_snapshots, subscriber_gateway_global_sri_etf_holdings TO signalops_subscriber_gateway;
