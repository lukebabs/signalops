-- The Gateway runtime currently connects as signalops. Grant it read access only
-- to the security-barrier SRI global projections; raw tenant-scoped tables are
-- deliberately not granted by this migration.
GRANT SELECT ON subscriber_gateway_global_sri_segments,
  subscriber_gateway_global_sri_etf_registry,
  subscriber_gateway_global_sri_snapshots,
  subscriber_gateway_global_sri_etf_holdings_snapshots,
  subscriber_gateway_global_sri_etf_holdings TO signalops;
