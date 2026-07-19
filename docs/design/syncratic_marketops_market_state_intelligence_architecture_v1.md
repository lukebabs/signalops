# Syncratic MarketOps Market State Intelligence

## Architecture and Implementation Specification

**Document version:** 1.0
**Status:** Implementation specification
**Date:** 2026-07-19
**Target implementer:** Code agent
**Parent subsystem:** Syncratic SignalOps
**Application profile:** MarketOps Daily Market Surveillance

---

## 1. Purpose

This specification defines the next implementation phase for Syncratic MarketOps. The objective is to evolve the existing deterministic market-surveillance implementation into a hypothesis-driven **Market State Intelligence** system.

The implementation MUST preserve the current SignalOps operating principles and existing MarketOps controls:

- immutable raw and normalized event ledgers;
- deterministic and reproducible processing;
- replayability and idempotency;
- bounded provider usage;
- versioned algorithms and detector policies;
- quality-gated proposal generation;
- explicit analyst review before signal materialization;
- operator-controlled knowledge graph mutation;
- bounded Syncratic reasoning context;
- non-interference with Syncratic core document ingestion and Ask workflows.

The purpose of this phase is not to build a comprehensive options-chain warehouse or produce a large number of loosely defined indicators. It is to identify, persist, test, and rank a small number of high-impact market-state transitions that can help an analyst narrow a broad asset universe to the most material opportunities.

The central architectural principle is:

> A production signal represents a statistically unusual, persistent, corroborated, and historically testable change in market state.

---

## 2. Current Implementation Baseline

The following MarketOps capabilities already exist and MUST be reused unless this specification explicitly states otherwise:

1. MarketOps application metadata boundary using:
   - `app_id=marketops`
   - `domain=market_data`
   - `use_case=daily_market_surveillance`
2. Canonical Top 50 mega-cap asset universe.
3. Massive provider adapter.
4. Immutable raw event ledger.
5. Normalized event ledger.
6. Replay and idempotency infrastructure.
7. Deterministic DSM detector pack.
8. Signal, alert, and insight ledgers.
9. DSM artifact persistence.
10. Review-controlled graph proposal workflow.
11. Historical detector backtesting and calibration substrate.
12. Generic algorithm plugin substrate.
13. Immutable algorithm result persistence.
14. Quality-gated algorithm signal proposal workflow.
15. Reviewed-proposal preflight and materialization workflow.
16. Options chain daily storage.
17. Options distribution daily storage.
18. Options feature materializer.
19. Bounded options coverage runner.
20. Options quality states and proposal gating.
21. Syncratic bounded context and Ask integration.
22. Existing MarketOps operator surfaces.

This phase MUST be implemented as an additive extension. It MUST NOT bypass or weaken the existing proposal-review, materialization, graph-review, data-quality, or evidence-purity boundaries.

---

## 3. Problem Statement

A full options chain contains thousands of contracts across strikes, expirations, calls, and puts. Persisting all contracts can preserve provider evidence, but raw chain breadth does not itself create useful intelligence. It can instead introduce:

- excessive data volume;
- highly correlated metrics;
- unstable comparisons as contracts expire or roll;
- false signals caused by illiquid strikes;
- stale or zero-valued provider observations;
- multiple-testing bias;
- analyst overload;
- signals that describe market data without predicting a material outcome.

The desired system must answer questions such as:

- Which implied-volatility region accelerated materially?
- Which maturity experienced the largest IV expansion?
- Which delta bucket attracted unusual open interest?
- Is open interest migrating from one expiration to another?
- Is call-premium expansion occurring across multiple expirations?
- Is the volatility term structure steepening, flattening, or inverting?
- Is activity becoming concentrated around a strike before a known event?
- Is option premium diverging from underlying price movement?
- Does downside hedging increase while the underlying is overbought?
- Has a similar combination historically preceded drawdown, continuation, or volatility expansion?

These questions require a state-transition system, not merely contract storage.

---

## 4. Target Outcomes

The implementation MUST produce the following outcomes.

### 4.1 Daily canonical market state

For each tracked asset and trading session, generate one canonical, versioned market-state snapshot composed of normalized feature domains:

- underlying price and momentum;
- realized volatility;
- implied volatility;
- volatility surface and term structure;
- option premium;
- open interest and volume positioning;
- liquidity and data quality;
- event context;
- optional future macro or alternative-data features.

### 4.2 State-transition observations

For each asset and session, compare the current state with relevant prior states and persist changes including:

- level change;
- percentage change;
- z-score;
- percentile;
- acceleration;
- persistence;
- regime change;
- cross-bucket migration;
- concentration shift;
- divergence;
- corroboration.

### 4.3 Versioned hypothesis registry

Introduce a first-class registry of testable market hypotheses. Each hypothesis MUST define:

- required feature inputs;
- eligible data-quality state;
- evaluation logic;
- expected outcome;
- forecast horizon;
- minimum persistence;
- corroboration requirements;
- backtest configuration;
- calibration status;
- promotion status;
- human-readable interpretation.

### 4.4 Evidence-first signal generation

Feature builders and algorithms MUST emit evidence and transition observations. They MUST NOT directly create production signals.

The hypothesis evaluator will combine eligible evidence into a signal candidate. The existing proposal workflow will then govern review and materialization.

### 4.5 Opportunity ranking

Create an opportunity layer that groups related signal candidates for an asset into a ranked analyst-facing opportunity. The system should present one coherent opportunity with evidence rather than many disconnected alerts.

### 4.6 Closed-loop outcome evaluation

Persist forward outcomes for each materialized signal and opportunity so that historical performance can be evaluated by:

- hypothesis version;
- asset;
- market regime;
- time horizon;
- signal direction;
- confidence band;
- feature combination;
- options liquidity profile.

---

## 5. Non-Goals

The first implementation MUST NOT attempt to:

- ingest all historical option quotes or trades at tick frequency;
- automatically trade or submit orders;
- generate unrestricted natural-language trading recommendations;
- implement every possible options indicator;
- calculate dealer gamma exposure without a separately validated methodology;
- infer institutional identity from open-interest changes;
- treat option volume as confirmed new positioning without open-interest confirmation;
- auto-promote hypotheses based solely on in-sample backtests;
- materialize algorithm outputs directly into the production signal ledger;
- mutate the knowledge graph without review;
- replace existing DSM detectors immediately;
- schedule unbounded Top 50 full-chain collection;
- ingest MarketOps data into Syncratic core.

---

## 6. Design Principles

### 6.1 Hypothesis before data expansion

New data collection MUST be justified by one or more registered hypotheses. The system should not collect high-volume data merely because it is available.

### 6.2 Preserve raw evidence, optimize analytical state

Raw provider snapshots may be retained for replay and forensic use. Operational analysis MUST use normalized features and market states rather than repeatedly scanning the full raw chain.

### 6.3 Measure change, not only level

A static IV value or open-interest value is rarely sufficient. Production logic should prioritize change, acceleration, rarity, persistence, and cross-domain confirmation.

