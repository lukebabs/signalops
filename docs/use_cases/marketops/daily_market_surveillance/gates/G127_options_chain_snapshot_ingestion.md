# G127 Options Chain Snapshot Ingestion

Status: implemented backend/CLI substrate
Timestamp: 2026-07-17T00:00:00Z

## Purpose

G127 populates the G125 options-chain substrate with provider-backed option-chain snapshot data for a bounded asset, starting with NVDA.

The gate intentionally ingests the current Massive option-chain snapshot rather than claiming a historical open-interest backfill. Historical option aggregate bars do not by themselves provide the full open-interest and greek fields needed by the MarketOps distribution substrate.

## Implemented Scope

- Added Massive option-chain snapshot client support for `GET /v3/snapshot/options/{underlying}`.
- Added bounded pagination controls through `limit` and `max-pages`.
- Parsed option details, day OHLC/volume/VWAP, open interest, implied volatility, greeks, and underlying price.
- Added conversion from Massive snapshot records into durable `marketops_options_chain_daily` rows.
- Added `signalops-marketops-options-chain-ingestor` CLI.
- The CLI fetches one symbol, upserts chain rows, reads the configured rolling window, and writes a `10_trade_days` options distribution snapshot.
- Added a Docker build target for the ingestor.

## Usage

Dry run:

```sh
signalops-marketops-options-chain-ingestor \
  --tenant-id tenant-local \
  --symbol NVDA \
  --limit 250 \
  --max-pages 1 \
  --dry-run
```

Persist current snapshot rows and distribution:

```sh
signalops-marketops-options-chain-ingestor \
  --tenant-id tenant-local \
  --symbol NVDA \
  --limit 250 \
  --max-pages 1
```

Then materialize algorithm-ready feature rows with G126:

```sh
signalops-marketops-options-feature-materializer \
  --tenant-id tenant-local \
  --symbol NVDA \
  --window 10_trade_days \
  --limit 10
```

## Boundaries

- No Top 50 batch ingestion in this gate.
- No scheduler wiring in this gate.
- No historical multi-day option-chain backfill claim.
- No automatic algorithm execution, signal proposal generation, or frontend changes.
- Provider calls remain explicit operator/CLI actions with bounded page limits.

## Validation

- Massive client tests validate snapshot path, auth query, page limit, pagination, open interest, greeks, and underlying price parsing.
- Conversion tests validate chain-row mapping, moneyness calculation, payload hash, and invalid-record rejection.
- CLI tests validate write behavior, dry-run behavior, and distribution derivation.
- Focused Go tests passed for Massive adapter, MarketOps options package, chain ingestor CLI, and feature materializer CLI.
- Full Go suite, JSON schema validation, and Docker target build passed.
- Authenticated Massive dry-run for NVDA with `limit=5`, `max-pages=1` fetched and converted 5 records with no writes.
- Authenticated NVDA persist run with `limit=250`, `max-pages=1` upserted 250 chain rows and wrote one distribution snapshot.
- G126 materialization with an explicit temporal DSN wrote one `options_distribution_daily` normalized feature row visible to the algorithm runner.
- Algorithm runner smoke scanned 1 usable NVDA options-distribution sample and wrote 0 results because at least 2 samples are required for scoring.

## Follow-On

- Run authenticated provider dry-run/persist smoke for NVDA when provider credentials and budget are available.
- Use G126 to materialize `options_distribution_daily` feature rows from persisted distributions.
- G128 should define the frontend-agent UX for asset-detail options distribution and ingestion status.
