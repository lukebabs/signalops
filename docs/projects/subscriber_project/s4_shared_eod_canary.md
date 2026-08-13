# Sprint S4 — Shared EOD Canary

Status: prepared-cohort implementation. Provider collection, scheduler cutover, coverage activation, and legacy MarketOps changes remain disabled.

## Purpose

S4 advances the existing S2 shadow plan into a small, immutable canary cohort without granting the cohort authority to fetch market data. The canary establishes exactly which global assets would be dual-run for one completed session and records that selection with its source plan, priority, actor, and correlation provenance.

Migration `000097_subscriber_global_eod_canary` adds platform-owned canary-run and membership records. The records are restricted to the existing `signalops_subscriber_global_eod` worker group. Their schema enforces `provider_execution_enabled=false` and `scheduled_execution_enabled=false`; the preparer cannot enqueue work, call Massive, update global coverage, or change any existing MarketOps scheduler.

## Controlled preparation

Only a pre-existing S2 `shadow` plan can supply a canary. The preparer selects the first 1–50 frozen plan members by persisted priority. It rejects a non-shadow plan, an empty plan, duplicate plan identity/priority, an unsafe cohort size, or an attempt to run without `--execute`.

```bash
SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL='<secret-managed dedicated global-EOD DSN>' \
  go run ./cmd/subscriber-global-eod-canary-preparer --execute \
  --plan-run-id '<shadow-plan-id>' \
  --session-date '<completed-session-date>' \
  --max-symbols 10 \
  --actor subscriber-global-eod-reconciler \
  --correlation-id '<change-or-incident-id>'
```

This command creates only `prepared` records. It is not an approval, provider request, collection run, or schedule change.

## Exit evidence required before execution is designed

A later, separately approved S4 execution slice must retain:

1. named canary and session approval;
2. dedicated global-EOD workload identity preflight;
3. a provider-request budget of one request per unique selected global asset;
4. a dual-run result comparison against the current `tenant-local` path, with an explicit same-session data/provenance contract;
5. per-symbol collection, normalization, algorithm, quality, and parity outcomes;
6. rollback that disables the new worker without deleting the prepared run, catalog, shared evidence, or legacy results; and
7. confirmation that no subscriber browser action or membership change can start the worker.

Until that slice lands and its evidence is approved, S4 remains prepared-only and the current MarketOps EOD scheduler remains authoritative.