### 6.4 Continuous exposures over unstable contracts

Where historical continuity matters, derive normalized DTE, delta, and moneyness buckets. Exact contract histories remain available for evidence, but hypothesis logic should not depend solely on a contract that will expire.

### 6.5 Asset-relative normalization

Thresholds SHOULD be evaluated against the asset's own trailing history before cross-sectional comparison. AAPL, TSLA, and MSFT do not share identical normal volatility or options-liquidity distributions.

### 6.6 Quality is explicit data

Every derived feature and state MUST carry quality metadata. Missing, stale, zero, sparse, crossed, illiquid, or partial observations MUST never be silently treated as valid zeros.

### 6.7 Explainable composition

Every signal candidate MUST reference the evidence and transitions that caused it. A reviewer must be able to reconstruct the decision without rerunning an opaque model.

### 6.8 Version everything that changes semantics

Feature definitions, bucket rules, state schemas, hypothesis rules, scoring formulas, and outcome definitions MUST be versioned.

### 6.9 Separate research from production

Research results may be persisted and compared, but only calibrated and reviewed hypothesis versions may produce materializable proposals.

### 6.10 Point-in-time correctness

Backtests MUST use only data that would have been available at the evaluation timestamp. Future open-interest revisions, earnings results, adjusted classifications, and later provider corrections MUST not leak into historical evaluation.

---

## 7. Target Architecture

```text
Massive and future providers
        |
        v
Existing raw event ledger
        |
        v
Existing normalized event ledger
        |
        +-------------------------------+
        |                               |
        v                               v
Existing deterministic DSM       Market feature pipeline
        |                               |
        |                               v
        |                       Daily feature observations
        |                               |
        |                               v
        |                         Market State Builder
        |                               |
        |                               v
        |                       State Transition Engine
        |                               |
        |                               v
        |                          Evidence Ledger
        |                               |
        |                               v
        |                        Hypothesis Evaluator
        |                               |
        +-------------------------------+
                                        |
                                        v
                          Existing signal proposal workflow
                                        |
                                        v
                          Review, preflight, materialization
                                        |
                          +-------------+-------------+
                          |                           |
                          v                           v
                   Opportunity Engine          Graph proposals
                          |                           |
                          v                           v
                   Analyst workbench       Review-controlled graph
                          |
                          v
                 Outcome and calibration loop
```

---

## 8. Component Model

### 8.1 Market Feature Pipeline

The Market Feature Pipeline transforms normalized price and option evidence into compact, versioned, algorithm-ready feature observations.

It MUST:

- consume only normalized events or explicitly approved persisted option distributions;
- calculate features at a defined `as_of_time` and `session_date`;
- preserve source event identifiers;
- emit deterministic feature IDs;
- support idempotent replay;
- attach quality metadata;
- avoid emitting a value when required source evidence is unusable;
- distinguish `missing`, `not_applicable`, and numeric zero;
- support feature-domain-specific versions.

Feature domains:

1. `underlying_momentum`
2. `realized_volatility`
3. `implied_volatility`
4. `volatility_surface`
5. `option_premium`
6. `option_positioning`
7. `option_liquidity`
8. `market_event_context`

### 8.2 Market State Builder

The Market State Builder aggregates eligible feature observations into one canonical asset/session state.

It MUST:

- use a declared state schema version;
- enforce one state per asset/session/schema version;
- include feature completeness and quality summaries;
- calculate domain-level quality;
- support partial state creation where permitted;
- indicate which hypotheses are evaluable from that state;
- never infer missing values as zero;
- link all state components to source feature observations.

### 8.3 State Transition Engine

The State Transition Engine compares market states across configured windows.

Supported initial windows:

- 1 trading session;
- 3 trading sessions;
- 5 trading sessions;
- 10 trading sessions;
- 20 trading sessions;
- 30 calendar days where explicitly required;
- 60 calendar days where explicitly required.

It MUST calculate only metrics meaningful for each feature. Supported transition operators include:

- absolute difference;
- percentage difference;
- log return;
- rolling z-score;
- rolling percentile;
- first derivative;
- second derivative or acceleration;
- exponentially weighted change;
- persistence count;
- sign consistency;
- threshold crossing;
- regime transition;
- divergence between two features;
- migration between buckets;
- concentration change;
- curve slope change;
- curve curvature change.

### 8.4 Evidence Ledger

The Evidence Ledger stores individual, reusable claims generated by feature and transition processing.

Examples:

- `AAPL RSI crossed above 70.`
- `AAPL 30-DTE ATM IV increased by 2.4 historical standard deviations over five sessions.`
- `AAPL 25-delta put open interest increased 18% while call open interest remained flat.`
- `AAPL open interest migrated from the nearest monthly expiry to the next monthly expiry.`
- `AAPL 30-to-90-DTE IV term-structure slope inverted.`

Evidence is not a production signal. Multiple evidence records may be combined by a hypothesis.

### 8.5 Hypothesis Registry

The Hypothesis Registry is the canonical definition layer for production-oriented market research.

Each definition MUST include:

- `hypothesis_key`;
- semantic version;
- title;
- domain;
- direction;
- description;
- rationale;
- input feature requirements;
- required transitions;
- quality policy;
- eligibility filter;
- trigger expression;
- persistence rule;
- corroboration rule;
- invalidation rule;
- expected outcome metric;
- forward horizons;
- scoring configuration;
- minimum calibration requirements;
- lifecycle status;
- owner;
- created and approved timestamps.

Lifecycle statuses:

- `draft`
- `research`
- `backtest_ready`
- `calibration`
- `candidate`
- `approved`
- `paused`
- `retired`

Only `approved` hypothesis versions MAY generate proposals eligible for production materialization. `candidate` versions may generate research-only proposals.

### 8.6 Hypothesis Evaluator

The evaluator combines state and transition evidence according to a registered hypothesis version.

It MUST:

- verify point-in-time data eligibility;
- verify feature and state schema compatibility;
- enforce quality policy;
- enforce minimum persistence;
- evaluate corroboration;
- apply invalidation logic;
- produce a deterministic evaluation result;
- persist both triggered and non-triggered evaluation summaries where configured;
- create signal proposals through the existing proposal generator rather than writing signals directly.

### 8.7 Opportunity Engine

The Opportunity Engine groups compatible signal proposals or materialized signals into a higher-level opportunity record.

It MUST:

- group by asset, evaluation session, direction, and compatible horizon;
- avoid double counting correlated hypotheses;
- calculate evidence-domain diversity;
- retain the contribution of each hypothesis;
- assign an opportunity score;
- expose conflicts and invalidating evidence;
- provide a concise analyst explanation;
- maintain lifecycle state separately from individual signals.

Opportunity lifecycle:

- `emerging`
- `active`
- `strengthening`
- `weakening`
- `invalidated`
- `resolved`
- `expired`

The Opportunity Engine MUST NOT submit trades.

### 8.8 Outcome Evaluator

The Outcome Evaluator calculates future observed outcomes for historical signals and opportunities after their forecast horizons mature.

