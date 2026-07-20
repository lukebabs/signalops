# G142 Prospective Options Analytics Capture

Status: implemented backend and operator capture substrate

Date: 2026-07-20

## Purpose

G142 starts the truthful path out of G141's historical options-quality blocker. It records bounded, point-in-time Massive option-chain snapshots prospectively and classifies each symbol/session against the exact analytics-readiness policy consumed by G141.

The gate does not reconstruct unavailable history. It creates an auditable daily capture ledger so sufficient real IV, Greeks, open-interest, and underlying-price coverage can accumulate over future sessions.

## Implemented Scope

- Added `marketops_options_capture_sessions` with deterministic symbol/session identities, terminal quality status, attempt count, provider lineage, capture metrics, and timestamps.
- Extended `signalops-marketops-options-coverage-runner` with explicit `--session-date`, `--skip-complete`, `--continue-on-error`, `--max-retries`, and `--retry-backoff` controls.
- The explicit capture session stamps the point-in-time snapshot; a contract activity timestamp after that session or a snapshot with no activity on that session is rejected before any chain row is written.
- Existing analytics-ready captures can be skipped without a provider call. Partial, no-data, and failed captures remain eligible for a bounded rerun.
- Readiness requires five actual surface cells: 30/60/90-DTE ATM and 30-DTE put/call 25-delta, with usable IV, delta, and underlying price.
- When Massive omits underlying spot, G142 may fill it only from the canonical same-session Massive equity normalized event and records that event id in capture metrics.
- G141 and G142 now share one readiness implementation.
- G141 accepts option-chain rows only when their session has an analytics-ready capture and their ingestion run matches that capture exactly.
- Added tenant-scoped capture list and detail APIs.
- Retained existing chain, distribution, and normalized-feature writes from G133.

## Status Semantics

- `analytics_ready`: all five required surface cells exist.
- `partial`: contracts were captured but the required surface is incomplete.
- `no_data`: the provider returned no usable contracts for the requested session.
- `failed`: provider, date-purity, conversion, or persistence processing failed.

Missing fields remain missing. Contract count is never used as a proxy for surface readiness.

## Bounded Execution

One invocation processes an explicit symbol list or one capped slice of `top50_megacap`. It is not one background job per asset. The existing hard bounds remain in force:

- maximum 50 symbols, default 3;
- maximum 250 records per provider page;
- maximum 20 pages per symbol, default 1;
- maximum 5 retries, default 1;
- bounded persisted-chain and distribution scans.

## Safety Boundary

G142 does not add historical synthesis, automatic always-on Top 50 scheduling, calibration, hypothesis promotion, production signal materialization, frontend mutation, or credential changes. External operators may schedule one bounded command after the market session once provider budget and runtime ownership are approved.

## Acceptance Result

The code path can now accumulate and audit genuine prospective options analytics. G141 remains blocked until at least 20 AAPL sessions are classified `analytics_ready`; G142 makes that dependency measurable rather than weakening it.

## Links

- Operations: `../operations/prospective_options_capture.md`
- API: `../api/options_capture_api.md`
- G141: `G141_historical_coverage_and_outcome_population.md`
- G133: `G133_bounded_top50_options_coverage_expansion.md`


## Live Validation

A bounded AAPL dry run requested the 2026-07-20 capture session and exhausted the provider chain within the 20-page hard limit:

- 3,376 contracts fetched and converted;
- 3,093 contracts with usable IV and complete Greeks;
- 3,376 contracts with open interest;
- zero provider underlying-price values;
- zero of five required surface cells after applying the full readiness policy;
- status `partial` and zero writes.

The canonical AAPL equity EOD pull for 2026-07-20 returned no bar at validation time, so G142 correctly did not enrich the missing spot value or claim an analytics-ready session. Deterministic tests prove that a same-session canonical equity event supplies the missing spot with event lineage and makes an otherwise complete surface ready.

Migration `000032` applied locally, passed isolated-schema up/down validation, and passed PostgreSQL upsert, attempt-increment, list, and detail integration.

A strict G141 rerun admitted zero option rows because no analytics-ready capture exists, retained all 135 equity sessions, returned `blocked_insufficient_coverage`, and wrote nothing.
