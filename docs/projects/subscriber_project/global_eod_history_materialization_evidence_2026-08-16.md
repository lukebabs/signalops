# Global EOD-history materialization evidence — 2026-08-16

## Purpose

Seed the platform-global, append-only EOD evidence ledger from already retained
dedicated temporal events. This is a historical bridge only; it is neither a
provider pull nor a tenant-data serving fallback.

## Controlled execution

| Item | Result |
| --- | --- |
| Agent action | `subscriber-global-eod-history-materialize` |
| Worker | `subscriber-global-eod-history-materializer` |
| Source | Dedicated temporal `normalized_event_ledger`, `tenant-local` / `src-massive` / `equity_eod_prices` |
| Selection policy | Earliest retained processing time per canonical symbol/session; event ID tie-breaker (`initial_capture`) |
| Global writer role | `signalops_subscriber_global_eod` |
| Correlation ID | `subglobaleodhist-20260816T025951Z` |
| Provider calls | None |
| Legacy mutations | None |
| Tenant/list mutations | None |
| Gateway reader or scheduler activation | None |

## Results

| Measure | Value |
| --- | ---: |
| Enabled global warm-EOD assets | 881 |
| Retained source symbols uniquely mapped into that cohort | 121 |
| Immutable `eod_bar` records selected and inserted | 17,421 |
| First session | 2025-06-23 |
| Last session | 2026-08-14 |
| Post-run covered global assets | 121 |

The dry-run and append-only execution returned identical source counts and
date bounds. Post-run verification queried the dedicated primary evidence
ledger by fixed `marketops.equity_eod.initial_capture` / `v1` identity and
returned the same figures.

## Coverage interpretation

The import does not make the entire warm cohort historical-ready. The remaining
760 warm assets had no retained, uniquely mapped source bar in this controlled
seed. They stay coverage-pending until the authoritative central EOD pipeline
captures the approved 50-prior-weekday price baseline and subsequent sessions.
No reader may substitute `tenant-local` data or render absent history as an
analytical result.

## Follow-on gate

Build a restricted, type-specific global EOD-history projection and migrate
each approved reader to it with its own authorization, freshness, parity, and
browser acceptance evidence. The raw evidence ledger is available; it is not
itself a browser API.