Initial outcomes:

- forward total return;
- maximum favorable excursion;
- maximum adverse excursion;
- realized volatility change;
- maximum drawdown;
- directional hit;
- threshold hit;
- days to threshold;
- IV change where required by the hypothesis;
- premium return for the selected normalized exposure where required.

Outcome rows MUST remain distinct from the original signal or opportunity record to preserve immutability.

---

## 9. Data Acquisition Strategy

### 9.1 General rule

Do not ingest the entire option universe at high frequency by default. Use a tiered strategy.

### 9.2 Daily underlying data

For every active MarketOps asset, collect or derive daily:

- regular-session open;
- high;
- low;
- last trade or close according to the existing MarketOps convention;
- adjusted close where separately required;
- volume;
- VWAP where available;
- prior close;
- corporate-action metadata;
- realized-return series.

### 9.3 Options end-of-day collection

For hypotheses in this specification, end-of-day options collection SHOULD occur after provider data is sufficiently settled. The job MUST store:

- provider observation timestamp;
- collection timestamp;
- session date;
- completeness status;
- pagination status;
- raw record count;
- usable record count;
- quality counts;
- provider request identifiers;
- coverage parameters.

### 9.4 Bounded contract eligibility

The analytical pipeline SHOULD initially process contracts matching configurable bounds:

- DTE: 7 to 730 days for retained analytical evidence;
- primary signal focus: 14 to 180 DTE;
- calls and puts;
- standard contracts only where contract metadata is available;
- positive bid and ask where premium calculations require both;
- non-crossed market;
- acceptable quote age;
- minimum open interest or volume where required;
- configurable maximum bid/ask spread percentage;
- configurable moneyness or delta range.

Raw daily chain evidence may remain broader than the analytical subset.

### 9.5 Initial normalized surface buckets

Create continuous exposure buckets that do not depend on a permanent exact contract.

DTE targets:

- 21
- 30
- 45
- 60
- 90
- 180
- 365

Delta targets:

- put 0.15
- put 0.25
- put 0.35
- ATM or absolute delta nearest 0.50
- call 0.35
- call 0.25
- call 0.15

For each target cell, choose the best eligible contract using a deterministic selection score based on:

- distance from target DTE;
- distance from target absolute delta;
- spread percentage;
- quote freshness;
- open interest;
- volume;
- standard-contract eligibility.

The selected source contract MUST be persisted with the cell observation.

### 9.6 Open-interest limitation

Open interest is not an intraday flow measure. The implementation MUST:

- treat OI as an end-of-day or provider-defined observation;
- calculate OI change only across eligible session observations;
- not label same-day volume as new open interest;
- classify roll or migration as an inference supported by simultaneous decreases and increases across related buckets;
- attach confidence and alternative interpretations.

---

## 10. Canonical Feature Catalog v1

The v1 feature catalog should remain intentionally limited. Every feature MUST have a documented hypothesis use.

### 10.1 Underlying momentum

- `return_1d`
- `return_5d`
- `return_10d`
- `return_20d`
- `rsi_14`
- `rsi_14_percentile_252d`
- `distance_sma_20_pct`
- `distance_sma_50_pct`
- `distance_52w_high_pct`
- `volume_ratio_20d`
- `gap_pct`
- `atr_14_pct`

### 10.2 Realized volatility

- `rv_10d`
- `rv_20d`
- `rv_60d`
- `rv_20d_percentile_252d`
- `rv_acceleration_5d`
- `intraday_range_percentile_252d`

### 10.3 Implied volatility

For each supported DTE and delta cell where applicable:

- `iv`
- `iv_change_1d`
- `iv_change_5d`
- `iv_zscore_60d`
- `iv_percentile_252d`

Asset-level summaries:

- `atm_iv_30d`
- `atm_iv_60d`
- `atm_iv_90d`
- `iv_rank_252d`
- `iv_percentile_252d`
- `iv_minus_rv_20d`
- `iv_rv_ratio_20d`

### 10.4 Volatility surface

- `put_25d_skew_30d`
- `put_25d_skew_60d`
- `call_25d_skew_30d`
- `risk_reversal_25d_30d`
- `risk_reversal_25d_60d`
- `term_slope_30_60`
- `term_slope_30_90`
- `term_slope_60_180`
- `term_curvature_30_60_90`
- `term_structure_state`
- `surface_dispersion`

### 10.5 Option premium

For selected cells:

- `mid_premium`
- `extrinsic_premium`
- `premium_pct_spot`
- `premium_change_1d`
- `premium_change_5d`
- `premium_acceleration_5d`
- `spread_pct`

Asset-level summaries:

- `atm_straddle_pct_spot_30d`
- `atm_straddle_pct_spot_60d`
- `put_call_premium_ratio_30d`
- `put_call_premium_ratio_60d`
- `expected_move_pct_30d`

### 10.6 Positioning

For each expiry, DTE, delta, and moneyness grouping where quality permits:

- `call_oi`
- `put_oi`
- `call_oi_change_1d`
- `put_oi_change_1d`
- `call_oi_change_5d`
- `put_oi_change_5d`
- `call_volume`
- `put_volume`
- `volume_oi_ratio`

Asset-level summaries:

- `put_call_oi_ratio`
- `put_call_volume_ratio`
- `oi_expiry_concentration_hhi`
- `oi_strike_concentration_hhi`
- `top_strike_oi_share`
- `top_expiry_oi_share`
- `oi_migration_near_to_next`
- `oi_migration_short_to_medium_dte`
- `oi_migration_confidence`

### 10.7 Liquidity and quality

- `usable_contract_ratio`
- `stale_quote_ratio`
- `zero_bid_ratio`
- `crossed_market_ratio`
- `missing_iv_ratio`
- `missing_greeks_ratio`
- `median_spread_pct`
- `weighted_spread_pct`
- `oi_quality_state`
- `surface_coverage_ratio`

### 10.8 Event context

Initial event context:

- `days_to_earnings`
- `days_since_earnings`
- `earnings_window_state`
- `days_to_ex_dividend`
- `corporate_action_flag`

Event context MUST be point-in-time correct. The system must distinguish announced event dates from dates learned retrospectively.

---

## 11. Data Model

The following schemas are logical specifications. Exact SQL types may be adapted to existing repository conventions.

### 11.1 `marketops_feature_definitions`

```sql
CREATE TABLE marketops_feature_definitions (
    feature_key             TEXT NOT NULL,
    feature_version         TEXT NOT NULL,
    domain                  TEXT NOT NULL,
    title                   TEXT NOT NULL,
    description             TEXT NOT NULL,
    value_type              TEXT NOT NULL,
    unit                     TEXT,
    calculation_spec        JSONB NOT NULL,
    required_inputs         JSONB NOT NULL,
    quality_policy          JSONB NOT NULL,
    status                   TEXT NOT NULL,
    created_at               TIMESTAMPTZ NOT NULL,
    updated_at               TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (feature_key, feature_version)
);
```

