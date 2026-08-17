# Syncratic SignalOps
# Signal Assurance Framework (SAF)
## Engineering Specification for Codex Agent
### Version 1.1

---

## 1. Purpose

The Signal Assurance Framework (SAF) is a first-class subsystem of Syncratic SignalOps responsible for continuously validating the effectiveness of analytical signals after they are confirmed.

SAF is not a next-day prediction validator.

SAF is designed to answer:

> After a signal is confirmed, does the expected outcome eventually materialize, how long does it take, how strong is the outcome, what adverse path occurs before materialization, and under what conditions does the signal perform best or worst?

The framework MUST support continuous, longitudinal evaluation of signals across assets, algorithms, market regimes, score bands, signal combinations, and algorithm versions.

SAF MUST preserve point-in-time correctness and MUST NOT use future information in validation.

SAF MUST be deterministic by default.

SAF MUST be idempotent.

SAF MUST be explainable.

SAF MUST support historical replay and live production evaluation using the same evaluation logic.

### 1.1 v1.1 Normative Decisions

This revision makes SAF an extension of the existing MarketOps forward-outcome capability, not a second competing outcome ledger. `marketops_signal_outcomes` remains the immutable, horizon-specific forward-outcome record for existing research workflows. SAF owns the assertion lifecycle, resolved validation contract, repeatable evaluation history, and assurance aggregates. A SAF evaluation may project a compatible fixed-horizon result into `marketops_signal_outcomes`; it MUST NOT write a second independent outcome for the same `(tenant_id, source_type, source_id, horizon_sessions, calculation_version)`.

SAF identifiers and foreign keys use the repository's canonical `TEXT` IDs. They MUST NOT introduce UUID-only keys for signals, tenants, or assets.

The source `signal.v1` event is immutable for SAF purposes. Because that schema forbids undeclared fields, publishers MUST NOT add SAF fields to it. Eligible confirmed signals are instead emitted through the versioned internal `marketops.signal.assurance.eligible.v1` event defined in Section 32, which references a signal already persisted in `signal_ledger` before assertion registration.

---

## 2. Strategic Role in SignalOps

SignalOps follows the operating model:

```text
DATA
  ↓
SIGNALS
  ↓
EVIDENCE
  ↓
OPPORTUNITY
  ↓
ANALYST DECISION
  ↓
REAL-WORLD OUTCOME
  ↓
SIGNAL ASSERTION
  ↓
VALIDATION
  ↓
ALGORITHM CALIBRATION
  └──────────────────────→ SIGNALS
```

SAF closes the feedback loop.

Analytical engines produce signals.

SignalOps correlates signals into evidence.

Opportunity orchestration engines consume evidence.

Analysts retain decision authority and conviction.

SAF evaluates whether confirmed signals produced the outcomes their assertion contracts claimed they should produce.

---

## 3. Primary Objectives

SAF SHALL:

1. Register every eligible confirmed signal as a testable assertion.
2. Capture the exact baseline state at confirmation time.
3. Evaluate assertions at deterministic intervals.
4. Measure time-to-materialization.
5. Measure maximum favorable excursion.
6. Measure maximum adverse excursion.
7. Measure absolute return.
8. Measure benchmark-relative return.
9. Measure sector-relative return where applicable.
10. Track signal invalidation.
11. Track expiry.
12. Track supersession by stronger contradictory signals.
13. Track algorithm version.
14. Track score at confirmation.
15. Track market regime at confirmation and throughout lifecycle.
16. Support multiple validation horizons.
17. Produce aggregate effectiveness metrics.
18. Evaluate score calibration.
19. Evaluate signal combinations.
20. Support historical backtesting without look-ahead bias.
21. Remain observational in v1.
22. Never automatically modify production algorithm weights.

---

## 4. Non-Goals

SAF v1 SHALL NOT:

- Predict future price movement.
- Trade assets.
- Execute orders.
- Modify VC, DOSM, technical, options, or other production algorithm weights automatically.
- Treat next-day movement as the default success condition.
- Assume every signal has the same validation horizon.
- Assume every signal has the same materialization rule.
- Replace existing analytical engines.
- Recompute analytical indicators.
- Use LLM judgment to determine success or failure.
- Infer success from narrative commentary.
- Mutate historical assertion baselines.

---

## 5. Key Concept: Signal Assertion

A confirmed signal becomes a testable assertion.

Example:

```json
{
  "signal_type": "DOSM_DISTRESSED_VALUE",
  "symbol": "XYZ",
  "direction": "bullish",
  "algorithm": "dosm",
  "algorithm_version": "2.3",
  "signal_score": 84.7,
  "confirmed_at": "2026-08-01T20:00:00Z",
  "validation_contract": {
    "primary_metric": "sector_relative_return",
    "threshold": 0.10,
    "evaluation_windows": [20, 30, 60, 90, 180],
    "max_horizon_trading_days": 180
  }
}
```

The assertion states:

> Given this signal, under this algorithm version, at this score, and under this market context, the expected condition should materialize within the defined horizon.

---

## 6. Signal Lifecycle

