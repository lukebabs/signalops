# Syncratic SignalOps --- DSM Initial Build Specification

Version: 0.1

## Purpose

Build the first production-ready SignalOps subsystem for Syncratic
focused on daily Decision Signal Modeling (DSM) using Massive.com market data.

The subsystem must follow Syncratic's **non-interference principle**: -
Operate independently from the existing document ingestion pipeline. -
Produce Event Artifacts consumable by the Syncratic Engine. - Be
replayable, idempotent, observable, and API-first.

------------------------------------------------------------------------

# Initial Scope

Universe:

-   Curated Top 50 US Mega-cap equities.

Cadence:

-   Daily post-market snapshot.
-   Optional next-morning refresh after open-interest settles.

Outputs:

-   Normalized market records
-   Option feature records
-   DSM signals
-   SignalOps Event Artifacts

------------------------------------------------------------------------

# Processing Pipeline

``` text
Scheduler
    ↓
Universe Refresh
    ↓
Underlying Market Snapshot
    ↓
Option Chain Snapshot
    ↓
Normalizer
    ↓
Feature Builder
    ↓
Signal Models Signal Generator
    ↓
SignalOps Event Artifact
    ↓
Syncratic Engine
```

------------------------------------------------------------------------

# Data Layers

1.  Raw Massive API payloads (immutable)
2.  Normalized market snapshots
3.  Normalized option snapshots
4.  Derived option-interest features
5.  DSM signals
6.  Event Artifacts

Never expose raw API payloads directly to DSM.

------------------------------------------------------------------------

# Required Daily Snapshot Fields

Underlying: - Open - High - Low - Close - Previous Close - Daily Return
% - Volume - VWAP

Options: - Strike - Expiration - Call/Put - Bid - Ask - Last - Volume -
Open Interest - Implied Volatility - Delta - Gamma - Theta - Vega -
Underlying Price - Moneyness - Days to Expiration

------------------------------------------------------------------------

# Database Schema

## asset_universe

-   asset_id
-   ticker
-   name
-   asset_type
-   exchange
-   sector
-   industry
-   universe_group
-   is_active

## market_snapshot

-   snapshot_id
-   asset_id
-   snapshot_date
-   snapshot_type
-   open_price
-   high_price
-   low_price
-   close_price
-   previous_close
-   daily_return_pct
-   volume
-   vwap
-   provider
-   source_timestamp

## option_contract

-   contract_id
-   provider_contract_symbol
-   underlying_asset_id
-   option_type
-   strike_price
-   expiration_date
-   contract_size

## option_snapshot

-   option_snapshot_id
-   snapshot_id
-   contract_id
-   bid
-   ask
-   last_price
-   mid_price
-   bid_ask_spread
-   volume
-   open_interest
-   implied_volatility
-   delta
-   gamma
-   theta
-   vega
-   underlying_price
-   moneyness
-   expiry_bucket

## option_interest_feature

Aggregate daily metrics:

-   total_call_open_interest
-   total_put_open_interest
-   call_put_oi_ratio
-   total_call_volume
-   total_put_volume
-   call_put_volume_ratio
-   ATM/ITM/OTM OI
-   Near-term OI
-   Long-term OI
-   IV Skew
-   Weighted IV
-   Max OI Strike
-   Max Pain
-   Gamma Exposure

## option_interest_change

Daily deltas:

-   Call OI Change
-   Put OI Change
-   Net OI Change
-   IV Change
-   Price Change
-   Largest OI Increase
-   Largest OI Decrease

## dsm_signal

-   signal_type
-   signal_direction
-   confidence_score
-   strength_score
-   price_context
-   options_context
-   supporting_metrics
-   supporting_contracts

## signalops_event_artifact

-   artifact_type
-   title
-   summary
-   event_facts
-   graph_candidates
-   dsm_signals
-   lineage

------------------------------------------------------------------------

# Derived Features

Compute:

-   Call/Put OI Ratio
-   Call/Put Volume Ratio
-   OI Change
-   Volume/OI Ratio
-   IV Skew
-   ATM Concentration
-   OTM Concentration
-   ITM Concentration
-   Gamma Concentration
-   Max Pain
-   Price vs OI Divergence

------------------------------------------------------------------------

# Initial DSM Signal Taxonomy

-   Accumulation
-   Hedging Pressure
-   Speculative Call Pressure
-   Volatility Expansion
-   Pinning Risk
-   Divergence
-   Event Anticipation

Each signal includes: - Direction - Confidence - Strength - Supporting
metrics - Supporting contracts

------------------------------------------------------------------------

# Massive API Usage

## Phase 1 (Required)

1.  Ticker Details

-   Initial load
-   Monthly refresh

2.  Previous Day Aggregate

-   Daily
-   Populate market_snapshot

3.  Options Chain Snapshot

-   Daily
-   Populate option_contract and option_snapshot

## Phase 2

-   Historical Aggregate Bars
-   Historical Options Snapshots
-   News

## Phase 3

-   Trades
-   Quotes

## Phase 4

-   WebSocket streaming

------------------------------------------------------------------------

# Operational Principles

-   Idempotent ingestion
-   Retry failed tickers only
-   Preserve raw payloads
-   Version schemas
-   Record provider run IDs
-   UTC timestamps internally
-   Convert market session context to US Eastern where required

------------------------------------------------------------------------

# Build Order

Phase 1

1.  Massive Connector
2.  Universe Refresh
3.  Market Snapshot Job
4.  Options Snapshot Job
5.  Normalizer
6.  Feature Builder
7.  DSM Signal Generator
8.  Event Artifact Generator

Future

-   Streaming
-   Graph clustering
-   Time-series anomaly detection
-   Predictive insight generation
-   Knowledge graph evolution

