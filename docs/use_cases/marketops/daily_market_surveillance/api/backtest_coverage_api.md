# MarketOps Back-Test Coverage API

G096 adds a read-only preflight API for checking whether normalized MarketOps events exist before launching historical back-test campaigns.

## Endpoint

- `GET /v1/marketops/backtest-coverage`

## Required Query Parameters

- `tenant_id`

## Optional Query Parameters

- `app_id`, default `marketops`
- `domain`, default `market_data`
- `use_case`, default `daily_market_surveillance`
- `source_id`
- `source_adapter`
- `dataset`
- `symbol`, `symbols`, or `subject_symbol`; values may be repeated or comma-separated
- `window_start` and `window_end` as RFC3339 timestamps
- `limit`, default bounded by the shared API limit helper

## Response

The response groups normalized event rows by tenant/app/domain/use case/source/dataset/symbol and returns:

- `event_count`
- `first_observed`
- `last_observed`

Example:

```json
{
  "coverage": [
    {
      "tenant_id": "tenant-local",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "source_id": "src-massive",
      "source_adapter": "market_data.massive",
      "dataset": "equity_eod_prices",
      "subject_symbol": "AAPL",
      "event_count": 5,
      "first_observed": "2026-07-01T00:00:00Z",
      "last_observed": "2026-07-05T00:00:00Z"
    }
  ]
}
```

## Operational Use

Run this endpoint before creating a G095 campaign. Empty coverage means the campaign would produce zero scanned records and should be preceded by ingestion, normalization, or replay into `normalized_event_ledger` with MarketOps metadata.

This endpoint does not ingest data, mutate back-test rows, deploy policies, or write production ledgers.
