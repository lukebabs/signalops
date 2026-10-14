# VC/DOSM Roadmap

This roadmap is deliberately sequenced. No item authorizes automated trading, alerting, or signal promotion.

## Now: TTM-only research profile

The current release uses four FMP quarterly rows to derive TTM financials. It publishes an explainable, weekly VC/DOSM research result with data_profile = ttm_only. Three-year CAGR, its score, and the high-valuation/low-growth penalty are withheld.

Acceptance is complete when a pilot persists source statements, TTM derivation, calculation trace, and UI disclosure for a completed session.

## Parallel annual v4 research profile

FMP Starter annual statements provide a separate, five-period annual route to
three-year revenue CAGR and a broader six-dimension fundamental profile. The
annual `vc-dosm-4.0-annual` model is captured centrally and remains parallel
to the live `vc-dosm-3.0` TTM profile until independent replay, calibration,
global-reader, analyst-review, and rollout gates pass. See the
[Subscriber Project annual-enrichment contract](../projects/subscriber_project/fmp_annual_v4_enrichment.md).

## Next: 16-quarter history and CAGR

Dependency: FMP entitlement or a verified provider endpoint that returns sixteen distinct quarterly filings per symbol with filing/accepted timestamps.

Deliverables:

- fetch, deduplicate, and retain sixteen rolling quarters;
- derive revenue three years ago and 3-year revenue CAGR;
- enable growth score and high-valuation/low-growth penalty only with valid history;
- version the financial and valuation profile;
- backtest historical outputs against retained as-of inputs.

Acceptance gate: replay for representative fiscal calendars produces the same result from immutable source rows, with no future filing used.

## Then: analytical completeness

1. Persisted technical snapshots (RSI-14, SMA-50, SMA-200) aligned to completed sessions.
2. Governed peer-group definitions and versioned peer medians.
3. Absolute Intrinsic Value (AIV) inputs and a separate explainable model.
4. Composite ranking with universe/version controls.

Each phase needs independent unit, data-quality, and analyst-review evidence.

## Last: SignalOps integration

Only after validation and governance review may results become candidates for SignalOps signals, alerts, graph relationships, or workflow promotion. The integration must retain research-only provenance, confidence caveats, model version, and evidence links. It must not turn a valuation score into an execution instruction.
