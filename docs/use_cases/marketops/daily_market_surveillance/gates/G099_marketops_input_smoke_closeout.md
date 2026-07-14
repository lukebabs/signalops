# G099 MarketOps Input Smoke Closeout

Status: implemented and live-validated.

## Purpose

G099 closes the immediate blocker discovered after G098: the provider key did not need replacement, but the local environment mapped the wrong value into `SIGNALOPS_MASSIVE_API_KEY`. After correcting the local mapping, the Massive credential preflight passed and the bounded ingestion smoke exposed a storage upsert bug.

## Implemented Scope

- Fixed `idempotency_records` upsert SQL to match the actual table schema.
- Validated that the existing key value, when mapped into `SIGNALOPS_MASSIVE_API_KEY`, passes Massive preflight.
- Rebuilt the `massive-puller` image so the storage fix is active.
- Ran the bounded ingestion smoke successfully.
- Confirmed G096 coverage returns one normalized NVDA `equity_eod_prices` row for `2026-07-13`.
- Ran a one-run G095 campaign against the data-bearing NVDA window and confirmed `scanned=1`.

## Out Of Scope

- Broad historical ingestion.
- Options coverage expansion.
- Calibration readiness promotion.
- Runtime policy deployment.
- Direct ledger inserts or synthetic data generation.

## Live Validation

- Massive preflight: HTTP `200`.
- Ingestion smoke: one provider request, one event built, one event published, zero failures.
- Coverage: `event_count=1` for `NVDA`, `equity_eod_prices`, `2026-07-13`.
- Campaign: `btcamp-g098-nvda-20260714175001`, status `succeeded`, `scanned=1`, `signals=0`.
