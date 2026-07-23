# SignalOps Architecture and Data Flow

Status: implemented-platform flow with MarketOps-specific detail
Audience: platform engineers, algorithm engineers, operators, and analysts

## Purpose

SignalOps is a multi-use-case evidence platform. It accepts source data, preserves its immutable origin, normalizes it into stable domain contracts, derives features and signals through deterministic and algorithmic paths, and exposes reviewable outputs to operators. MarketOps Daily Market Surveillance is the first specialized use case; it uses the shared platform without becoming a separate platform.

This document describes what moves through the system, where it becomes durable, and which components are allowed to transform it. It distinguishes research evidence from recommendations and from production actions.

## Core design rules

- **Raw before derived.** External/provider payloads are retained in the raw ledger before a detector, feature process, or algorithm can treat them as input.
- **Normalized is the processing boundary.** Domain processors consume normalized records, not provider-specific payloads, except in explicit ingestion and provider-debug workflows.
- **Derived records retain lineage.** Features, states, algorithm results, signals, artifacts, proposals, and outcomes link back to their source records and versions.
- **Algorithms are platform components.** Python algorithms operate behind versioned platform contracts. They do not own the gateway, bypass ledgers, or directly mutate production signal state.
- **Durable work uses the broker.** Ingestion, normalization, replayable worker processing, and asynchronous algorithm execution use Redpanda/Kafka topics and durable ledgers.
- **Research is not automation.** A MarketOps state, hypothesis, algorithm result, or risk/reward posture is evidence for an analyst. It is not a trade, order, portfolio action, or automatic alert-lifecycle decision.

## Platform data flow

```text
External source / provider / operator input
                 |
                 v
      Gateway + source adapter validation
                 |
       idempotency + raw-event ledger
                 |
                 v
         signalops.<env>.raw.v1 topic
                 |
                 v
             Normalizer worker
                 |
    normalized-event ledger + validation result
                 |
        +--------+---------+------------------+
        |                  |                  |
        v                  v                  v
deterministic domain   feature/state      replay/back-test
processors             materialization    isolated execution
        |                  |                  |
        +--------+---------+------------------+
                 |
                 v
      algorithm execution requests / feature vectors
                 |
                 v
       Python or Go algorithm runtime adapters
                 |
                 v
          immutable algorithm-result ledger
                 |
       +---------+------------+----------------+
       |                      |                |
       v                      v                v
quality/proposal gates   domain evidence   analyst read APIs
       |                      |                |
       v                      v                v
signals -> artifacts -> review surfaces -> bounded context / Ask
```

### 1. Ingress and source validation

The gateway is the public ingestion boundary. Source adapters translate provider-specific responses or incoming events into a raw-event contract with tenant, source, dataset, observation time, payload, entity hints, and idempotency identity. The gateway validates envelope shape and records idempotency before publishing durable work.

Key responsibilities:

- Attribute each record to `tenant_id`, `app_id`, `domain`, `use_case`, and `source_id` where applicable.
- Preserve provider observation time rather than substituting wall-clock time when provenance matters.
- Persist raw-event and idempotency records before downstream consumers rely on the event.
- Publish acknowledged raw events to the broker; workers do not need direct provider access to process them.

The raw ledger is the audit and replay source. It is intentionally provider-shaped and should not be treated as a stable detector or algorithm interface.

### 2. Normalization

The normalizer consumes raw events, validates dataset-specific contracts, and writes normalized records to the normalized-event ledger. A normalized record has a stable schema, canonical entity references, source lineage, processing metadata, and validation outcome.

Normalization separates source variability from domain behavior:

| Concern | Raw event | Normalized event |
| --- | --- | --- |
| Schema | Provider/integration shaped | SignalOps dataset contract |
| Primary purpose | Provenance, replay, source debugging | Detection, features, algorithms, reporting |
| Mutability | Immutable | Immutable |
| Consumer | Normalizer/replay tooling | Domain workers, algorithm runners, APIs |

Records that cannot be normalized do not silently become valid inputs. Their failure state remains observable through worker/run status and source lineage.

### 3. Domain processing and feature extraction