### 11.2 `marketops_feature_observations`

```sql
CREATE TABLE marketops_feature_observations (
    feature_observation_id  UUID PRIMARY KEY,
    app_id                  TEXT NOT NULL,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    session_date            DATE NOT NULL,
    as_of_time              TIMESTAMPTZ NOT NULL,
    feature_key             TEXT NOT NULL,
    feature_version         TEXT NOT NULL,
    dimensions              JSONB NOT NULL DEFAULT '{}'::jsonb,
    numeric_value           DOUBLE PRECISION,
    text_value              TEXT,
    boolean_value           BOOLEAN,
    quality_state           TEXT NOT NULL,
    quality_score           DOUBLE PRECISION,
    quality_details         JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_event_ids        JSONB NOT NULL,
    source_artifact_ids     JSONB NOT NULL DEFAULT '[]'::jsonb,
    calculation_run_id      UUID NOT NULL,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key)
);
```

Recommended dimensions:

```json
{
  "option_type": "put",
  "target_dte": 30,
  "actual_dte": 32,
  "target_delta": 0.25,
  "actual_delta": -0.247,
  "expiry": "2026-08-21",
  "strike": 205.0,
  "source_contract_id": "O:AAPL260821P00205000"
}
```

### 11.3 `marketops_market_states`

```sql
CREATE TABLE marketops_market_states (
    market_state_id         UUID PRIMARY KEY,
    app_id                  TEXT NOT NULL,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    session_date            DATE NOT NULL,
    as_of_time              TIMESTAMPTZ NOT NULL,
    state_schema_version    TEXT NOT NULL,
    state_payload           JSONB NOT NULL,
    feature_count           INTEGER NOT NULL,
    required_feature_count  INTEGER NOT NULL,
    completeness_ratio      DOUBLE PRECISION NOT NULL,
    quality_state           TEXT NOT NULL,
    quality_score           DOUBLE PRECISION,
    quality_summary         JSONB NOT NULL,
    eligible_hypotheses     JSONB NOT NULL DEFAULT '[]'::jsonb,
    build_run_id            UUID NOT NULL,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key)
);
```

The payload SHOULD be partitioned by domain rather than one flat namespace:

```json
{
  "underlying_momentum": {},
  "realized_volatility": {},
  "implied_volatility": {},
  "volatility_surface": {},
  "option_premium": {},
  "option_positioning": {},
  "option_liquidity": {},
  "event_context": {}
}
```

### 11.4 `marketops_state_transitions`

```sql
CREATE TABLE marketops_state_transitions (
    transition_id           UUID PRIMARY KEY,
    app_id                  TEXT NOT NULL,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    session_date            DATE NOT NULL,
    as_of_time              TIMESTAMPTZ NOT NULL,
    current_state_id        UUID NOT NULL,
    baseline_state_id       UUID,
    feature_key             TEXT NOT NULL,
    feature_version         TEXT NOT NULL,
    dimensions              JSONB NOT NULL DEFAULT '{}'::jsonb,
    transition_type         TEXT NOT NULL,
    lookback_sessions       INTEGER,
    current_value           DOUBLE PRECISION,
    baseline_value          DOUBLE PRECISION,
    transition_value        DOUBLE PRECISION,
    zscore                  DOUBLE PRECISION,
    percentile              DOUBLE PRECISION,
    persistence_sessions    INTEGER,
    direction               TEXT,
    quality_state           TEXT NOT NULL,
    transition_payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
    calculation_run_id      UUID NOT NULL,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key)
);
```

### 11.5 `marketops_evidence`

```sql
CREATE TABLE marketops_evidence (
    evidence_id             UUID PRIMARY KEY,
    app_id                  TEXT NOT NULL,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    session_date            DATE NOT NULL,
    as_of_time              TIMESTAMPTZ NOT NULL,
    evidence_type           TEXT NOT NULL,
    evidence_version        TEXT NOT NULL,
    domain                  TEXT NOT NULL,
    direction               TEXT,
    magnitude               DOUBLE PRECISION,
    rarity_score            DOUBLE PRECISION,
    persistence_score       DOUBLE PRECISION,
    quality_score           DOUBLE PRECISION,
    statement               TEXT NOT NULL,
    evidence_payload        JSONB NOT NULL,
    source_feature_ids      JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_transition_ids   JSONB NOT NULL DEFAULT '[]'::jsonb,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key)
);
```

### 11.6 `marketops_hypothesis_definitions`

```sql
CREATE TABLE marketops_hypothesis_definitions (
    hypothesis_key          TEXT NOT NULL,
    hypothesis_version      TEXT NOT NULL,
    title                   TEXT NOT NULL,
    domain                  TEXT NOT NULL,
    direction               TEXT NOT NULL,
    description             TEXT NOT NULL,
    rationale               TEXT NOT NULL,
    required_features       JSONB NOT NULL,
    required_transitions    JSONB NOT NULL,
    quality_policy          JSONB NOT NULL,
    eligibility_expression  JSONB NOT NULL,
    trigger_expression      JSONB NOT NULL,
    persistence_rule        JSONB NOT NULL,
    corroboration_rule      JSONB NOT NULL,
    invalidation_rule       JSONB NOT NULL,
    expected_outcomes       JSONB NOT NULL,
    scoring_config          JSONB NOT NULL,
    calibration_policy      JSONB NOT NULL,
    lifecycle_status        TEXT NOT NULL,
    owner                    TEXT,
    approved_by              TEXT,
    approved_at              TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL,
    updated_at               TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (hypothesis_key, hypothesis_version)
);
```

### 11.7 `marketops_hypothesis_evaluations`

```sql
CREATE TABLE marketops_hypothesis_evaluations (
    evaluation_id           UUID PRIMARY KEY,
    hypothesis_key          TEXT NOT NULL,
    hypothesis_version      TEXT NOT NULL,
    market_state_id         UUID NOT NULL,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    session_date            DATE NOT NULL,
    as_of_time              TIMESTAMPTZ NOT NULL,
    eligible                BOOLEAN NOT NULL,
    triggered               BOOLEAN NOT NULL,
    trigger_score           DOUBLE PRECISION,
    confidence_score        DOUBLE PRECISION,
    magnitude_score         DOUBLE PRECISION,
    rarity_score            DOUBLE PRECISION,
    persistence_score       DOUBLE PRECISION,
    corroboration_score     DOUBLE PRECISION,
    quality_score           DOUBLE PRECISION,
    invalidated             BOOLEAN NOT NULL DEFAULT FALSE,
    evidence_ids            JSONB NOT NULL DEFAULT '[]'::jsonb,
    reason_codes            JSONB NOT NULL DEFAULT '[]'::jsonb,
    evaluation_payload      JSONB NOT NULL,
    evaluation_run_id       UUID NOT NULL,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key)
);
```

### 11.8 `marketops_opportunities`

