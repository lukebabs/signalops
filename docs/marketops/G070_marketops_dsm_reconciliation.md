# G070 MarketOps DSM Reconciliation

Timestamp: `2026-07-10T16:48:03Z`

## Decision

The two MarketOps specifications in this directory describe the target DSM architecture. G070 is the first deterministic algorithm gate inside the existing SignalOps platform, not the full MarketOps DSM platform build.

## Adopted In G070

- Daily MarketOps surveillance for normalized Massive equity EOD prices.
- Existing normalized input boundary: raw Massive payloads remain immutable and the detector reads `normalized_payload` from `signalops.<env>.normalized.v1`.
- Existing Redpanda topic convention: `signalops.<env>.*.v1` remains canonical.
- Existing `/v1/*` gateway APIs remain canonical; the draft `/api/v1/signalops/*` path in the target spec is not introduced in this gate.
- Existing Postgres/Timescale ledgers remain canonical until specialized MarketOps asset, snapshot, feature, artifact, or graph tables are introduced.
- App metadata is required for MarketOps detector scope: `app_id=marketops`, `domain=market_data`, and `use_case=daily_market_surveillance`.

## Implemented Gate Scope

G070 adds the Python detector `marketops.dsm.eod_price_v1` for normalized events where:

- `app_id=marketops`
- `domain=market_data`
- `source_adapter=market_data.massive`
- `dataset=equity_eod_prices`
- `use_case=daily_market_surveillance`

The detector computes deterministic v0 features from the current Massive EOD payload:

- open/close move percent
- intraday range percent
- VWAP distance percent when `vwap` is available
- daily return percent when `previous_close` is available
- price-field quality exceptions

It emits one DSM-style `signal.v1` signal per matching normalized event when a threshold is crossed. Price-quality exceptions take precedence over volatility expansion because the current plugin contract emits a single signal per event.

## Deferred Target Architecture

The following remain future gates:

- dedicated asset, market snapshot, option snapshot, feature, artifact, and graph proposal tables;
- option-chain feature engineering;
- independent feature-builder and artifact services;
- object storage for raw/provider payload archives;
- Kubernetes/Helm production packaging;
- graph acceptance workflows;
- broader DSM taxonomy including accumulation, hedging pressure, speculative call/put pressure, pinning risk, and divergence.

## Follow-On Gates

- G071: MarketOps asset universe as first-class storage/API from the Top 50 Mega-cap seed.
- G072: Massive options contract daily normalization.
- G073: MarketOps feature-builder layer for option-interest and price-derived features.
- G074: DSM artifact generation and graph proposal payloads.
- G075: Broader DSM taxonomy pack.
