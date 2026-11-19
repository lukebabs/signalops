-- The projection is owned by the migrator role and must execute its nested
-- read projections with owner privileges. Keep these grants internal; the
-- gateway receives only the narrow daily validation view.
GRANT SELECT ON subscriber_gateway_global_risk_reward_snapshots,
  subscriber_global_marketops_evidence_records,
  subscriber_global_marketops_evidence_runs
  TO signalops_subscriber_migrator;
