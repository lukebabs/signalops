# Prospective Options Capture Operations

G142 uses one bounded coverage-runner invocation per market session. Run it after canonical equity EOD and provider options data have settled. The session path resolves canonical spot before acquisition, requests only the configured analytical DTE/moneyness slice, aggregates transient candidates, and persists only selected surface evidence. It rejects future-dated or stale-session activity before any options write.

## Prerequisites

- Migration `000032_marketops_options_capture_sessions` is applied.
- Existing Massive credentials are configured. Do not replace a valid key.
- PostgreSQL and TimescaleDB URLs are configured.
- The canonical same-session Massive equity normalized event is available before the options capture starts; it is required for point-in-time moneyness bounds.
- The target session is a weekday in `YYYY-MM-DD` form.

## AAPL Dry Run

```bash
signalops-marketops-options-coverage-runner \
  --tenant-id tenant-local \
  --symbols AAPL \
  --max-symbols 1 \
  --session-date 2026-07-20 \
  --limit 250 \
  --max-pages 2 \
  --max-candidates 500 \
  --min-dte 14 \
  --max-dte 120 \
  --min-moneyness 0.70 \
  --max-moneyness 1.30 \
  --max-retries 1 \
  --dry-run
```

Dry run makes no chain, distribution, feature, or capture-ledger writes.

## Bounded Persist Run

```bash
signalops-marketops-options-coverage-runner \
  --tenant-id tenant-local \
  --symbols AAPL,MSFT,NVDA \
  --max-symbols 3 \
  --session-date 2026-07-20 \
  --limit 250 \
  --max-pages 2 \
  --max-candidates 500 \
  --min-dte 14 \
  --max-dte 120 \
  --min-moneyness 0.70 \
  --max-moneyness 1.30 \
  --skip-complete \
  --continue-on-error \
  --max-retries 1
```

Do not widen acquisition merely to turn a partial result green. Change DTE, moneyness, or candidate bounds only when a registered feature or hypothesis requires the additional evidence. The candidate budget remains a hard cap even when a larger page count is supplied.

## Readiness Inspection

```text
GET /v1/tenants/tenant-local/marketops/options/captures?symbol=AAPL&limit=30
GET /v1/tenants/tenant-local/marketops/options/captures?analytics_ready=true&session_start=2026-07-01&limit=100
GET /v1/tenants/tenant-local/marketops/options/captures/{capture_id}
```

Review `required_surface_cells`, usable-field counts, quality reasons, acquisition bounds, `fetched`, `selected_evidence`, `discarded_candidates`, provider session, attempts, and error message. For the current surface policy, at most seven selected contracts should be persisted.

## Scheduling Policy

Schedule one bounded batch command after each completed market session. Do not create one independent scheduler per asset. Keep `max-symbols`, analytical bounds, candidate budget, page count, retries, and provider budget explicit in deployment configuration. Weekend sessions are rejected. A weekday snapshot with no contract activity on the requested session is recorded as failed rather than relabeling a prior market-day snapshot.

Run G141 only after the capture API shows at least 20 analytics-ready AAPL sessions in the requested source window.