Every assertion SHALL use the following lifecycle:

```text
CANDIDATE
    ↓
CONFIRMED
    ↓
ACTIVE
    ├──────────────→ MATERIALIZED
    │                    ↓
    │                  CLOSED
    │
    ├──────────────→ INVALIDATED
    │                    ↓
    │                  CLOSED
    │
    ├──────────────→ SUPERSEDED
    │                    ↓
    │                  CLOSED
    │
    └──────────────→ EXPIRED
                         ↓
                       CLOSED
```

### 6.1 CANDIDATE

Signal exists but has not crossed the algorithm-specific confirmation threshold.

Candidate signals SHOULD NOT create SAF assertions unless explicitly enabled for research.

### 6.2 CONFIRMED

Signal crosses its confirmation threshold.

SAF creates a signal assertion.

Baseline values are captured.

### 6.3 ACTIVE

The assertion is being evaluated.

### 6.4 MATERIALIZED

The materialization rule has been satisfied.

### 6.5 INVALIDATED

A deterministic invalidation rule has been met before materialization.

### 6.6 SUPERSEDED

A stronger contradictory signal or algorithm state makes the original assertion no longer operationally relevant.

### 6.7 EXPIRED

Maximum evaluation horizon is reached without materialization.

### 6.8 CLOSED

Terminal archival state.

### 6.9 Lifecycle Transition and Precedence Rules

`ACTIVE` is the only evaluable state. `MATERIALIZED`, `INVALIDATED`, `SUPERSEDED`, and `EXPIRED` are terminal outcome states; each transitions to `CLOSED` only by an explicit archival operation and retains its terminal outcome. No terminal outcome may be replaced by another.

For one evaluation session, apply conditions in this order: (1) materialization, (2) invalidation, (3) supersession, (4) expiry. The first satisfied condition wins. A missing required input prevents all lifecycle transitions for that session, except a contract-specific delisting rule.

A transition is committed with its evaluation row and transactional outbox record in one database transaction. The outbox has a unique `(assertion_id, transition_sequence, event_type)` key; publishers mark the outbox row delivered rather than recreating it.

---

## 7. Time-to-Materialization

Time-to-Materialization (TTM) is a primary SAF metric.

```text
TTM_calendar = materialized_at - confirmed_at
```

For equity signals also calculate:

```text
TTM_trading = count_trading_sessions(confirmed_at, materialized_at)
```

Store both.

The framework MUST support materialization probability by horizon.

Default reporting horizons:

- 5 trading days
- 10 trading days
- 20 trading days
- 30 trading days
- 60 trading days
- 90 trading days
- 180 trading days

---

## 8. Materialization Contracts

Materialization rules MUST be signal-specific.

Each signal publisher SHALL provide or reference a validation contract.

Canonical contract:

```json
{
  "signal_type": "DOSM_DISTRESSED_VALUE",
  "direction": "bullish",
  "validation": {
    "primary_metric": "sector_relative_return",
    "threshold": 0.10,
    "comparison_operator": ">=",
    "evaluation_windows": [20, 30, 60, 90, 180],
    "max_horizon_trading_days": 180,
    "benchmark": "SPY",
    "sector_benchmark_source": "asset_registry",
    "materialization_policy": "FIRST_THRESHOLD_CROSS",
    "invalidation_policy": "SIGNAL_SPECIFIC"
  }
}
```

---

## 9. Example Validation Contracts

### 9.1 Bullish Technical Trend

Possible materialization:

```text
benchmark_relative_return >= +3%
within 20 trading days
```

### 9.2 Bearish Technical Trend

```text
benchmark_relative_return <= -3%
within 20 trading days
```

### 9.3 DOSM Distressed Value

Possible production rule:

```text
sector_relative_return >= +10%
within 90 trading days
```

### 9.4 VC Advanced Value

Possible rule:

```text
price >= 0.80 * model_fair_value_at_confirmation
within 180 trading days
```

or:

```text
sector_relative_return >= +12%
within 180 trading days
```

### 9.5 Options Bullish Positioning

Possible rule:

```text
benchmark_relative_return >= +3%
within 10 trading days
```

All production contracts MUST be explicit and version controlled.

---

## 10. Baseline Capture

At confirmation, SAF MUST capture immutable baseline data.

Required baseline fields:

```json
{
  "asset_price": 42.50,
  "asset_price_source": "regular_session_last_trade",
  "benchmark_symbol": "SPY",
  "benchmark_price": 615.22,
  "sector_benchmark_symbol": "XLK",
  "sector_benchmark_price": 251.80,
  "signal_score": 84.7,
  "algorithm_version": "2.3",
  "market_regime": "risk_on_normal_volatility",
  "captured_at": "2026-08-01T20:00:00Z"
}
```

Baseline records MUST NOT be updated after assertion creation.

---

## 11. Price Convention

For equity validation, SAF SHALL use MarketOps' canonical convention:

> last trade during the regular market session

If adjusted prices are required for historical return calculations, distinguish:

- raw_regular_session_close
- adjusted_close
- corporate_action_adjusted_return

