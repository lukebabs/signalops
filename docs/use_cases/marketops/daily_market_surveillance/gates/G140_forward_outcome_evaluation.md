# G140 Forward Outcome Evaluation

Status: implemented and live-validated on 2026-07-20.

## Purpose

G140 adds the first immutable forward-outcome ledger for MarketOps research sources. It measures realized behavior after a hypothesis evaluation or opportunity origin without mutating the source evaluation, opportunity, evidence, or signal record.

## Implemented Boundary

- Tenant-aware `marketops_signal_outcomes` ledger with one deterministic row per source, horizon, and calculation version.
- Source types reserved for `hypothesis_evaluation`, `opportunity`, and `signal`.
- The v1 materializer admits eligible, triggered, non-invalidated hypothesis evaluations and persisted opportunities.
- Materialized-signal adaptation is deliberately deferred until a governed hypothesis-to-signal relationship exists; the schema does not infer one from unrelated signal rows.
- Fixed 1, 5, 10, and 20 observed-session horizons.
- Explicit `pending`, `matured`, and `missing_price` statuses.
- Forward return, direction-adjusted favorable/adverse excursion, underlying maximum drawdown, realized-volatility change where enough observations exist, directional hit, threshold hit, and days to threshold.
- Exact origin and forward normalized-event lineage.
- Point-in-time `as_of` cutoff and normalized `equity_eod_prices` input only.
- AAPL-bounded `signalops-marketops-outcome-materializer` CLI with source-date bounds, a 50-session cap, threshold parameter, explicit run ID, and dry-run mode.
- Read-only list/detail APIs.

## Outcome Semantics

The v1 origin price is the persisted normalized EOD close on the source session. A horizon matures after that many subsequent persisted trading-session observations.

A matured row records:

- close-to-close forward total return;
- intraday high/low excursion adjusted to the source direction;
- close-series maximum drawdown;
- annualized realized-volatility change when both backward and forward windows contain enough valid returns;
- direction hit for upside/downside sources;
- threshold hit and first observed session to threshold.

For non-directional sources, directional hit and adverse excursion remain unavailable. Threshold hit means either positive or negative absolute movement reached the configured threshold.

`pending` means the point-in-time cutoff has not reached the horizon. `missing_price` means the horizon should have matured under the weekday session approximation but the required persisted price path is absent or invalid. Missing values remain null and are not converted to zero.

## Immutability And Idempotency

Outcome rows are separate from source records. Their identity is stable across calculation runs and excludes the run ID:

`tenant + source_type + source_id + horizon_sessions + calculation_version`

A later bounded rerun may advance the same row from pending to matured or missing-price and refresh calculation lineage. It does not change the originating evaluation or opportunity. Matured rows cannot regress, and missing-price rows cannot regress to pending.

## Live Result

The bounded AAPL run from 2026-07-01 through 2026-07-20 read:

- 24 hypothesis evaluations;
- 0 opportunities;
- 3 persisted normalized equity EOD events;
- 0 admitted outcome sources;
- 0 outcome rows.

All 24 evaluations were excluded as not triggered. The dry-run and write run produced the same zero-row result, and the persisted outcome ledger remains empty. This is expected: G140 must not create learning records from rejected hypotheses.

Synthetic deterministic tests cover positive matured outcomes, all four horizons, upside/downside/non-directional semantics, threshold timing, pending and missing-price transitions, exact event lineage, and stable rerun identity.

## API

- `GET /v1/marketops/outcomes`
- `GET /v1/marketops/outcomes/{outcome_id}?tenant_id=...`

The list supports source, hypothesis, symbol, direction, status, horizon, source-session range, and limit filters. G140 exposes no browser materialization endpoint.

## Safety Boundary

G140 does not add:

- provider calls or live Massive acquisition;
- scheduling or Top 50 fanout;
- browser-triggered outcome materialization;
- source evaluation, opportunity, evidence, or signal mutation;
- production signal, proposal, alert, insight, artifact, graph, or trade writes;
- hypothesis promotion, policy deployment, or automatic calibration decisions;
- frontend work.

## Next Gate

G141 now provides broader point-in-time equity coverage and strict historical orchestration. Live population remains blocked by historical option analytics and zero triggered sources; calibration and promotion decisions must wait for real matured samples. Materialized-signal outcomes still require an explicit governed hypothesis/proposal/materialization link.

## Links

- Target architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- Architecture evaluation: `../architecture/market_state_intelligence_evaluation.md`
- G138 evaluator: `G138_research_hypothesis_evaluator.md`
- G139 opportunities: `G139_opportunity_layer.md`
- Operations: `../operations/outcome_evaluation.md`
- G141 historical pipeline: `G141_historical_coverage_and_outcome_population.md`
- API contract: `../../../../api.md`
