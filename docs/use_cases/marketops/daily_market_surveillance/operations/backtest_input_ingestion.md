# MarketOps Back-Test Input Ingestion

G097 documents the bounded operator path for creating data-bearing normalized MarketOps rows after G096 coverage preflight returns empty.

## Required Credential

Set one of these in `.env` or the shell environment before running a live provider pull:

- `SIGNALOPS_MASSIVE_API_KEY`
- `MASSIVE_API_KEY`
The smoke script intentionally ignores generic `API_KEY` so unrelated credentials are not used by accident. It fails before provider access when no explicit key is present.

## Bounded Smoke

Run:

```bash
scripts/marketops_calibration_ingest_smoke.sh
```

Default behavior:

- date: previous UTC day unless `MARKETOPS_INGEST_SMOKE_DATE` or `SIGNALOPS_MASSIVE_OBSERVATION_DATE` is set
- datasets: `equity` unless `MARKETOPS_INGEST_SMOKE_DATASETS` is set
- max companies: `1` unless `MARKETOPS_INGEST_SMOKE_MAX_COMPANIES` is set
- max provider requests: `1` unless `MARKETOPS_INGEST_SMOKE_MAX_PROVIDER_REQUESTS` is set
- max retries: `0` unless `MARKETOPS_INGEST_SMOKE_MAX_RETRIES` is set
- max built events: `1` unless `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_BUILT` is set
- max published events: `1` unless `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_PUBLISHED` is set
- dry-run: forced to `false` for the smoke
- continue-on-error: forced to `false`

Provider HTTP `401` means a credential was present but Massive rejected it. Replace the key before retrying.

The script starts the existing `normalizer`, `raw-worker`, and `signal-persister` services, then runs the existing Compose `massive-puller` profile. The puller publishes canonical raw events to Redpanda; the normalizer creates `normalized_event_ledger` rows; downstream workers may produce signals if thresholds are crossed.

## Verification

After the smoke completes, wait a few seconds and query:

```bash
GET /v1/marketops/backtest-coverage?tenant_id=tenant-local&dataset=equity_eod_prices
```

A non-empty `coverage` array means G095 campaigns can target the returned dataset/symbol/window. Empty coverage means provider pull, broker publish, or normalization did not produce MarketOps rows and should be inspected before launching campaigns.

## Boundary

This path uses the existing Massive adapter, raw topic, normalizer, normalized ledger, and worker flow. It does not insert synthetic rows directly into Postgres, bypass broker idempotency, deploy policies, mutate detectors, or write graph state.