The same convention MUST be used for live and historical evaluation.

For every baseline and evaluation, persist `price_source`, provider instrument identifier, exchange calendar ID and version, session date, bar end timestamp, retrieval/availability timestamp, price basis (`raw_regular_session_close` or `adjusted_close`), and corporate-action adjustment version. Returns within an assertion use one declared basis; a raw price MUST never be divided into an adjusted price. Dividends are excluded from price return unless the contract explicitly selects total return.

A confirmation received after that session's cutoff uses the next completed regular session as its baseline session. Trading-day counts use the captured exchange calendar, exclude the baseline session, and do not advance for missing-price sessions.

---

## 12. Primary Outcome Metrics

Each active assertion SHALL maintain:

- current absolute return
- benchmark return
- benchmark-relative return
- sector return
- sector-relative return
- MFE
- MAE
- current trading days active
- current calendar days active
- materialization status
- invalidation status
- latest evaluated price
- latest evaluation timestamp

---

## 13. Maximum Favorable Excursion

For bullish assertions:

```text
MFE = max(return_t) over assertion lifecycle
```

For bearish assertions, normalize favorable downside movement as positive MFE magnitude.

Store both normalized favorable excursion and signed return.

---

## 14. Maximum Adverse Excursion

For bullish assertions:

```text
MAE = min(return_t)
```

For bearish assertions:

```text
MAE = max(return_t)
```

Store timestamp of maximum adverse excursion.

MFE and MAE are calculated from the assertion's `excursion_metric`, which defaults to absolute return and may be `benchmark_relative_return` or `sector_relative_return` only when explicitly named by the resolved contract. For bullish assertions, normalized return equals the signed metric. For bearish assertions, normalized return equals the negated signed metric. `MFE` is the maximum normalized return and `MAE` is the minimum normalized return.

---

## 15. Path Quality

Two assertions with identical terminal return SHOULD NOT be considered equivalent.

Recommended initial metric:

```text
PathQuality = MFE / (abs(MAE) + epsilon)
```

Store the raw MFE and MAE regardless of derived ratio.

---

## 16. Benchmark-Relative Validation

Calculate:

```text
asset_return = asset_price_t / baseline_asset_price - 1
```

```text
benchmark_return = benchmark_price_t / baseline_benchmark_price - 1
```

```text
benchmark_relative_return = asset_return - benchmark_return
```

Sector-relative:

```text
sector_relative_return = asset_return - sector_return
```

Benchmark and sector symbols MUST be captured at assertion creation.

---

## 17. Signal Score Calibration

SAF SHALL evaluate whether higher signal scores correspond to stronger outcomes.

Default score bands:

```text
50-59
60-69
70-79
80-89
90-100
```

For each band calculate:

- count
- materialized_count
- materialization_rate
- median TTM
- P25 TTM
- P75 TTM
- median MFE
- median MAE
- median benchmark-relative return
- median sector-relative return

---

## 18. Signal Combination Validation

SAF SHALL support combination analysis.

Examples:

```text
DOSM only
VC only
Technical bullish only
Options bullish only
DOSM + Technical bullish
DOSM + Options bullish
DOSM + Technical + Options
VC + Technical
VC + Technical + Options
```

Combination definitions SHOULD use a configurable overlap window.

Store canonical combination keys deterministically.

---

## 19. Signal Convergence Analysis

For supported combinations calculate:

- sample size
- materialization rate
- median TTM
- MFE
- MAE
- excess return
- lift over component signals

```text
combination_lift =
combination_materialization_rate
-
baseline_signal_materialization_rate
```

---

## 20. Market Regime Segmentation

Every assertion SHALL record market regime at confirmation.

Example:

```json
{
  "trend_regime": "bull",
  "volatility_regime": "normal",
  "rates_regime": "stable",
  "risk_regime": "risk_on"
}
```

Regime calculation belongs to a separate Market Regime Engine.

SAF consumes regime labels and MUST NOT duplicate regime logic.

---

## 21. Assertion Data Model

Canonical assertion object:

```json
{
  "assertion_id": "ast_01K...",
  "tenant_id": "tenant_...",
  "asset_id": "asset_xyz",
  "symbol": "XYZ",
  "signal_id": "sig_...",
  "signal_type": "DOSM_DISTRESSED_VALUE",
  "signal_direction": "bullish",
  "signal_score": 84.7,
  "algorithm": "dosm",
  "algorithm_version": "2.3",
  "confirmed_at": "2026-08-01T20:00:00Z",
  "validation_state": "active",
  "baseline": {
    "price": 42.50,
    "benchmark": "SPY",
    "benchmark_price": 615.22,
    "sector_benchmark": "XLK",
    "sector_benchmark_price": 251.80
  },
  "materialization_rule": {
    "metric": "sector_relative_return",
    "threshold": 0.10,
    "operator": ">=",
    "max_horizon_trading_days": 90
  },
  "evaluation": {
    "trading_days_active": 31,
    "calendar_days_active": 45,
    "current_return": 0.074,
    "benchmark_return": 0.031,
    "sector_return": 0.028,
    "benchmark_relative_return": 0.043,
    "sector_relative_return": 0.046,
    "mfe": 0.102,
    "mae": -0.041
  },
  "materialized": false,
  "materialized_at": null,
  "invalidated_at": null,
  "expired_at": null
}
```

