# S4 As-Of Selection Evidence — 2026-08-13

Status: deployed and verified for the S4 AAPL/NVDA revision pair.

## Deployment

- Commit: `ad56852`.
- Migration: `000105_subscriber_global_eod_revision_selection`.
- Policy version: `s4-as-of-selection-v1`.

The migration created two fixed consumer contexts and the `subscriber_global_eod_resolved_observations` projection. It made no provider call, scheduler change, coverage activation, or mutation of the tenant-local normalized ledger.

## Verified resolution

| Symbol | Context | Chosen role | As-of time | Volume | VWAP |
|---|---|---|---|---:|---:|
| AAPL | `historical_assurance` | initial tenant-local capture | 2026-08-12 22:01:57 UTC | 42,819,737 | 302.2979 |
| AAPL | `current_market_context` | global re-observation | 2026-08-13 12:48:31 UTC | 43,907,632 | 302.2955 |
| NVDA | `historical_assurance` | initial tenant-local capture | 2026-08-12 22:01:57 UTC | 108,239,860 | 223.3425 |
| NVDA | `current_market_context` | global re-observation | 2026-08-13 12:48:31 UTC | 108,536,006 | 223.3434 |

Each result includes selected role, policy version, source event or run, provider payload fingerprint, algorithm version, quality state, provenance, and `as_of_time`. There is no free-form version override.

## Operating result

SAF effectiveness, historical outcomes, and backtests can remain reproducible with the initial capture. Current MarketOps context and future recalculations can consistently use the latest verified provider revision. A later integration must explicitly declare one of these contexts; it cannot silently substitute the other.
