# G129 Options Distribution Backfill

Status: implemented backend/CLI substrate
Timestamp: 2026-07-17T00:00:00Z

## Purpose

G129 turns already-persisted option-chain rows into multiple daily distribution snapshots so the algorithm runner has enough `options_distribution_daily` feature samples to score.

This gate does not make additional Massive provider calls. It reuses the durable G125/G127 chain table.

## Implemented Scope

- Added `signalops-marketops-options-distribution-backfill`.
- Reads persisted `marketops_options_chain_daily` rows for one tenant/symbol.
- Builds one `10_trade_days` distribution snapshot per available trade date using the configured calendar-day lookback.
- Upserts snapshots idempotently into `marketops_options_distribution_daily`.
- Supports `--dry-run` for no-write validation.
- Added a Docker target for the backfill CLI.
- Increased the Postgres options-chain read clamp for the options-chain query path so persisted chain inspection/backfill can scan more than 200 rows without changing global query limits.

## Usage

Dry run:

```sh
signalops-marketops-options-distribution-backfill \
  --tenant-id tenant-local \
  --symbol NVDA \
  --window-days 10 \
  --dry-run
```

Persist distribution snapshots:

```sh
signalops-marketops-options-distribution-backfill \
  --tenant-id tenant-local \
  --symbol NVDA \
  --window-days 10
```

Then materialize normalized feature rows:

```sh
signalops-marketops-options-feature-materializer \
  --tenant-id tenant-local \
  --symbol NVDA \
  --window 10_trade_days \
  --limit 100
```

## Boundaries

- No Massive API calls.
- No scheduler wiring.
- No Top 50 loop.
- No algorithm execution automation.
- No frontend changes.

## Validation

- Unit tests cover per-date distribution writes and dry-run behavior.
- Focused tests passed for the backfill CLI, Postgres storage, API, and MarketOps options package.
- Full Go suite, JSON schema validation, and Docker target build passed.
- Live NVDA backfill scanned 250 persisted chain rows, found 27 trade dates, and upserted 27 distribution snapshots.
- G126 materialization wrote 27 normalized `options_distribution_daily` feature rows with the temporal DSN configured.
- Algorithm runner z-score smoke scanned 27 usable NVDA `call_put_open_interest_ratio` samples and wrote 27 results.

## Follow-On

- Run the CLI against live NVDA storage, materialize features with the temporal DSN, and confirm the algorithm runner scores more than one sample.
- If useful later, add scheduler-controlled distribution backfill after each provider ingestion run.