Domain workers consume normalized events and write first-class derived records. Their responsibility is to interpret the normalized domain contract, not to infer missing source data.

Examples of durable outputs include:

- Deterministic signals and evidence.
- Feature definitions and point-in-time feature observations.
- State records and state transitions.
- Options distributions, capture-quality records, and source-coverage records.
- Domain artifacts and reviewable graph/proposal candidates.

Every derived row retains the temporal and source scope needed to reproduce or explain it. Missing, partial, stale, or unusable inputs are represented explicitly rather than converted into zeroes or optimistic defaults.

### 4. Algorithm layer

The algorithm layer is shared SignalOps infrastructure. It supports versioned algorithm definitions, execution requests, runtime metadata, feature vectors, and immutable results. MarketOps is a consumer of this layer, not its owner.

```text
Persisted normalized/domain features
          |
          v
Algorithm execution request
  - algorithm_id and version
  - tenant/domain/use-case scope
  - subject/entity and observation window
  - versioned feature-vector contract
          |
          v
Runtime adapter
  - Python platform algorithm
  - bounded Go algorithm where appropriate
          |
          v
Algorithm result ledger
  - score, confidence, severity
  - feature/result payload
  - execution and source lineage
```

Python is used for reusable statistical and temporal algorithms such as z-score anomaly detection, change-point detection, and temporal risk/reward analysis. Go owns the gateway, broker integration, ledgers, APIs, and orchestration. Algorithms may corroborate or challenge deterministic domain observations, but they do not rewrite source evidence, research hypotheses, or recommendations.

Results can be retained even when their inputs are low quality. A later quality/proposal gate decides whether a result is eligible for a reviewable proposal or other bounded downstream use. This preserves auditability without turning partial data into an operational claim.

### 5. Review, evidence, and operator interfaces

The gateway exposes storage-backed read APIs for operational, domain, algorithm, and review records. The web application uses typed API clients and TanStack Query to render persisted state. Browser reads do not call providers for historical/evidence data.

Operator surfaces may show:

- Source and pipeline health, scheduler runs, provider usage, raw and normalized records.
- Domain states, evidence, signals, artifacts, proposals, and outcomes.
- Algorithm definitions, executions, results, quality, and adjudication.
- MarketOps assets, prices, conditions, hypotheses, options evidence, and risk/reward posture.

Read surfaces remain separate from controlled mutation paths. A review decision, where available, mutates only the governed review record; it does not automatically materialize a signal, write an external graph, or execute an action.

## Reliability, replay, and quality controls

### Durable boundaries

The broker and ledgers divide work into restart-safe stages. Consumers commit source offsets only after the associated durable write succeeds. Scheduler runs, provider usage, idempotency records, replay jobs, and worker health make execution observable.

### Replay and back-testing

Replay uses persisted records and explicit scope, not a hidden re-fetch from an external provider. Back-tests use isolated run identities and storage so historical evaluation cannot change live operational ledgers. Point-in-time cutoffs prevent later knowledge from entering an earlier analysis.

### Fail-closed quality behavior

Quality is part of the data model. Examples include absent source coverage, incomplete normalization, stale quote data, partial options surfaces, zero open interest, denominator-zero ratios, and missing outcome prices. Quality gates block unsupported downstream claims while retaining the underlying audit record.

## MarketOps Daily Market Surveillance

MarketOps applies the shared platform to a centrally managed active asset universe. Its processing is intentionally split into two paths:

1. **Post-close research evidence** — immutable daily ingestion, normalization, options capture, feature/state materialization, algorithm execution, hypotheses, and outcomes.
2. **Intraday analyst display** — delayed/EOD quote cache and 15-minute conditions for the Assets interface. This path is read-only presentation and does not enter the EOD evidence pipeline.

### MarketOps end-to-end flow

