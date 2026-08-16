-- Security-barrier views run with their owner privileges. The projection owner
-- needs source-table SELECT solely to evaluate the views for gateway callers.
-- The runtime signalops role remains restricted to the views granted in 000133.
GRANT SELECT ON sri_segments, sri_etf_registry, sri_segment_snapshots,
  sri_etf_holdings_snapshots, sri_etf_holdings TO signalops_subscriber_migrator;