---

## 22. Persistence Model

Recommended tables:

```text
signal_assertions
signal_assertion_evaluations
signal_assertion_events
signal_validation_contracts
signal_effectiveness_snapshots
signal_combination_statistics
signal_calibration_statistics
signal_assurance_registration_inbox
```

Time-series evaluation records SHOULD use TimescaleDB if available.

---

## 23. signal_assertions Table

The illustrative DDL below is normative as amended by the v1.1 columns and constraints. The implementation MUST add `evaluation_mode TEXT NOT NULL CHECK (evaluation_mode IN ('LIVE','BACKTEST','RESEARCH'))`, `evaluation_run_id TEXT`, `source_ledger_signal_id TEXT NOT NULL`, `registration_idempotency_key TEXT NOT NULL`, `validation_contract_id TEXT NOT NULL`, `evaluation_engine_version TEXT NOT NULL`, `baseline_snapshot JSONB NOT NULL`, and `baseline_provenance JSONB NOT NULL`. `baseline_snapshot` contains all immutable baseline values; `baseline_provenance` contains the fields required by Section 11 and the point-in-time IDs for benchmark mapping, regime, fair value, and corporate-action policy.

Add `UNIQUE (tenant_id, registration_idempotency_key)` and `CHECK ((evaluation_mode = 'LIVE' AND evaluation_run_id IS NULL) OR (evaluation_mode <> 'LIVE' AND evaluation_run_id IS NOT NULL))`. Backtest and research assertions are isolated by `evaluation_run_id` and are never included in live aggregates.



```sql
CREATE TABLE signal_assertions (
    assertion_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    signal_id TEXT,
    signal_type VARCHAR(128) NOT NULL,
    signal_direction VARCHAR(16) NOT NULL,
    signal_score DOUBLE PRECISION,
    algorithm VARCHAR(64) NOT NULL,
    algorithm_version VARCHAR(32) NOT NULL,
    confirmed_at TIMESTAMPTZ NOT NULL,
    state VARCHAR(32) NOT NULL,
    baseline_price DOUBLE PRECISION NOT NULL,
    benchmark_symbol VARCHAR(32),
    baseline_benchmark_price DOUBLE PRECISION,
    sector_benchmark_symbol VARCHAR(32),
    baseline_sector_benchmark_price DOUBLE PRECISION,
    validation_contract_version VARCHAR(32) NOT NULL,
    validation_contract JSONB NOT NULL,
    regime_snapshot JSONB,
    materialized_at TIMESTAMPTZ,
    invalidated_at TIMESTAMPTZ,
    superseded_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 24. signal_assertion_evaluations Table

```sql
CREATE TABLE signal_assertion_evaluations (
    evaluation_id TEXT PRIMARY KEY,
    assertion_id TEXT NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL,
    evaluation_session_date DATE NOT NULL,
    evaluation_mode VARCHAR(16) NOT NULL,
    evaluation_run_id TEXT,
    input_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_completeness VARCHAR(32) NOT NULL DEFAULT 'COMPLETE',
    transition_sequence INTEGER NOT NULL DEFAULT 0,
    trading_days_active INTEGER NOT NULL,
    calendar_days_active INTEGER NOT NULL,
    asset_price DOUBLE PRECISION NOT NULL,
    benchmark_price DOUBLE PRECISION,
    sector_benchmark_price DOUBLE PRECISION,
    absolute_return DOUBLE PRECISION,
    benchmark_return DOUBLE PRECISION,
    sector_return DOUBLE PRECISION,
    benchmark_relative_return DOUBLE PRECISION,
    sector_relative_return DOUBLE PRECISION,
    mfe DOUBLE PRECISION,
    mae DOUBLE PRECISION,
    materialization_condition_met BOOLEAN NOT NULL DEFAULT FALSE,
    invalidation_condition_met BOOLEAN NOT NULL DEFAULT FALSE,
    evaluation_version VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(assertion_id, evaluation_session_date, evaluation_version)
);
```

---

## 25. Assertion Events Table

Events are an outbox-backed audit stream, not the authority for current state. Add `transition_sequence INTEGER NOT NULL`, `evaluation_id TEXT`, `evaluation_mode VARCHAR(16) NOT NULL`, `evaluation_run_id TEXT`, `idempotency_key TEXT NOT NULL`, and `published_at TIMESTAMPTZ`. Add `UNIQUE (assertion_id, transition_sequence, event_type)` and `UNIQUE (idempotency_key)`.

```sql
CREATE TABLE signal_assertion_events (
    event_id TEXT PRIMARY KEY,
    assertion_id TEXT NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    previous_state VARCHAR(32),
    new_state VARCHAR(32),
    reason_code VARCHAR(128),
    details JSONB,
    occurred_at TIMESTAMPTZ NOT NULL
);
```

Event types:

```text
ASSERTION_CREATED
ASSERTION_ACTIVATED
ASSERTION_EVALUATED
ASSERTION_MATERIALIZED
ASSERTION_INVALIDATED
ASSERTION_SUPERSEDED
ASSERTION_EXPIRED
ASSERTION_CLOSED
```

---

## 26. Validation Contract Registry

Validation rules MUST be versioned.

```sql
CREATE TABLE signal_validation_contracts (
    contract_id TEXT PRIMARY KEY,
    signal_type VARCHAR(128) NOT NULL,
    contract_version VARCHAR(32) NOT NULL,
    algorithm VARCHAR(64),
    algorithm_version VARCHAR(32),
    direction VARCHAR(16) NOT NULL,
    primary_metric VARCHAR(64) NOT NULL,
    comparison_operator VARCHAR(8) NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    evaluation_windows JSONB NOT NULL,
    max_horizon_trading_days INTEGER,
    materialization_policy VARCHAR(64) NOT NULL,
    invalidation_policy VARCHAR(64),
    config JSONB NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL,
    contract_scope_key TEXT NOT NULL,
    UNIQUE(signal_type, direction, contract_scope_key, contract_version)
);
```

`contract_scope_key` is the normalized `algorithm` and `algorithm_version` pair; an empty pair is the generic fallback.

The registry is append-only: a resolved contract is never updated or deleted. Activation changes only which eligible version may be selected for new assertions. Resolution first matches signal type, direction, algorithm, and algorithm version; a contract with a null algorithm or version is a fallback only when exactly one more-specific match does not exist. Ambiguous or missing resolution rejects registration and records an operational error.

---

## 27. Idempotency

SAF MUST be fully idempotent.

Assertion creation idempotency key:

```text
tenant_id + asset_id + signal_id + algorithm_version + confirmed_at
```

Daily evaluation key:

```text
assertion_id + evaluation_session_date + evaluation_version
```

Replay MUST NOT create duplicate lifecycle transitions.

---

## 28. Evaluation Cadence

Default equities workflow:

```text
Regular market close
    ↓
