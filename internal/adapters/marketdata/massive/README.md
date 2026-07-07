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


## HTTP Client

The Massive client is configured from environment without logging or committing
secrets.

Supported API key variables, in precedence order:

- `SIGNALOPS_MASSIVE_API_KEY`
- `MASSIVE_API_KEY`
- `API_KEY`

Optional base URL override:

- `SIGNALOPS_MASSIVE_BASE_URL`

The local `.env` file is ignored by git and may contain the API key for manual
validation. Unit tests use fixture-backed `httptest` servers and do not call the
live Massive API.

Implemented client methods:

- `ListOptionContracts(ctx, underlying, asOf, limit)`
- `GetEquityDailyBar(ctx, symbol, date)`
- `GetOptionDailyBar(ctx, optionTicker, underlying, date)`

These methods parse provider responses into the internal record types consumed
by the scheduled event builders. They do not publish broker messages directly.

Option aggregate bars contain price/volume fields for an option ticker, but not
the full contract metadata required by `BuildOptionContractDailyEvent`.
Scheduled option ingestion must pair aggregate bars with the option contract
listing record before building and publishing the canonical raw event.


## Scheduled Pull Runner

The scheduled pull runner combines the megacap seed universe, Massive HTTP
client, event builders, and broker publisher.

Go entrypoint:

```text
cmd/massive-puller
```

Docker image target:

```text
massive-puller
```

Compose profile:

```text
massive-pull
```

Runtime controls:

- `SIGNALOPS_MASSIVE_OBSERVATION_DATE`: optional `YYYY-MM-DD`; defaults to the
  previous UTC day.
- `SIGNALOPS_MASSIVE_DATASETS`: comma-separated `equity,options` selection.
- `SIGNALOPS_MASSIVE_MAX_COMPANIES`: maximum seed companies to process.
- `SIGNALOPS_MASSIVE_OPTIONS_LIMIT`: option contract listing limit per
  underlying.
- `SIGNALOPS_MASSIVE_REQUEST_DELAY`: delay before each provider request, such
  as `250ms` or `1s`.
- `SIGNALOPS_MASSIVE_MAX_RETRIES`: retry attempts for each provider request.
- `SIGNALOPS_MASSIVE_RETRY_BACKOFF`: base retry backoff, multiplied by retry
  attempt.
- `SIGNALOPS_MASSIVE_DRY_RUN`: defaults to `true`; set to `false` to publish
  canonical raw events to `signalops.<env>.raw.v1`.
- `SIGNALOPS_MASSIVE_CONTINUE_ON_ERROR`: continue across symbols after
  provider, build, or publish failures.
- `SIGNALOPS_MASSIVE_TENANT_ID` and `SIGNALOPS_MASSIVE_SOURCE_ID`: envelope
  identity for emitted events.

Dry-run builds events and reports counts without broker publication. Publish
mode writes JSON `RawSignalEvent` values with the idempotency key as the broker
message key and scheduled-pull headers for downstream audit.

Reports include provider request and retry counters so broad scheduled runs can
be audited for API usage and transient provider behavior.
