# S4 Live Canary Execution Evidence — 2026-08-13

Status: execution completed; parity failed closed (0 of 2 matched). No scheduler was enabled.

## Release and authorization

- Live-release commits: `7bfcb5f`, `3a3feb3`, `98fb752`.
- Migration `000103_subscriber_global_eod_canary_live_execution` applied from clean commit `98fb752`.
- One-time authorization: `subeodauth_20260813_aapl_nvda`.
- Authorized worker: `subscriber-global-eod-reconciler`.
- Provider/session: Massive EOD, 2026-08-12.
- Fixed request budget: 2; retry budget: 0.
- Scheduler execution: `false`.
- Frozen request slots: NVDA #1 and AAPL #2.

Two schema syntax errors were detected during deployment before the authorization row, live-run row, or provider call existed. In each case, the partial tables were verified empty, the unrecorded migration attempt was removed, and the corrected clean revision was applied. This was an implementation deployment correction, not a data or provider incident.

## Execution

- Live run: `subeodlive_3fe919b63640ecce8d10ceb2`.
- Correlation: `subscriber-s4-live-aapl-nvda-2026-08-13`.
- Provider requests: 2.
- Provider retries: 0.
- Each request intent was committed before the external call.
- Evidence ledger: two `provider_request_started`, two `provider_response_received`, and two `normalization_completed` events.
- Both global canonical baseline rows have `quality_state=usable` and `algorithm_version=subscriber-global-eod-baseline-v1`.

No browser request, coverage-state mutation, existing MarketOps scheduler change, or future recurring worker was enabled.

## Parity outcome

The provider-free parity reporter compared each global canonical Massive payload with the existing `tenant-local` / `src-massive` normalized EOD record for the same symbol and session. Both comparisons were persisted as `mismatched`; therefore the canary is **not** a passing release.

| Symbol | Shared global value | Tenant-local value | Outcome |
|---|---:|---:|---|
| AAPL VWAP | 302.2955 | 302.2979 | mismatch |
| AAPL volume | 43,907,632 | 42,819,737 | mismatch |
| NVDA VWAP | 223.3434 | 223.3425 | mismatch |
| NVDA volume | 108,536,006 | 108,239,860 | mismatch |

Open, high, low, close, symbol, session, and provider event ID agree for both symbols. The divergence is specifically in Massive’s VWAP/volume values between the newly fetched response and the historical tenant-local normalized ledger. That is a material data-lineage revision signal, not a formatting difference.

## Decision

The global baseline records and full immutable evidence are retained, but no coverage activation or cohort expansion is authorized. Before any retry or expansion, reconcile the Massive revision policy: determine whether the system should preserve initial capture, accept provider corrections, or version both; then recalibrate the tenant-local/global parity contract against that explicit policy.