Market data finalization
    ↓
Algorithm runs complete
    ↓
New confirmed signals published
    ↓
SAF assertion registration
    ↓
Active assertion evaluation
    ↓
Lifecycle transitions
    ↓
Aggregate metrics refresh
```

Evaluate only after canonical session data is complete unless a contract explicitly requires intraday evaluation.

---

## 29. Daily Evaluation Worker

Pseudo-flow:

```text
load active assertions
    ↓
resolve evaluation date
    ↓
load canonical asset price
    ↓
load benchmark price
    ↓
load sector benchmark price
    ↓
calculate returns
    ↓
update MFE/MAE
    ↓
check materialization
    ↓
check invalidation
    ↓
check supersession
    ↓
check expiry
    ↓
persist evaluation
    ↓
emit lifecycle event if state changed
```

---

## 30. Go Service Interface

```go
type AssertionService interface {
    RegisterAssertion(
        ctx context.Context,
        signal ConfirmedSignal,
    ) (*SignalAssertion, error)

    EvaluateAssertion(
        ctx context.Context,
        assertionID string,
        asOf time.Time,
    ) (*AssertionEvaluation, error)

    EvaluateActiveAssertions(
        ctx context.Context,
        asOf time.Time,
    ) error
}
```

---

## 31. Validation Engine Interface

```go
type ValidationEngine interface {
    Evaluate(
        ctx context.Context,
        assertion SignalAssertion,
        market EvaluationMarketState,
    ) (ValidationResult, error)
}
```

---

## 32. Signal Publisher Contract

Analytical engines MUST first publish the existing `signal.v1` event. A trusted confirmation adapter then persists that signal in `signal_ledger` and emits `marketops.signal.assurance.eligible.v1`. The SAF event has its own versioned schema and MUST contain:

- `eligible_event_id`, `tenant_id`, `signal_id`, `asset_id`, `symbol`, and `signal_ledger_id`
- `signal_type`, `direction`, `score`, `algorithm`, `algorithm_version`, and `confirmed_at`
- `validation_contract_ref`, `confirmation_rule_version`, and `event_available_at`
- immutable `baseline_request` and point-in-time provenance references

SAF accepts only an eligible event whose `signal_id` matches an already persisted ledger record for the same tenant. The event ID is the registration correlation ID; the assertion idempotency key remains the key in Section 27.

The payload shape is:

```json
{
  "eligible_event_id": "safelig_...",
  "tenant_id": "tenant_...",
  "signal_id": "sig_...",
  "signal_ledger_id": "sig_...",
  "asset_id": "asset_xyz",
  "symbol": "XYZ",
  "signal_type": "DOSM_DISTRESSED_VALUE",
  "direction": "bullish",
  "score": 84.7,
  "status": "confirmed",
  "algorithm": "dosm",
  "algorithm_version": "2.3",
  "confirmed_at": "2026-08-01T20:00:00Z",
  "event_available_at": "2026-08-01T20:01:00Z",
  "confirmation_rule_version": "dosm-confirmation:v1",
  "validation_contract_ref": "DOSM_DISTRESSED_VALUE:v1",
  "baseline_request": {"price_basis": "raw_regular_session_close"},
  "provenance_refs": {"market_calendar_version": "nyse:v1"}
}
```

Missing, inactive, or ambiguous validation contracts reject assertion registration, emit a durable `signal_assurance.contract_resolution_failed` operational event, and do not retry until contract state or input data changes.

---

## 33. Internal Events

SAF SHOULD publish:

```text
signal_assertion.created
signal_assertion.materialized
signal_assertion.invalidated
signal_assertion.superseded
signal_assertion.expired
signal_assertion.closed
signal_effectiveness.updated
```

---

## 34. Effectiveness Aggregation

Aggregate by:

```text
algorithm
algorithm_version
signal_type
direction
score_band
sector
market_regime
signal_combination
```

Metrics:

```text
assertions_created
assertions_active
assertions_materialized
assertions_invalidated
assertions_expired
materialization_rate
median_ttm
p25_ttm
p75_ttm
median_mfe
median_mae
median_absolute_return
median_benchmark_relative_return
median_sector_relative_return
```

---

## 35. Materialization Curve

For each signal type and version calculate:

```text
P(materialized by T)
```

Initial implementation may use deterministic empirical ratios.

Later versions MAY use Kaplan-Meier estimators to correctly handle right-censored active assertions.

---

## 36. Censoring

Active assertions that have not reached maximum horizon are right-censored.

They MUST NOT automatically count as failures.

For a horizon-specific materialization rate, include only assertions that have either:

- completed the captured exchange-calendar session for that horizon, or
- materialized on or before that horizon.

Invalidated, superseded, delisted, and data-unavailable assertions are excluded from the materialization-rate denominator unless an explicit report filter selects them. Every aggregate response reports each excluded count and the censoring rule.

---

## 37. Point-in-Time Correctness

Historical evaluation MUST reconstruct what was known at signal confirmation time.

Never use future:

- signal scores
- algorithm versions
- fair values
- benchmark mappings
- event metadata
- revised fundamentals

Baseline snapshots are immutable.

Evaluation records are append-only except controlled repair operations.

Each registration and evaluation stores an input snapshot containing immutable provider record IDs, source event IDs, observed/effective/available timestamps, calendar version, benchmark/sector mapping version, regime snapshot version, corporate-action policy version, and the exact resolved contract ID. A repair creates a new evaluation-engine version and preserves the superseded record; it never overwrites a baseline or silently replaces a published aggregate.

---

## 38. Backtesting

Use the same ValidationEngine for live and historical replay.

```text
load historical confirmed signals
    ↓
