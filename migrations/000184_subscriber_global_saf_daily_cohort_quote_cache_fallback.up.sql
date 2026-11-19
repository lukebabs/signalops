-- SAF-3: use the verified current quote cache as a bounded latest-session
-- fallback while the historical EOD evidence stream catches up. This never
-- fabricates a future outcome and remains read-only at the gateway boundary.
CREATE OR REPLACE VIEW subscriber_gateway_global_saf_daily_cohort_validation
WITH (security_barrier = true) AS
WITH raw_eod AS (
  SELECT record.global_asset_id, record.session_date,
    NULLIF(record.payload->>'close', '')::double precision AS close,
    1 AS source_priority
  FROM subscriber_global_marketops_evidence_records record
  JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id = record.evidence_run_id
  WHERE record.evidence_kind = 'eod_bar'
    AND record.algorithm_id = 'marketops.equity_eod.initial_capture'
    AND record.algorithm_version = 'v1'
    AND record.quality_state = 'usable'
    AND run.source_scope IN ('global_provider_capture', 'legacy_materialization')
  UNION ALL
  SELECT asset.global_asset_id,
    (quote.quote_timestamp AT TIME ZONE 'America/New_York')::date,
    quote.price,
    2 AS source_priority
  FROM marketops_asset_quote_cache quote
  JOIN subscriber_global_assets asset ON upper(asset.canonical_symbol) = upper(quote.ticker)
  WHERE quote.stale = false AND quote.price > 0 AND quote.quote_timestamp IS NOT NULL
), eod AS (
  SELECT DISTINCT ON (global_asset_id, session_date) global_asset_id, session_date, close
  FROM raw_eod
  WHERE close IS NOT NULL AND close > 0
  ORDER BY global_asset_id, session_date, source_priority, close DESC
), signals AS (
  SELECT global_asset_id, symbol, session_date,
    lower(COALESCE(result_payload->>'technical_direction', 'neutral')) AS signal_direction
  FROM subscriber_gateway_global_risk_reward_snapshots
)
SELECT signal.global_asset_id, signal.symbol, signal.session_date, signal.signal_direction,
  origin.close AS origin_close, outcome.session_date AS outcome_session_date, outcome.close AS outcome_close,
  CASE WHEN signal.signal_direction IN ('bullish', 'bearish') AND origin.close IS NOT NULL AND outcome.close IS NOT NULL
    THEN (outcome.close - origin.close) / NULLIF(origin.close, 0) END AS forward_return,
  CASE WHEN signal.signal_direction IN ('bullish', 'bearish') AND origin.close IS NOT NULL AND outcome.close IS NOT NULL
    THEN CASE WHEN signal.signal_direction = 'bullish' THEN outcome.close > origin.close ELSE outcome.close < origin.close END END AS directional_hit,
  (signal.signal_direction IN ('bullish', 'bearish')) AS signal_eligible,
  (outcome.close IS NOT NULL) AS outcome_available
FROM signals signal
LEFT JOIN eod origin ON origin.global_asset_id = signal.global_asset_id AND origin.session_date = signal.session_date
LEFT JOIN LATERAL (
  SELECT candidate.session_date, candidate.close FROM eod candidate
  WHERE candidate.global_asset_id = signal.global_asset_id AND candidate.session_date > signal.session_date
  ORDER BY candidate.session_date LIMIT 1
) outcome ON true;

ALTER VIEW subscriber_gateway_global_saf_daily_cohort_validation OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_gateway_global_saf_daily_cohort_validation FROM PUBLIC;
GRANT SELECT ON subscriber_gateway_global_saf_daily_cohort_validation TO signalops_subscriber_gateway;
GRANT SELECT ON subscriber_gateway_global_saf_daily_cohort_validation TO signalops_subscriber_gateway_runtime;
GRANT SELECT ON subscriber_gateway_global_saf_daily_cohort_validation TO signalops_subscriber_global_eod;
