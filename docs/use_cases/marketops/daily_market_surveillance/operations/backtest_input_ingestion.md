# MarketOps Back-Test Input Ingestion

G097 documents the bounded operator path for creating data-bearing normalized MarketOps rows after G096 coverage preflight returns empty.

## Required Credential

Set one of these in `.env` or the shell environment before running a live provider pull:

- `SIGNALOPS_MASSIVE_API_KEY`
- `MASSIVE_API_KEY`
The smoke script and credential preflight intentionally ignore generic `API_KEY` so unrelated credentials are not used by accident. They fail before provider access when no explicit key is present.

## Credential Preflight

Run this before starting a live ingestion smoke:

```bash
scripts/marketops_massive_credential_preflight.sh
```

Defaults:

- symbol: `NVDA` unless `MARKETOPS_MASSIVE_PREFLIGHT_SYMBOL` is set
- date: previous UTC day unless `MARKETOPS_MASSIVE_PREFLIGHT_DATE`, `MARKETOPS_INGEST_SMOKE_DATE`, or `SIGNALOPS_MASSIVE_OBSERVATION_DATE` is set
- base URL: `https://api.massive.com` unless `SIGNALOPS_MASSIVE_BASE_URL` is set

HTTP `401` or `403` means the configured key is present but Massive rejected it. No Compose services are started by this preflight.

Duplicate `API_KEY` entries in `.env` can mask a valid generic key if a later blank assignment overwrites it. Prefer copying the intended value into `SIGNALOPS_MASSIVE_API_KEY` or `MASSIVE_API_KEY`; these explicit variables are the only credentials used by the smoke scripts.

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

The script runs the credential preflight first, unless `MARKETOPS_INGEST_SKIP_PREFLIGHT=true` is set. After preflight passes, it starts the existing `normalizer`, `raw-worker`, and `signal-persister` services, then runs the existing Compose `massive-puller` profile. The puller publishes canonical raw events to Redpanda; the normalizer creates `normalized_event_ledger` rows; downstream workers may produce signals if thresholds are crossed.

## Verification

After the smoke completes, wait a few seconds and query:

```bash
GET /v1/marketops/backtest-coverage?tenant_id=tenant-local&dataset=equity_eod_prices
```

A non-empty `coverage` array means G095 campaigns can target the returned dataset/symbol/window. Empty coverage means provider pull, broker publish, or normalization did not produce MarketOps rows and should be inspected before launching campaigns.

## Boundary

This path uses the existing Massive adapter, raw topic, normalizer, normalized ledger, and worker flow. It does not insert synthetic rows directly into Postgres, bypass broker idempotency, deploy policies, mutate detectors, or write graph state.