load point-in-time baselines
    ↓
instantiate assertions
    ↓
replay trading sessions
    ↓
run same evaluation logic
    ↓
persist isolated backtest results
```

Do not implement separate success logic for backtesting.

---

## 39. Live vs Backtest Namespace

Every assertion SHALL include:

```text
evaluation_mode = LIVE | BACKTEST | RESEARCH
```

Metrics MUST be filterable by mode.

---

## 40. Algorithm Versioning

All evaluations MUST preserve:

```text
algorithm
algorithm_version
validation_contract_version
evaluation_engine_version
```

Cross-version aggregation MUST be explicit.

---

## 41. Calibration Workflow

SAF MUST initially be observational.

```text
Production Algorithm
        ↓
Signal Assertions
        ↓
Validation Statistics
        ↓
Research Review
        ↓
Proposed Calibration
        ↓
Backtest
        ↓
New Algorithm Version
```

No automatic production mutation.

---

## 42. Dashboard Requirements

Provide:

- active assertions
- materialized assertions
- invalidated assertions
- expired assertions
- materialization rate
- median TTM
- P25/P75 TTM
- median MFE
- median MAE
- score calibration view
- materialization curve
- signal combination performance
- regime performance
- algorithm version comparison

---

## 43. Asset-Level View

For any asset show:

```text
Signal
Confirmed Date
Score
Algorithm Version
State
Days Active
Current Return
Excess Return
MFE
MAE
Materialized Date
TTM
```

Provide a lifecycle timeline.

---

## 44. API Endpoints

All APIs use the existing tenant-scoped boundary and require service authentication. Direct assertion creation is restricted to the confirmation adapter; operators use read APIs only.

```http
POST /v1/tenants/{tenant_id}/marketops/signal-assurance/assertions
GET  /v1/tenants/{tenant_id}/marketops/signal-assurance/assertions/{assertion_id}
GET  /v1/tenants/{tenant_id}/marketops/signal-assurance/assertions
GET  /v1/tenants/{tenant_id}/marketops/signal-assurance/effectiveness
GET  /v1/tenants/{tenant_id}/marketops/signal-assurance/calibration
GET  /v1/tenants/{tenant_id}/marketops/signal-assurance/materialization
GET  /v1/tenants/{tenant_id}/marketops/signal-assurance/combinations
```

List endpoints require bounded `limit` (default 50, maximum 200) and a cursor. Assertion lists accept `state`, `symbol`, `asset_id`, `signal_type`, `algorithm`, `algorithm_version`, `evaluation_mode`, and date-range filters. Aggregate endpoints require `evaluation_mode` and return the exact filters, sample size, censoring rule, calculation/evaluation-engine versions, and `as_of` timestamp. POST requires an `Idempotency-Key` and returns `201`, or `200` with the existing assertion for a matching key.

---

## 45. Sample Size Guardrails

Every aggregate metric MUST expose sample size.

Suggested flags:

```text
n < 20       INSUFFICIENT
20 <= n < 50 LOW_SAMPLE
50 <= n <100 MODERATE_SAMPLE
n >= 100     ESTABLISHED_SAMPLE
```

Thresholds SHOULD be configurable.

---

## 46. Missing Data

If asset price is unavailable:

- do not fabricate evaluation
- mark evaluation pending/missing
- retry later
- retain assertion as active

If benchmark price is unavailable:

- calculate absolute return if possible
- flag relative metrics incomplete

If sector benchmark is unavailable:

- do not silently substitute SPY

---

## 47. Corporate Actions

Splits, dividends, mergers, ticker changes, and spin-offs can corrupt returns.

SAF SHALL support corporate-action-adjusted historical return calculations.

Asset identity SHALL use immutable `asset_id`, not ticker alone.

---

## 48. Delistings

Do not silently drop active assertions.

Suggested terminal reason codes:

```text
DELISTED_ACQUISITION
DELISTED_BANKRUPTCY
DELISTED_OTHER
```

Materialization contracts MAY define treatment.

---

## 49. Supersession

A signal MAY be superseded when a deterministic contradictory signal becomes dominant.

Supersession rules MUST be explicit.

SAF SHALL NOT infer supersession subjectively.

---

## 50. Invalidation

Invalidation differs from expiry.

Invalidated assertions SHOULD be reported separately from expired assertions.

This distinction is important for evaluating whether the original thesis was actively disproven or merely failed to materialize in time.

---

## 51. Opportunity Engine Integration

Future orchestration engines such as EEOM MAY consume SAF metrics including:

- historical signal effectiveness
- materialization rate
- TTM distribution
- signal-combination lift
- regime-specific reliability

This is a later controlled integration.

SAF v1 is measurement-first.

---

## 52. Reliability Score Roadmap

Future versions MAY derive:

```text
Reliability = f(
  materialization_rate,
  calibration,
  sample_size,
  path_quality,
  regime_stability,
  version_stability
)
```

Do not use it as production weighting in v1 without separate approval.

---

## 53. Observability

Metrics:

```text
saf_assertions_created_total
saf_assertions_active
saf_assertions_materialized_total
saf_assertions_invalidated_total
saf_assertions_expired_total
saf_evaluations_total
saf_evaluation_failures_total
saf_missing_price_total
saf_processing_latency_seconds
saf_batch_duration_seconds
saf_contract_resolution_failures_total
```

Logs MUST include assertion, asset, signal type, version, evaluation date, and state transition.

---

## 54. Performance

Initial target:

```text
10,000 active assertions
daily evaluation
< 10 minutes batch execution
```

Architecture MUST scale horizontally.

Evaluation workers SHOULD partition by assertion or asset.

No evaluation may depend on mutable in-process state.

---

## 55. Concurrency

Multiple workers MAY evaluate separate assertions concurrently.

Lifecycle transitions MUST be atomic.

Duplicate materialization events MUST NOT be emitted.

---

## 56. Retry Model

Retry transient failures with bounded backoff.

Do not retry deterministic validation errors indefinitely.

Send non-recoverable items to a failure/dead-letter queue.

---

## 57. Multi-Tenancy

Every assertion and aggregate record MUST carry tenant_id.

No cross-tenant exposure.

---

## 58. Security

Service-to-service authentication is required.

Write endpoints are restricted to trusted SignalOps services.

Validation contract modification requires privileged authorization.

---

## 59. Testing Strategy

Unit tests MUST cover:

- returns
- excess returns
- MFE
- MAE
- TTM
- horizon eligibility
- materialization operators
- invalidation rules
- lifecycle transitions
- idempotency

Integration tests MUST cover eligible-event schema validation, ledger linkage, contract resolution, signal registration, price retrieval, evaluation, outbox publication, duplicate delivery, aggregation, and prevention of a duplicate `marketops_signal_outcomes` projection.

Concurrency tests MUST prove a single terminal transition and outbox event under racing workers. Missing-data, delisting, corporate-action, cutoff/session-calendar, bearish-normalization, and repair-version fixtures are required.

Replay tests MUST prove deterministic outcomes.

Point-in-time tests MUST prove future data cannot leak into historical evaluation.

---

## 60. Golden Test Cases

Create canonical deterministic fixtures.

Example bullish signal:

```text
Baseline = 100
Day 1 = 98
Day 5 = 96
Day 10 = 104
Day 15 = 108
Day 20 = 112
```

Expected:

```text
MFE = +12%
MAE = -4%
```

If materialization threshold is +10%:

```text
materializes on Day 20
```

Create corresponding bearish, expiry, invalidation, and supersession fixtures.

---

## 61. v1.1 Acceptance Criteria

SAF v1.1 is complete when:

1. The versioned eligible event validates, references an existing same-tenant `signal_ledger` record, and creates exactly one assertion for a registration idempotency key.
2. Assertions persist immutable resolved contracts, baseline snapshots, full input provenance, and an explicit LIVE/BACKTEST/RESEARCH namespace.
3. Daily live evaluation calculates absolute and benchmark-relative returns, normalized MFE/MAE, TTM, and the lifecycle precedence rules.
4. Transition, evaluation, and outbox persistence are atomic; duplicate delivery emits no duplicate lifecycle event.
5. Missing inputs produce an append-only incomplete evaluation and do not fabricate a result.
6. Live aggregates report sample size, censoring/exclusion counts, versions, and `as_of`; backtest/research data is excluded by default.
7. A compatible fixed-horizon projection does not duplicate an existing `marketops_signal_outcomes` record.
8. Replay uses only captured point-in-time inputs and produces deterministic, isolated results.
9. Authenticated tenant-scoped APIs enforce bounded pagination and trusted-only writes.

Combination analysis, regime segmentation, sector-relative returns, calibration, and dashboard views are Phase 2/3 capabilities, not v1.1 completion criteria.

---

## 62. Repository Integration Layout

SAF is implemented in this repository's root Go module and shared API/router, not as a nested standalone module. Add the domain packages under `internal/marketops/signalassurance/` (assertions, contracts, evaluation, lifecycle, aggregation, events); persistence under `internal/storage/postgres/`; routes under `internal/api/`; versioned event schemas under `contracts/events/`; and ordered shared migrations under `migrations/`. The worker is a root `cmd/marketops-signal-assurance-worker` command and is scheduled after the existing post-close data-finalization job.

---

## 63. Configuration

```yaml
signal_assurance:
  evaluation:
    schedule: "after_market_close"
    default_benchmark: "SPY"

  horizons:
    default: [5, 10, 20, 30, 60, 90, 180]

  sample_size:
    insufficient: 20
    low: 50
    moderate: 100

  aggregation:
    refresh_after_evaluation: true
