-- K8s global projection workers run through the non-login runtime role and
-- assume this controlled role. Keep parity reads behind the security-definer
-- source view; do not grant raw MarketOps analytical tables.
GRANT signalops_subscriber_global_eod TO signalops_subscriber_global_eod_runtime;
GRANT SELECT ON subscriber_global_marketops_legacy_parity_source_v3,
  subscriber_global_marketops_legacy_parity_manifest_entries
  TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT ON subscriber_global_marketops_legacy_parity_runs,
  subscriber_global_marketops_legacy_parity_manifest_entries
  TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT ON subscriber_global_marketops_evidence_runs,
  subscriber_global_marketops_evidence_records
  TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_asset_identity_resolutions,
  subscriber_gateway_global_canonical_assets
  TO signalops_subscriber_global_eod;
INSERT INTO schema_migrations (version, applied_at)
VALUES ('000176_subscriber_global_projection_runtime_grants', now())
ON CONFLICT (version) DO NOTHING;
