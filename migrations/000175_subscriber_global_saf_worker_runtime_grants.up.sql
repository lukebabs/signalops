-- The Kubernetes global-EOD worker uses a dedicated runtime role. Keep its
-- SAF read path explicit and read-only; gateway access remains unchanged.
GRANT SELECT ON subscriber_gateway_global_signal_assurance_observations,
  subscriber_gateway_global_canonical_assets
  TO signalops_subscriber_global_eod_runtime, signalops_subscriber_global_eod;

GRANT SELECT ON subscriber_global_assets, subscriber_watchlist_memberships TO signalops_subscriber_global_eod_runtime, signalops_subscriber_global_eod;

GRANT SELECT, INSERT ON subscriber_global_saf_benchmark_observations TO signalops_subscriber_global_eod;
GRANT EXECUTE ON FUNCTION subscriber_global_saf_benchmark_observation_immutable_guard() TO signalops_subscriber_global_eod;

GRANT SELECT ON subscriber_global_marketops_evidence_records,
  subscriber_global_marketops_evidence_runs,
  subscriber_global_asset_identity_resolutions,
  subscriber_global_saf_benchmark_observations
  TO signalops_subscriber_global_eod_runtime, signalops_subscriber_global_eod;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000175_subscriber_global_saf_worker_runtime_grants', now())
ON CONFLICT (version) DO NOTHING;