```

---

## 64. Implementation Sequence

### Phase 1 — v1.1 Core

Implement:

- eligible-event schema, confirmation adapter, durable registration inbox, and ledger linkage
- append-only contract registry and deterministic resolution
- assertion/baseline/provenance persistence, idempotency, mode/run isolation, and transactional outbox
- daily evaluation using canonical session data, absolute and benchmark-relative returns, normalized MFE/MAE, TTM, and lifecycle precedence
- tenant-scoped read APIs, trusted-only registration, observability, deterministic replay fixtures, and compatible outcome projection

### Phase 2

Add:

- sector-relative returns
- calibration
- score bands
- materialization curves
- dashboard API

### Phase 3

Add:

- signal combinations
- market regimes
- advanced historical backtesting

### Phase 4

Add:

- survival analysis
- reliability modeling
- controlled feedback into opportunity orchestration

---

## 65. Design Principle for Codex Agent

Do not treat SAF as a reporting feature.

It is a durable SignalOps subsystem.

Do not embed validation logic in MarketOps UI code.

Do not embed validation logic inside VC, DOSM, technical, or options algorithms.

Signal engines publish signals.

SAF owns assertion lifecycle and outcome validation.

Opportunity engines MAY consume assurance metrics later.

---

## 66. Canonical Questions SAF Must Answer

```text
How often does this signal materialize?