```text
Active asset universe (Top 50 + analyst watchlist)
                 |
                 v
Weekday post-close scheduler, 18:01:55 America/New_York
                 |
                 v
Massive equity aggregate puller ------------------------------+
                 |                                            |
         raw event + idempotency                              |
                 v                                            |
          normalizer -> normalized equity EOD barrier         |
                 |                                            |
                 +-- failed/missing assets -> reconciliation -+
                 |
                 v
Options coverage / selected chain capture / distributions
                 |
                 v
MarketOps feature materialization and Market State
                 |
       +---------+---------------------------+
       |                                     |
       v                                     v
deterministic hypothesis evaluation     generic algorithm execution
and opportunity/outcome lineage         (Python platform layer)
       |                                     |
       +----------- evidence + quality -------+
                           |
                           v
              persisted research / review APIs
                           |
                           v
 Assets, Market State, Algorithms, DSM, Backtests, Syncratic views
```

### A. Universe, scheduling, and equity ingestion

`marketops_asset_universe` is the durable asset catalog. The central `marketops_universal_assets` projection combines active Top 50 and analyst-watchlist assets for scheduled pipelines while preserving the source universe group needed for cache and lineage joins.

The governed post-close workflow is scheduled on trading weekdays at 18:01:55 ET. It plans the active universe, acquires bounded equity EOD aggregates, writes canonical raw events, and requires normalized same-session coverage before dependent research stages proceed. A provider or normalization failure does not quietly shrink the intended scope.

Missing or failed equity pulls enter the reconciliation path. Reconciliation tasks are claimed and processed sequentially with bounded provider attempts, explicit status, and normalization confirmation. Analyst-requested historical backfills use a separate sequential worker and retain their own job status, requested/completed sessions, provider counts, and errors.

### B. Options ingestion and quality

For eligible assets, the options pipeline captures bounded selected chain rows and produces distribution snapshots over trade dates, moneyness, and expiration buckets. It records coverage and quality rather than claiming a complete options surface when the provider data is partial.

Important MarketOps quality states include usable, partial, missing, all-zero, partial-zero, and denominator-zero. For ratio evidence, the analyst-facing convention is **put/call**:

- Below `1.0`: calls elevated, bullish positioning context.
- Above `1.0`: puts elevated, bearish positioning context.
- Equal to `1.0`: neutral.

This is descriptive options-flow evidence. It is speculative corroboration, not a technical indicator and not a recommendation.

### C. Feature extraction and Market State

Feature materializers build point-in-time observations from persisted normalized equity and options evidence. The Market State layer stores:

- Versioned feature definitions and observations.
- A state for an asset/session with explicit completeness and quality.
- State transitions across observation windows.
- Evidence references, reason codes, and contribution payloads.
- Deterministic research-hypothesis evaluations over exact feature/state versions.

Examples of features include price/range position, return and gap measures, RSI, realized volatility, volume divergence, moving-average trend/slope, implied-volatility relationships, options distribution ratios, and event context where coverage permits.

Feature completeness is not a cosmetic score. It controls whether an evaluation is eligible and whether a downstream opportunity, calibration metric, or reasoning context may make a claim.

### D. Deterministic research path

The deterministic path evaluates registered MarketOps hypotheses against persisted Market State input. It records both triggered and non-triggered results with evidence, checks, score, required-feature status, and version lineage. Compatible eligible evaluations may be grouped into research-only opportunities. Forward outcomes are materialized at 1, 5, 10, and 20 trading-session horizons with `pending`, `matured`, or `missing_price` state.

Back-tests and calibration use isolated point-in-time inputs. They do not reinterpret future records as historical inputs and do not alter live state, proposal, or signal ledgers.

### E. Platform algorithm corroboration

Generic algorithms consume MarketOps feature vectors through the platform execution contract. Current examples include z-score anomaly and change-point algorithms plus `signalops.algorithms.risk_reward_temporal_v1`.

The temporal risk/reward algorithm prioritizes technical features—52-week range position, RSI, volume divergence, trend/slope, and volatility/range behavior—and treats put/call deviation as separately labelled speculative corroboration. It writes an immutable result with score, confidence, monitored feature payload, and quality context. A post-close runner executes it for active assets; its output is visible as research-only posture in the Assets view.

Algorithm adjudication may compare independent algorithm observations with deterministic hypotheses. Adjudication is evidence for an analyst; it does not change hypothesis definitions, thresholds, recommendations, or signal materialization authority.

### F. Intraday display path

