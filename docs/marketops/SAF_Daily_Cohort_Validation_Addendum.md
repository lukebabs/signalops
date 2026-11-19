# SAF Daily Cohort Validation Addendum

## Purpose

Signal Assurance now exposes two intentionally separate measurements:

1. **Daily cohort validation** evaluates the selected cohort's Risk/Reward
   direction against the next available verified EOD close. It reports cohort
   coverage, directional signal coverage, evaluated assets, and daily accuracy.
2. **Matured assertion effectiveness** evaluates only confirmed SAF assertions
   after their declared maturity horizon. It remains the authoritative measure
   of assertion usefulness, materialization, invalidation, and benchmark
   performance.

The two measures must not share a denominator. A daily cohort sample is not a
claim that every asset produced a directional signal, and a matured assertion
sample is not a measure of daily system coverage.

## Read path

`GET /v1/marketops/signal-assurance/daily-cohort-validation` returns the
platform-global projection for the selected watchlist. The projection is
append-only/read-only and uses the canonical Risk/Reward and EOD evidence
projections. It does not create SAF assertions or rewrite historical outcomes.

The UI presents cohort validation before matured assertion progression and
labels the metrics explicitly:

- `cohort_size`: authorized watchlist denominator
- `signal_coverage`: assets with a directional signal snapshot
- `directional_signals`: eligible bullish/bearish signals
- `evaluated`: eligible signals with a next-session close
- `directional_hits`: signals whose next-session move agreed with direction

Neutral signals remain in coverage accounting but are not counted as hits or
misses. This prevents a low-signal day from being misrepresented as algorithm
failure while preserving visibility into breadth.

## Session-close fallback and pending days

When the historical EOD evidence stream has not yet caught up, the projection
may use the verified `marketops_asset_quote_cache` session close and its
`previous_close` as a bounded read-only fallback. The cache freshness flag is
an intraday freshness indicator; it does not invalidate a completed session
close. This allows the latest completed cohort to be evaluated without
fabricating a future result. The newest session remains explicitly pending
until a subsequent close exists, so a latest-session coverage card can show
`132/132` while the latest evaluable day is an earlier session.
