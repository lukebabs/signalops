# Massive Market Data Adapter Seeds

This directory contains the first market-data universe seed for the Massive
(formerly Polygon.io) adapter.

## Top 50 Megacap Seed

Source text:

```text
top50megacap.txt
```

Normalized DB-seed artifact:

```text
top50megacap.normalized.csv
```

Parser:

```text
megacap_seed.go
```

The parser exposes `TopMegacapCompanies()`, returning records with:

- `rank`
- `ticker`
- `ticker_key`
- `company`
- `company_key`
- `sector`
- `sector_key`
- `industry`
- `industry_key`

Normalization rules:

- Tickers are uppercased for display/storage.
- `ticker_key` lowercases tickers and converts exchange/class separators such
  as `.` and `-` to `_`.
- Company, sector, and industry keys are lowercase snake-case strings.
- Lines with `Sector / Industry` are split into primary sector and industry.
- Lines with only a sector leave industry blank.


## Scheduled Event Builder

The first adapter implementation is intentionally network-free. It maps
already-fetched Massive records into canonical `RawSignalEvent` envelopes for
scheduled ingestion.

Implemented datasets:

- `options_contracts_daily`
- `equity_eod_prices`

Builder functions:

- `BuildOptionContractDailyEvent(config, record)`
- `BuildEquityEODPriceEvent(config, record)`

Both builders set:

- `source_domain = market_data`
- `source_adapter = market_data.massive`
- `ingestion_mode = scheduled_pull`
- `schema_id = signalops.raw_signal_event.v1`
- stable `event_id` and `idempotency_key`
- source and entity hints for ticker/option-contract routing

This layer does not call Massive APIs. The later connector layer will fetch
provider payloads and pass normalized provider records into these builders.