The intraday monitor runs during regular and extended sessions. It obtains entitled delayed aggregates, writes `marketops_asset_quote_cache`, and persists only the latest intraday condition snapshot per asset. The Assets screen reads this cache immediately, avoiding provider calls at page load. Outside supported sessions, the display falls back to the latest completed EOD close and does not request a live refresh.

The quote cache, current price percentage, 52-week range visualization, and intraday condition labels are therefore analyst display data. They must not be used as a substitute for the post-close normalized-evidence barrier or be injected into research outcomes.

### G. Analyst and reasoning interfaces

The MarketOps UI reads persisted APIs for Assets, Market State, options coverage/distributions/chain rows, algorithms, DSM artifacts, back-tests, and intelligence readiness. Asset display names and sector tags can be overridden by an analyst without changing the ticker, provider metadata, asset universe membership, or scheduled pipeline scope.

Syncratic is an explainability boundary. SignalOps constructs bounded context windows from its own persisted MarketOps evidence and can request Ask-based interpretation when explicitly authorized. MarketOps records are not bulk-ingested into Syncratic core; evidence purity, prompt bounds, quality, and governance remain enforced by SignalOps.

## MarketOps operational checkpoints

| Checkpoint | Success condition | Fail-closed behavior |
| --- | --- | --- |
| Universe planning | Every active scheduled asset is included | No silent shrink of planned scope |
| Equity normalization | Same-session normalized EOD barrier passes | Reconciliation tasks retain failed symbols |
| Options capture | Coverage and quality are explicit | Partial/zero surfaces remain visible and can block proposal eligibility |
| Feature/state materialization | Point-in-time inputs and completeness are persisted | Missing inputs remain missing; no fabricated values |
| Algorithm execution | Versioned request/result ledger completes | Failed/low-quality results stay observable but are not promoted automatically |
| Hypothesis/outcomes | Exact lineage and maturity status exist | Pending or missing-price outcomes are not treated as losses or successes |
| Analyst presentation | Reads persisted caches and ledgers | UI does not create provider acquisition or research evidence by viewing data |

## Interfaces and ownership

| Layer | Primary technology | Owns | Must not do |
| --- | --- | --- | --- |
| Gateway/adapters | Go + REST | Source validation, raw persistence, public/query APIs | Embed domain-specific algorithm logic or call Python in process |
| Broker/workers | Redpanda/Kafka + Go/Python | Durable asynchronous delivery, retries, replayable work | Treat an unpersisted result as canonical truth |
| Storage | PostgreSQL/TimescaleDB | Immutable ledgers, domain state, execution/audit records | Infer absent evidence from defaults |
| MarketOps processors | Go workers/CLIs | Domain normalization consumers, options/state/hypothesis/outcome materialization | Bypass normalized inputs for ordinary research processing |
| Algorithm runtime | Python platform packages and bounded adapters | Feature-vector analysis and immutable results | Directly mutate production signals, trades, or portfolio state |
| Web application | React/TanStack Query | Analyst-facing persisted read/mutation controls | Call Massive for evidence data or invent unavailable data |

## Boundaries and non-goals

- SignalOps does not turn MarketOps evidence into trading, orders, portfolio allocation, or investment advice.
- Current research readiness is not an empirical effectiveness claim. Prospective coverage, genuine triggers, and matured outcomes are required before calibration or promotion decisions.
- The generic algorithm layer must remain reusable by non-market domains. MarketOps-specific feature mapping and quality policy belong at the use-case boundary.
- Provider entitlement limits are respected. Quote cache behavior must not be mistaken for real-time market-data authority.

## Related documentation

- [SignalOps Processing Specification](../Syncratic_SignalOps_Processing_Specification.md)
- [MarketOps Functional Components](../use_cases/marketops/daily_market_surveillance/architecture/functional_components.md)
- [Market State Intelligence Evaluation](../use_cases/marketops/daily_market_surveillance/architecture/market_state_intelligence_evaluation.md)
- [MarketOps daily post-close operations](../use_cases/marketops/daily_market_surveillance/operations/daily_postclose_pipeline.md)
- [Deployment and quote-cache behavior](../deployment.md)