```sql
CREATE TABLE marketops_opportunities (
    opportunity_id          UUID PRIMARY KEY,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    opened_session_date     DATE NOT NULL,
    last_evaluated_date     DATE NOT NULL,
    direction               TEXT NOT NULL,
    horizon                 TEXT NOT NULL,
    lifecycle_status        TEXT NOT NULL,
    opportunity_score       DOUBLE PRECISION NOT NULL,
    confidence_score        DOUBLE PRECISION NOT NULL,
    domain_diversity_score  DOUBLE PRECISION NOT NULL,
    conflict_score          DOUBLE PRECISION NOT NULL,
    hypothesis_evaluation_ids JSONB NOT NULL,
    signal_ids              JSONB NOT NULL DEFAULT '[]'::jsonb,
    supporting_evidence_ids JSONB NOT NULL,
    invalidating_evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    summary                 TEXT NOT NULL,
    opportunity_payload     JSONB NOT NULL,
    version                 INTEGER NOT NULL,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    updated_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key, version)
);
```

### 11.9 `marketops_signal_outcomes`

```sql
CREATE TABLE marketops_signal_outcomes (
    outcome_id              UUID PRIMARY KEY,
    source_type             TEXT NOT NULL,
    source_id               UUID NOT NULL,
    hypothesis_key          TEXT,
    hypothesis_version      TEXT,
    asset_id                TEXT NOT NULL,
    symbol                  TEXT NOT NULL,
    origin_session_date     DATE NOT NULL,
    horizon_sessions        INTEGER NOT NULL,
    matured_session_date    DATE,
    outcome_status          TEXT NOT NULL,
    forward_return          DOUBLE PRECISION,
    max_favorable_excursion DOUBLE PRECISION,
    max_adverse_excursion   DOUBLE PRECISION,
    maximum_drawdown        DOUBLE PRECISION,
    realized_vol_change     DOUBLE PRECISION,
    directional_hit         BOOLEAN,
    threshold_hit           BOOLEAN,
    days_to_threshold       INTEGER,
    outcome_payload         JSONB NOT NULL,
    calculation_version     TEXT NOT NULL,
    deterministic_key       TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL,
    UNIQUE (deterministic_key)
);
```

---

## 12. Quality Model

### 12.1 Standard quality states

Use a shared enumeration:

- `usable`
- `usable_with_warning`
- `partial`
- `sparse`
- `stale`
- `invalid`
- `missing`
- `not_applicable`

Existing options states such as `partial_zero`, `all_zero`, and `denominator_zero` SHOULD remain in details or be mapped into the shared quality state while preserving the original reason code.

### 12.2 Quality dimensions

Each observation MAY include:

- completeness;
- freshness;
- liquidity;
- provider consistency;
- surface coverage;
- denominator validity;
- source agreement;
- temporal continuity.

### 12.3 Gating behavior

The evaluator MUST reject a hypothesis evaluation when:

- a required feature is absent;
- a required feature is invalid;
- source evidence is stale beyond policy;
- surface coverage is below policy;
- OI quality is unusable for an OI hypothesis;
- bid/ask data is unusable for a premium hypothesis;
- event context is not point-in-time safe;
- the feature version is incompatible with the hypothesis version.

Rejected evaluations SHOULD persist `eligible=false` and reason codes for audit.

---

## 13. First Hypothesis Pack

Implement a small v1 pack. Do not exceed the hypotheses below until the pipeline and backtesting controls are validated.

### H001: Overbought downside-hedging expansion

**Question:** Is downside protection becoming materially more expensive and more heavily positioned while the underlying is overbought?

Required evidence:

- RSI 14 above an asset-relative threshold or absolute 70 threshold;
- 30- to 60-DTE put IV expansion;
- put premium expansion after controlling for intrinsic value;
- positive put OI change in eligible delta buckets;
- acceptable liquidity and surface quality.

Corroboration:

- put IV expansion across at least two maturities; or
- put OI expansion across at least two delta buckets.

Expected outcome:

- elevated probability of 5-, 10-, or 20-session drawdown;
- elevated future realized volatility.

Invalidation examples:

- call-side OI and premium expansion materially stronger than put-side evidence;
- data occurs inside a known earnings window unless evaluated by a separate event-conditioned version.

### H002: Bullish premium expansion across maturities

**Question:** Are call premiums and IV accelerating across multiple expirations while the underlying trend remains constructive?

Required evidence:

- positive underlying trend;
- call premium acceleration across at least two DTE targets;
- call IV expansion;
- usable spread and quote quality.

Expected outcome:

- positive forward return or positive breakout probability;
- increased realized volatility.

### H003: Open-interest expiry migration

**Question:** Is positioning rolling from a near expiration into a later expiration rather than being closed outright?

Required evidence:

- material OI decrease in one expiry bucket;
- material OI increase in an adjacent later expiry bucket;
- compatible option type, delta, or moneyness region;
- migration magnitude above asset-relative rarity threshold;
- persistence or confirmation across at least two sessions where practical.

Output MUST use language such as `positioning appears to be rolling` rather than claiming verified institutional action.

Expected outcome:

- persistence of directional or hedging exposure beyond the near expiry.

### H004: Volatility term-structure regime shift

**Question:** Has the IV curve materially steepened, flattened, or inverted?

Required evidence:

- valid ATM IV at 30, 60, and 90 DTE;
- slope transition rarity;
- persistence beyond one isolated bad quote;
- sufficient surface coverage.

Expected outcome:

- future volatility regime change;
- event-risk repricing;
- mean reversion or continuation depending on transition class.

### H005: Strike concentration before event

**Question:** Is open interest or premium becoming unusually concentrated near a strike before a known event?

Required evidence:

- event date known at evaluation time;
- rising strike-concentration HHI or top-strike share;
- material concentration change;
- adequate contract liquidity.

Expected outcome:

- increased pinning, breakout, or volatility risk around the event and expiration window.

The first version SHOULD classify the observation rather than assert a directional forecast unless separately calibrated.

### H006: Premium-price divergence

**Question:** Is option-market repricing diverging from the underlying move?

Initial classes:

1. price up, call premium/IV weakening;
2. price up, put premium/IV strengthening;
3. price down, put premium/IV weakening;
4. price down, call premium/IV strengthening.

Required evidence:

- underlying move above asset-relative threshold;
- premium or IV transition above rarity threshold;
- delta-aware or bucket-normalized exposure;
- quality-valid bid/ask evidence.

Expected outcome:

- continuation or reversal depending on divergence class and historical calibration.

### H007: Delta-bucket unusual OI accumulation

**Question:** Which normalized delta region attracted statistically unusual new open interest?

Required evidence:

- usable OI for current and prior eligible session;
- OI increase beyond rolling percentile or z-score threshold;
- minimum absolute OI threshold;
- acceptable bucket continuity and contract-selection confidence.

Expected outcome:

- candidate directional, hedging, or event positioning requiring cross-domain confirmation.

This hypothesis SHOULD NOT become a high-confidence opportunity without corroboration from premium, IV, or price behavior.

---

## 14. Signal and Opportunity Scoring

### 14.1 Signal evaluation score