How long does it normally take?

How much favorable movement occurs?

How much adverse movement occurs first?

Does performance improve with higher scores?

Does performance depend on market regime?

Do combinations of signals materially improve outcomes?

Has the current algorithm version improved over the previous version?
```

---

## 67. Final Architectural Principle

Signal quality must be empirically demonstrated rather than assumed.

The Signal Assurance Framework creates the evidence required to establish that quality.

The framework is part of the SignalOps trust architecture.

The intended operating loop is:

```text
DATA
  ↓
SIGNALS
  ↓
EVIDENCE
  ↓
OPPORTUNITY
  ↓
DECISION
  ↓
OUTCOME
  ↓
ASSURANCE
  ↓
LEARNING
  ↓
NEW VERSIONED SIGNALS
```

The system remains deterministic, version-controlled, auditable, and analyst-governed.

---

## 68. Operational Viability Extension

SAF effectiveness is not a dashboard-only statistic. A viable research signal must be evaluated as a versioned cohort with a declared horizon, immutable selection context, sample/maturity accounting, directional and benchmark-relative results, and an independent out-of-sample check. No viability state may create a trade instruction, alert, or automatic algorithm mutation.

The active primary-cohort charter, states, evidence gates, and phased delivery plan are maintained in [the Signal Assurance Viability Sprint](../projects/signal_assurance_viability/README.md).

---

# End of Specification
