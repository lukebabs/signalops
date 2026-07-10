# Syncratic SignalOps Engineering Specification v1.0

**Status:** Engineering Design Specification\
**Subsystem:** SignalOps\
**Primary Use Case:** Decision Signal Modeling (DSM)

------------------------------------------------------------------------

# 1. Vision

SignalOps is an independently deployable subsystem that continuously
ingests, normalizes, correlates and analyzes structured event streams
while preserving the Syncratic non-interference principle.

The initial production implementation targets daily market intelligence
for a curated universe of approximately fifty U.S. mega-cap equities
using Massive market data.

SignalOps transforms raw market observations into governed Event
Artifacts that can be consumed by the Syncratic Engine for reasoning,
graph enrichment and future insight evolution.

------------------------------------------------------------------------

# 2. Design Principles

-   Independent subsystem
-   API-first
-   Event-driven
-   Replayable
-   Idempotent
-   Observable
-   Horizontally scalable
-   Multi-tenant ready
-   Schema versioned
-   Raw data immutable
-   Derived knowledge reproducible

------------------------------------------------------------------------

# 3. High-Level Architecture

``` text
                Massive REST API
                       │
         ------------------------------
         │            │               │
 Universe Refresh  Market Bars  Option Chain
         │            │               │
         └────────────┴───────────────┘
                    Connector
                        │
                 Raw Payload Store
                        │
                  Normalization Layer
                        │
                 Feature Engineering
                        │
                 DSM Signal Engine
                        │
              Event Artifact Generator
                        │
         ---------------------------------
         │               │               │
     Time-series DB   Knowledge      Syncratic
                      Graph Proposals   Engine
```

------------------------------------------------------------------------

# 4. Service Architecture

Services:

1.  signalops-scheduler
2.  signalops-massive-connector
3.  signalops-normalizer
4.  signalops-feature-engine
5.  signalops-DSM-engine
6.  signalops-artifact-service
7.  signalops-api
8.  signalops-worker
9.  signalops-observability

Each service is independently deployable.

------------------------------------------------------------------------

# 5. Storage

## PostgreSQL / TimescaleDB

Stores:

-   asset universe
-   market snapshots
-   option snapshots
-   derived features
-   DSM signals

Partition daily snapshot tables.

Retention:

Raw payloads: configurable (default 2 years)

Derived analytics: retain indefinitely.

## Object Storage

Persist immutable JSON payloads.

Never overwrite.

------------------------------------------------------------------------

# 6. Event Bus

Recommended: - Redpanda (preferred) - Apache Kafka

Topics:

-   universe.refresh
-   market.snapshot
-   option.snapshot
-   feature.generated
-   DSM.signal
-   artifact.created
-   replay.request

Messages are immutable.

------------------------------------------------------------------------

# 7. Connector Specification

Connector responsibilities:

-   Authentication
-   Rate limiting
-   Retry
-   Pagination
-   Backoff
-   Provider metadata
-   Provider run ID
-   UTC timestamps
-   Response validation

Required endpoints:

-   Ticker Details
-   Previous Day Aggregates
-   Options Chain Snapshot

Future:

-   Historical bars
-   Historical options
-   News
-   Trades
-   Quotes
-   WebSocket

------------------------------------------------------------------------

# 8. Scheduler

Jobs

Universe Refresh - Weekly

Market Snapshot - Daily after market close

Option Snapshot - Daily after market close

Feature Build - Triggered after snapshots complete

DSM - Triggered after feature build

Artifact Generation - Triggered after DSM

Replay - On demand

Jobs must be idempotent.

------------------------------------------------------------------------

# 9. Canonical Schema

Core entities

-   Asset
-   MarketSnapshot
-   OptionContract
-   OptionSnapshot
-   OptionInterestFeature
-   OptionInterestChange
-   DSMSignal
-   EventArtifact

Every entity contains

-   UUID
-   Version
-   Created timestamp
-   Updated timestamp
-   Provider lineage

------------------------------------------------------------------------

# 10. Feature Engineering

Generate

Price

-   Daily return
-   Gap
-   Rolling volatility
-   ATR
-   Moving averages

Options

-   Total Call OI
-   Total Put OI
-   OI ratio
-   Volume ratio
-   ATM concentration
-   OTM concentration
-   ITM concentration
-   IV skew
-   Gamma exposure
-   Max pain
-   Largest OI movement
-   Largest volume movement

Cross-domain

-   Price/OI divergence
-   IV expansion
-   Volume/OI anomalies

------------------------------------------------------------------------

# 11. DSM Engine

Initial taxonomy

-   Accumulation
-   Distribution
-   Hedging Pressure
-   Speculative Calls
-   Speculative Puts
-   Volatility Expansion
-   Volatility Compression
-   Pinning Risk
-   Divergence
-   Event Anticipation

Each signal emits:

-   direction
-   confidence
-   strength
-   supporting metrics
-   supporting contracts
-   explanation

No LLM required for deterministic signal generation.

------------------------------------------------------------------------

# 12. Event Artifact

Each artifact contains

-   summary
-   facts
-   feature values
-   DSM signals
-   graph candidates
-   lineage
-   replay metadata

Artifacts become first-class Syncratic knowledge objects.

------------------------------------------------------------------------

# 13. Graph Proposal Layer

Generate candidate relationships only.

Examples

Ticker → exhibits_signal → Accumulation

Ticker → strongest_interest → Strike

Ticker → volatility_regime → Expansion

The Syncratic Engine decides acceptance.

------------------------------------------------------------------------

# 14. Public API

/api/v1/signalops

Endpoints

GET /assets

GET /snapshots

GET /options

GET /features

GET /signals

GET /artifacts

POST /replay

GET /health

GET /metrics

------------------------------------------------------------------------

# 15. Observability

OpenTelemetry

Prometheus

Grafana

Metrics

-   ingestion latency
-   API failures
-   provider latency
-   queue depth
-   feature duration
-   signal duration
-   artifact generation duration

------------------------------------------------------------------------

# 16. Security

Secrets via Vault/Kubernetes Secret.

Role-based API authorization.

Audit every replay.

Immutable lineage.

------------------------------------------------------------------------

# 17. Deployment

Docker Compose (development)

Kubernetes (production)

Dedicated namespace

Independent Helm chart

Independent scaling

------------------------------------------------------------------------

# 18. Replay

Replay from:

-   raw payloads
-   object storage

Recompute

-   normalization
-   features
-   signals
-   artifacts

Replay never mutates historical raw payloads.

------------------------------------------------------------------------

# 19. Future Roadmap

Phase 2

-   News enrichment
-   Technical indicators
-   Historical backtesting

Phase 3

-   Streaming via WebSocket
-   Near real-time DSM

Phase 4

-   Multi-asset support
-   ETFs
-   Futures
-   FX
-   Crypto

Phase 5

-   Graph clustering
-   Temporal graph analytics
-   Predictive insight generation
-   Autonomous insight proposals

------------------------------------------------------------------------

# 20. Acceptance Criteria

The initial implementation is complete when:

-   Daily ingestion of 50 curated equities succeeds.
-   Market snapshots are normalized.
-   Full options chains are normalized.
-   Derived features are generated.
-   DSM signals are produced deterministically.
-   Event Artifacts are emitted.
-   Replay reproduces identical outputs.
-   Services are independently deployable.
-   Existing Syncratic Engine functionality remains unaffected.

