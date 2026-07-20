# Prospective Options Capture Operations

G142 uses one bounded coverage-runner invocation per market session. Run it after the provider snapshot reflects the completed session. The command stamps the explicit point-in-time capture session while retaining per-contract activity timestamps in provider evidence. It rejects future-dated activity and snapshots with no activity on the requested session before chain persistence.

## Prerequisites

- Migration `000032_marketops_options_capture_sessions` is applied.
- Existing Massive credentials are configured. Do not replace a valid key.
- PostgreSQL and TimescaleDB URLs are configured.
- The canonical same-session Massive equity normalized event is available when the chain snapshot omits underlying spot.
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
  --skip-complete \
  --continue-on-error \
  --max-retries 1
```

Use a larger page budget only after a dry run shows the required strikes and expirations are not present in the bounded sample. Do not interpret a partial capture as an analytics-ready session.

## Readiness Inspection

```text
GET /v1/tenants/tenant-local/marketops/options/captures?symbol=AAPL&limit=30
GET /v1/tenants/tenant-local/marketops/options/captures?analytics_ready=true&session_start=2026-07-01&limit=100
GET /v1/tenants/tenant-local/marketops/options/captures/{capture_id}
```

Review `required_surface_cells`, usable-field counts, quality reasons, provider session, attempts, and error message. A contract-heavy capture can still be partial.

## Scheduling Policy

Schedule one bounded batch command after each completed market session. Do not create one independent scheduler per asset. Keep `max-symbols`, page count, retries, and provider budget explicit in deployment configuration. Weekend sessions are rejected. A weekday snapshot with no contract activity on the requested session is recorded as failed rather than relabeling a prior market-day snapshot.

Run G141 only after the capture API shows at least 20 analytics-ready AAPL sessions in the requested source window.