The v1 score SHOULD be composed from normalized components:

```text
trigger_score =
    magnitude_weight       * magnitude_score
  + rarity_weight          * rarity_score
  + persistence_weight     * persistence_score
  + corroboration_weight   * corroboration_score
  + quality_weight         * quality_score
  - conflict_weight        * conflict_score
```

Default weights MUST be configurable by hypothesis version. Do not hard-code a single global weighting formula.

### 14.2 Confidence semantics

Confidence MUST represent confidence that the defined market-state transition exists and satisfies the hypothesis, not certainty of future profit.

Historical success probability MUST be presented separately as calibration metadata.

### 14.3 Opportunity score

The Opportunity Engine SHOULD consider:

- highest contributing signal score;
- number of independent corroborating hypotheses;
- domain diversity;
- historical calibrated performance;
- signal persistence;
- evidence quality;
- conflicts;
- overlap penalty for highly correlated hypotheses;
- event proximity;
- liquidity suitability.

### 14.4 Correlation control

Do not add scores linearly for hypotheses built from essentially identical evidence. Define hypothesis families and overlap groups, for example:

- volatility expansion family;
- positioning family;
- momentum exhaustion family;
- divergence family;
- event concentration family.

Only the strongest member of a highly overlapping group should receive full contribution.

---

## 15. Backtesting and Calibration

### 15.1 Reuse existing substrate

Extend the existing `/v1/marketops/backtests` infrastructure rather than creating an unrelated test runner.

### 15.2 Required backtest modes

1. Single hypothesis/version over one or more assets.
2. Comparison between hypothesis versions.
3. Walk-forward calibration.
4. Cross-sectional daily ranking.
5. Event-conditioned evaluation.
6. Regime-segmented evaluation.
7. Ablation test removing one evidence domain.

### 15.3 Required horizons

Initial forward horizons:

- 1 session;
- 3 sessions;
- 5 sessions;
- 10 sessions;
- 20 sessions;
- 60 sessions where sample size permits.

### 15.4 Required metrics

- sample count;
- eligible-state count;
- trigger count;
- trigger rate;
- directional hit rate;
- mean and median forward return;
- mean and median maximum favorable excursion;
- mean and median maximum adverse excursion;
- drawdown incidence;
- realized-volatility change;
- precision by confidence band;
- calibration error;
- performance by asset;
- performance by year;
- performance by volatility regime;
- performance around earnings versus outside earnings;
- false-positive and missed-opportunity review labels.

### 15.5 Statistical safeguards

The code agent MUST include support for:

- minimum sample size;
- train/test or walk-forward segmentation;
- no look-ahead data access;
- multiple-hypothesis tracking;
- confidence intervals or bootstrap intervals;
- baseline comparison;
- transaction-cost-aware metrics only where a concrete strategy is later defined;
- survivorship-bias documentation for the selected universe;
- versioned asset-universe membership.

### 15.6 Promotion criteria

A hypothesis version MUST NOT reach `approved` merely because aggregate return is positive. Minimum readiness SHOULD include:

- adequate sample size;
- stable out-of-sample or walk-forward behavior;
- no material data-quality dependency;
- understandable failure modes;
- acceptable calibration;
- documented regime sensitivity;
- no severe concentration in one asset or short period;
- analyst review labels;
- reproducible run artifacts.

Promotion remains operator-controlled.

---

## 16. Knowledge Graph Integration

### 16.1 What should enter the graph

The graph SHOULD represent durable analytical concepts rather than every option contract row.

Candidate node types:

- `Asset`
- `MarketState`
- `StateTransition`
- `Evidence`
- `Hypothesis`
- `Signal`
- `Opportunity`
- `MarketEvent`
- `Outcome`
- `Regime`

Candidate relationships:

- `HAS_STATE`
- `TRANSITIONED_TO`
- `SUPPORTED_BY`
- `EVALUATES`
- `TRIGGERED`
- `CONTRIBUTES_TO`
- `CONFLICTS_WITH`
- `PRECEDED`
- `RESULTED_IN`
- `OCCURRED_BEFORE`
- `SIMILAR_TO`

### 16.2 Graph proposal boundary

Graph records MUST continue through the existing proposal lifecycle. No Market State component may directly mutate the production graph.

### 16.3 Contract references

Exact option contracts may be referenced inside evidence payloads and artifact lineage. They SHOULD NOT become graph nodes by default unless a later use case demonstrates durable analytical value.

---

## 17. API Specification

Follow existing MarketOps route conventions.

### 17.1 Features

- `GET /v1/marketops/features/definitions`
- `GET /v1/marketops/features/observations`
- `POST /v1/marketops/features/materialize`
- `GET /v1/marketops/features/coverage`

Filters SHOULD include:

- symbol;
- asset ID;
- session-date range;
- feature key;
- domain;
- quality state;
- feature version;
- dimensions.

### 17.2 Market states

- `GET /v1/marketops/states`
- `GET /v1/marketops/states/{market_state_id}`
- `POST /v1/marketops/states/build`
- `GET /v1/marketops/states/{market_state_id}/lineage`

### 17.3 Transitions and evidence

- `GET /v1/marketops/transitions`
- `POST /v1/marketops/transitions/calculate`
- `GET /v1/marketops/evidence`
- `GET /v1/marketops/evidence/{evidence_id}`

### 17.4 Hypotheses

- `GET /v1/marketops/hypotheses`
- `POST /v1/marketops/hypotheses`
- `GET /v1/marketops/hypotheses/{key}/{version}`
- `POST /v1/marketops/hypotheses/{key}/{version}/evaluate`
- `POST /v1/marketops/hypotheses/{key}/{version}/status`
- `GET /v1/marketops/hypothesis-evaluations`

Mutation endpoints MUST follow existing authentication, audit, and operator-control patterns.

### 17.5 Opportunities

- `GET /v1/marketops/opportunities`
- `GET /v1/marketops/opportunities/{opportunity_id}`
- `POST /v1/marketops/opportunities/build`
- `POST /v1/marketops/opportunities/{opportunity_id}/review`

### 17.6 Outcomes

- `POST /v1/marketops/outcomes/materialize`
- `GET /v1/marketops/outcomes`
- `GET /v1/marketops/calibration/hypotheses/{key}/{version}`

### 17.7 Job requests

All materialization endpoints SHOULD accept:

- explicit symbols;
- asset universe reference;
- maximum symbols;
- session-date range;
- dry-run mode;
- force-recompute flag restricted to privileged workflows;
- feature or hypothesis version;
- correlation ID.

Broad provider fanout MUST remain bounded.

---

## 18. Worker and Job Topology

Introduce the following logical workers. They may initially run as commands in existing worker images if that matches repository conventions.

1. `marketops-feature-materializer`
2. `marketops-state-builder`
3. `marketops-transition-calculator`
4. `marketops-evidence-generator`
5. `marketops-hypothesis-evaluator`
6. `marketops-opportunity-builder`
7. `marketops-outcome-materializer`

Each worker MUST support:

- deterministic idempotency key;
- explicit date and symbol bounds;
- dry run;
- structured run summary;
- durable run status where existing infrastructure supports it;
- partial-failure reporting;
- retry without duplicate rows;
- metrics and logs;
- version reporting;
- no hidden scheduler creation.

Suggested daily sequence:

```text
1. Confirm normalized equity and options evidence availability.
2. Materialize feature observations.
3. Build market states.
4. Calculate transitions.
5. Generate reusable evidence.
6. Evaluate approved and research hypotheses.
7. Generate signal proposals through existing workflow.
8. Build or update opportunities.
9. Build bounded Syncratic context where requested.
10. Materialize matured historical outcomes.
```

---

## 19. Frontend Requirements

Extend the existing MarketOps application rather than creating a disconnected UI.

### 19.1 Market State view

For an asset and date, show:

- underlying state summary;
- volatility state;
- positioning state;
- premium state;
- liquidity and quality;
- event context;
- comparison with prior session;
- feature lineage.

### 19.2 Surface view

Display a DTE-by-delta matrix for:

- IV;
- IV daily change;
- IV z-score;
- premium change;
- OI change;
- spread or quality.

The UI should support questions such as:

- which cells accelerated;
- which maturity expanded most;
- which delta bucket received unusual OI;
- which cells are unusable and why.

### 19.3 Transition timeline

Show material transitions over time, grouped by domain. Avoid rendering every feature change as an alert.

### 19.4 Hypothesis workbench

Show:

- hypothesis definition and version;
- required evidence;
- current evaluation;
- reason codes;
- trigger contribution;
- backtest metrics;
- calibration state;
- review and promotion status.

### 19.5 Opportunity queue

Rank active opportunities and show:

- score;
- direction;
- horizon;
- contributing hypotheses;
- supporting evidence;
- conflicts;
- state change summary;
- historical calibration;
- data-quality warnings.

### 19.6 Separation of controls

The UI MUST keep these actions distinct:

- provider acquisition;
- feature materialization;
- state construction;
- research evaluation;
- proposal review;
- signal materialization;
- graph proposal review;
- opportunity analyst disposition.

---

## 20. Syncratic Ask Integration

The existing bounded reasoning boundary remains in force.

### 20.1 Context bundle

A Market State Ask context SHOULD include only the evidence needed for the question:

- asset and timestamp;
- current market-state summary;
- material transitions;
- triggered hypotheses;
- supporting and conflicting evidence;
- quality warnings;
- historical calibration summary;
- source lineage identifiers.

### 20.2 Prompting constraints

Ask responses MUST distinguish:

- observed fact;
- calculated feature;
- statistical rarity;
- hypothesis inference;
- historical association;
- unknown future outcome.

The reasoning layer MUST NOT state that an options observation guarantees the next price move.

### 20.3 Evidence purity

Do not include evidence from a different asset, session, hypothesis version, or incompatible state schema. Existing evidence-purity checks MUST be extended for new state and transition records.

---

## 21. Observability

Emit metrics at each stage.

### 21.1 Acquisition

- symbols requested;
- symbols completed;
- provider calls;
- pages fetched;
- rows fetched;
- incomplete chains;
- provider errors;
- rate-limit events.

### 21.2 Features

- features attempted;
- features emitted;
- features missing;
- quality-state counts;
- surface coverage;
- calculation latency;
- idempotent duplicate count.

### 21.3 States

- states built;
- average completeness;
- domain quality counts;
- hypotheses evaluable per state;
- incompatible feature versions.

### 21.4 Evaluations

- hypothesis evaluations;
- eligible evaluations;
- triggers;
- blocked evaluations by reason;
- proposal count;
- proposal rejection count;
- confidence distribution.

### 21.5 Opportunities

- active opportunities;
- opportunities by status;
- average supporting domains;
- conflict rates;
- analyst dispositions.

### 21.6 Outcomes

- outcomes matured;
- outcomes pending;
- performance by hypothesis and confidence band;
- calibration drift;
- missing outcome prices.

Structured logs MUST include correlation IDs, run IDs, symbol, session date, and component version.

---

## 22. Security and Governance

- Reuse existing authentication and authorization.
- Hypothesis status promotion requires an audited operator action.
- Feature-definition changes require a new version when semantics change.
- Approved hypothesis definitions MUST be immutable; changes create a new version.
- Materialization and graph mutation remain separately authorized.
- Store provider credentials only through existing secret management.
- Do not expose raw provider payloads in frontend responses unless explicitly authorized.
- Preserve all review and materialization audit records.
- Track model or library version for algorithm-derived evidence.

---

## 23. Testing Requirements

### 23.1 Unit tests

Cover:

- feature formulas;
- DTE and delta bucket selection;
- intrinsic and extrinsic premium calculations;
- IV curve slope and curvature;
- OI migration inference;
- concentration metrics;
- transition z-score and percentile;
- hypothesis expression evaluation;
- score composition;
- quality gating;
- deterministic ID generation.

### 23.2 Integration tests

Cover:

- normalized events to feature observations;
- persisted options distribution to feature materialization;
- feature observations to market state;
- state to transitions and evidence;
- evidence to hypothesis evaluation;
- triggered evaluation to existing proposal generator;
- reviewed proposal to existing materialization path;
- signal to opportunity;
- outcome materialization;
- graph proposal generation without direct mutation.

### 23.3 Replay tests

Re-running the same symbol/date/version MUST:

- produce no duplicate rows;
- produce identical deterministic payloads unless source data changed under an explicit correction workflow;
- retain auditable run summaries.

### 23.4 Data-quality tests

Use fixtures for:

- zero OI;
- missing OI;
- crossed bid/ask;
- zero bid;
- stale quote;
- missing Greeks;
- sparse expirations;
- missing target DTE;
- no eligible delta contract;
- earnings-date uncertainty;
- adjusted/non-standard contract.

### 23.5 Backtest correctness tests

Verify:

- no future state access;
- asset-universe membership by historical date;
- outcome maturity calculation;
- trading-session window logic;
- event-date point-in-time behavior;
- hypothesis-version isolation;
- train/test segmentation.

---

## 24. Implementation Phases

### Phase 1: Foundation and schemas

Deliver:

- database migrations;
- feature registry;
- feature observation model;
- market-state model;
- transition model;
- evidence model;
- hypothesis registry and evaluation model;
- API read endpoints;
- shared quality enumeration;
- deterministic key utilities.

Acceptance criteria:

- migrations apply cleanly;
- schemas preserve MarketOps metadata boundary;
- all new rows are versioned and idempotent;
- no existing tests regress.

### Phase 2: AAPL vertical slice

Implement one complete daily AAPL path:

- underlying momentum features;
- 30-, 60-, and 90-DTE ATM IV;
- 25-delta put/call IV;
- selected premium features;
- put/call OI and OI change;
- surface and liquidity quality;
- market state;
- transitions;
- evidence.

Implement H001, H004, H006, and H007 in research status.

Acceptance criteria:

- at least 60 eligible historical sessions can be replayed where source data exists;
- every state has lineage;
- unusable OI or quotes block relevant evidence;
- repeated runs are idempotent;
- hypothesis evaluations explain trigger and non-trigger reason codes.

### Phase 3: Backtest integration

Deliver:

- hypothesis backtest adapter;
- forward outcome materializer;
- comparison and walk-forward modes;
- metrics endpoints;
- calibration reports;
- baseline support.

Acceptance criteria:

- no look-ahead leakage in tests;
- hypothesis versions compare independently;
- results segment by earnings window and volatility regime;
- reports show sample-size warnings.

### Phase 4: Proposal and opportunity integration

Deliver:

- hypothesis evaluation to existing proposal generator;
- proposal quality policy integration;
- opportunity grouping and scoring;
- opportunity API;
- operator queue.

Acceptance criteria:

- no direct production-signal write exists;
- only eligible lifecycle states generate proposals;
- correlation overlap penalty prevents double counting;
- conflicts remain visible.

### Phase 5: Knowledge graph and Ask

Deliver:

- graph proposal mappings for state, transition, hypothesis, signal, opportunity, and outcome;
- bounded Ask context;
- explanation templates;
- evidence purity extensions.

Acceptance criteria:

- graph mutation remains review-controlled;
- Ask output clearly separates observation from inference;
- context is bounded by asset, date, and evidence lineage.

### Phase 6: Top 50 bounded rollout

Deliver:

- capped batch execution;
- per-symbol quality summary;
- provider-budget controls;
- operational dashboards;
- staged hypothesis approval.

Acceptance criteria:

- no unbounded fanout;
- partial symbol failures do not invalidate successful symbols;
- quality and coverage are visible before proposal generation;
- rollout can be limited to explicit symbol cohorts.

---

## 25. Code Organization

Adapt names to the repository, but preserve separation of responsibilities.

```text
signalops/
  marketops/
    features/
      definitions/
      calculators/
      registry.py
      materializer.py
      quality.py
    states/
      schemas/
      builder.py
      lineage.py
    transitions/
      calculators/
      engine.py
    evidence/
      generators/
      ledger.py
    hypotheses/
      definitions/
      registry.py
      evaluator.py
      expressions.py
      scoring.py
    opportunities/
      builder.py
      scoring.py
      lifecycle.py
    outcomes/
      materializer.py
      metrics.py
    backtests/
      hypothesis_adapter.py
      walk_forward.py
      reports.py
    api/
      features.py
      states.py
      transitions.py
      evidence.py
      hypotheses.py
      opportunities.py
      outcomes.py
    workers/
      feature_materializer.py
      state_builder.py
      transition_calculator.py
      hypothesis_evaluator.py
      opportunity_builder.py
      outcome_materializer.py
```

Definitions SHOULD be declarative where practical, but arbitrary unsafe code execution from stored JSON MUST NOT be permitted. Use a constrained expression schema or validated internal DSL.

---

## 26. Example Daily AAPL Flow

```text
Session: 2026-07-17
Asset: AAPL

1. Underlying EOD normalized event is present.
2. Bounded options snapshot is complete.
3. Feature pipeline selects eligible 30/60/90-DTE surface cells.
4. It calculates:
   - RSI 14 = 73.8
   - 30-DTE 25-delta put IV = 35.2%
   - five-session IV z-score = 2.1
   - put extrinsic premium increase = 11.7%
   - put OI increase = 16.4%
   - quality = usable
5. Market State Builder creates AAPL state v1.
6. Transition Engine identifies:
   - RSI threshold crossing;
   - multi-maturity put IV expansion;
   - unusual put OI accumulation;
   - premium-price divergence.
7. Evidence Generator persists individual claims.
8. H001 evaluator combines the evidence.
9. Evaluation is triggered with:
   - magnitude 0.78
   - rarity 0.84
   - persistence 0.66
   - corroboration 0.83
   - quality 0.92
   - confidence 0.81
10. Existing proposal generator creates a reviewable signal proposal.
11. Opportunity Engine groups H001 with any compatible volatility-regime hypothesis.
12. Analyst sees one ranked AAPL opportunity with evidence and conflicts.
13. At 5, 10, and 20 sessions, the Outcome Evaluator records realized outcomes.
14. Backtest and calibration views update for H001 v1.
```

---

## 27. Acceptance Criteria for the Complete v1

The v1 implementation is complete only when all of the following are true:

1. Existing MarketOps functionality continues to pass its validation gates.
2. AAPL can be processed end-to-end from normalized data to market state.
3. Feature observations are deterministic, versioned, and traceable.
4. DTE/delta cells retain selected source-contract lineage.
5. State transitions are persisted rather than calculated only in the UI.
6. Quality failures block relevant hypotheses without deleting audit evidence.
7. At least four v1 hypotheses can be evaluated historically.
8. Hypothesis evaluations cannot write directly to production signals.
9. Existing proposal review and materialization controls remain mandatory.
10. Opportunities combine signals without double counting correlated evidence.
11. Forward outcomes can be materialized without mutating source records.
12. Backtests enforce point-in-time correctness.
13. Knowledge graph changes remain proposal-based.
14. Syncratic Ask receives bounded, evidence-pure context.
15. Top 50 execution remains bounded by explicit operator parameters.
16. The system exposes enough lineage to reproduce any signal proposal.
17. Confidence is not presented as guaranteed future performance.
18. All semantic definitions are versioned.

---

## 28. Code-Agent Execution Rules

The code agent MUST:

- inspect existing migrations, repository conventions, schemas, APIs, workers, and tests before adding parallel abstractions;
- reuse current ledger, run, proposal, audit, and routing patterns;
- implement changes incrementally by phase;
- include migrations and rollback considerations;
- include unit and integration tests with each component;
- not silently change existing detector behavior;
- not enable automatic scheduling;
- not promote a hypothesis automatically;
- not bypass proposal review;
- not bypass graph review;
- not introduce provider calls inside analytical calculators;
- not use future data in backtests;
- not equate missing data with zero;
- not infer institutional intent as fact;
- document configuration defaults;
- produce run summaries and validation evidence for each phase.

For each phase, the code agent should return:

1. files changed;
2. migrations added;
3. APIs added or changed;
4. tests added;
5. commands used to validate;
6. remaining limitations;
7. evidence that existing MarketOps gates still pass.

---

## 29. Final Architectural Position

MarketOps should continue to preserve options-chain and distribution evidence, but those records are an analytical substrate rather than the primary product abstraction.

The primary abstraction is the **Market State**. The primary analytical event is the **State Transition**. The primary governed research object is the **Hypothesis**. The primary analyst-facing object is the **Opportunity**. The primary learning mechanism is the relationship between a hypothesis-triggered signal and its later measured outcome.

This architecture keeps the existing SignalOps strengths—immutability, replay, quality control, proposal governance, graph review, and bounded reasoning—while creating a disciplined path from broad market data to precise, backtestable, high-impact signals.
