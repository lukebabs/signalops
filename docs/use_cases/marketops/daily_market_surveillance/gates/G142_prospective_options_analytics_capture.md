# G142 Prospective Options Analytics Capture

Status: implemented backend and operator capture substrate

Date: 2026-07-20

## Purpose

G142 starts the truthful path out of G141's historical options-quality blocker. It records bounded, point-in-time Massive option-chain snapshots prospectively and classifies each symbol/session against the exact analytics-readiness policy consumed by G141.

The gate does not reconstruct unavailable history. It creates an auditable daily capture ledger so sufficient real IV, Greeks, open-interest, and underlying-price coverage can accumulate over future sessions.

## Implemented Scope

- Added `marketops_options_capture_sessions` with deterministic symbol/session identities, terminal quality status, attempt count, provider lineage, capture metrics, and timestamps.
- Extended `signalops-marketops-options-coverage-runner` with explicit session, retry/resume, DTE, moneyness, and candidate-budget controls.
- A canonical same-session Massive equity close is now required before any session options request. It defines point-in-time moneyness bounds and its event ID is retained in capture metrics.
- Massive acquisition is provider-filtered to the configured expiration and strike ranges. Defaults are 14-120 DTE, 70%-130% moneyness, and at most 500 transient candidates per symbol/session.
- The bounded candidate set is aggregated in memory into the daily positioning distribution; candidates are not bulk-written to the contract ledger.
- At most seven deterministic contracts are retained as exact evidence for the implemented 30/60/90-DTE ATM and 30/60-DTE put/call 25-delta surface cells.
- The explicit capture session stamps point-in-time evidence; future activity or a snapshot with no activity on that session is rejected before any options write.
- Existing analytics-ready captures can be skipped without a provider call. Partial, no-data, and failed captures remain eligible for a bounded rerun.
- G141 and G142 share one readiness implementation and G141 accepts only evidence rows from an analytics-ready capture's exact ingestion run.
- Added tenant-scoped capture list and detail APIs.

## Status Semantics

- `analytics_ready`: all seven required surface cells exist.
- `partial`: contracts were captured but the required surface is incomplete.
- `no_data`: the provider returned no usable contracts for the requested session.
- `failed`: provider, date-purity, conversion, or persistence processing failed.

Missing fields remain missing. Contract count is never used as a proxy for surface readiness.

## Bounded Execution

One invocation processes an explicit symbol list or one capped slice of `top50_megacap`; it is not one background job per asset. For the prospective session path, bounds are analytical rather than only pagination-based:

- canonical equity evidence is resolved before provider acquisition;
- configurable DTE range, constrained to 7-180 and defaulting to 14-120;
- configurable strike/spot range, defaulting to 70%-130%;
- hard candidate budget of at most 1,000 and default 500 per symbol/session;
- maximum 250 records per provider page and only enough pages to satisfy the lower of the page and candidate budgets;
- up to seven selected surface contracts persisted per symbol/session;
- one compact positioning distribution and normalized feature event built from the transient candidate set.

The capture ledger retains fetched candidate count, selected/discarded counts, acquisition bounds, usable-field counts, and quality results. Contract count is an acquisition metric, not a promise that every candidate was stored.

## Safety Boundary

G142 does not add historical synthesis, automatic always-on Top 50 scheduling, calibration, hypothesis promotion, production signal materialization, frontend mutation, or credential changes. External operators may schedule one bounded command after the market session once provider budget and runtime ownership are approved.

## Acceptance Result

The code path can now accumulate and audit genuine prospective options analytics. G141 remains blocked until at least 20 AAPL sessions are classified `analytics_ready`; G142 makes that dependency measurable rather than weakening it.

## Links

- Operations: `../operations/prospective_options_capture.md`
- API: `../api/options_capture_api.md`
- G141: `G141_historical_coverage_and_outcome_population.md`
- G133: `G133_bounded_top50_options_coverage_expansion.md`

## Validation And Superseded Diagnostic

The initial 2026-07-20 AAPL dry run fetched 3,376 contracts because the first implementation reused the legacy full-chain G133 pagination path. It performed zero writes, but the acquisition behavior did not satisfy the market-state architecture's hypothesis-before-expansion principle and has been superseded.

The corrected prospective path:

- stops before a provider call when canonical same-session equity close is unavailable;
- sends Massive expiration and strike bounds derived from the current surface hypothesis inputs;
- caps transient candidates independently of page count;
- aggregates bounded positioning evidence in memory;
- persists no more than seven deterministic source contracts for the implemented surface cells;
- reports candidate, selected, discarded, and acquisition-bound metrics separately.

Focused tests pass for provider-filter query construction, the no-equity/no-provider guard, deterministic seven-cell selection, compact persistence, and candidate-set distribution aggregation. The canonical AAPL equity EOD pull for 2026-07-20 still had no bar at validation time, so no corrected live provider request or persisted capture was attempted for that session.

Migration `000032` remains applied and previously passed isolated-schema up/down validation plus PostgreSQL upsert, attempt-increment, list, and detail integration. A strict G141 rerun still admits zero options rows until genuine analytics-ready prospective sessions exist.


## G143 Extension

G143 extends the bounded selector from five to seven cells by adding 60-DTE 25-delta put/call evidence. Selected rows now retain bid/ask, quote timestamp, exercise style, shares per contract, provider request ID, selection cell, policy version, and score. The provider and candidate limits above are unchanged, and non-selected candidates remain transient.

See `G143_options_surface_evidence_v1.md` for the generated premium, surface-shape, dimensioned OI-change, quality, and G138 compatibility contract.
