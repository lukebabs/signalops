# Market State Intelligence Evaluation

Status: architecture evaluation
Date: 2026-07-19
Reviewed target: `docs/design/syncratic_marketops_market_state_intelligence_architecture_v1.md`

## Purpose

This note evaluates the MarketOps project against the Market State Intelligence architecture from both functional-outcome and design perspectives.

The target design defines MarketOps as a hypothesis-driven market-state intelligence system. Its primary abstractions are:

- `MarketState`: one canonical, versioned asset/session state.
- `StateTransition`: a material change between current and prior states.
- `Evidence`: reusable claims derived from features and transitions.
- `Hypothesis`: a governed, versioned market thesis with explicit quality, trigger, calibration, and lifecycle rules.
- `Opportunity`: the analyst-facing grouping of compatible hypothesis evaluations.
- `Outcome`: the forward result used to evaluate historical performance.

The current implementation has many of the required operating controls, but it has not yet implemented those abstractions as first-class persisted objects.

## Current Functional Baseline

Implemented MarketOps capabilities include:

- Top 50 asset universe and MarketOps metadata boundary.
- Massive ingestion, raw event ledger, normalized event ledger, replay, and idempotency.
- Deterministic DSM detectors and persisted `signal.v1` records.
- Derived alert and insight lifecycle ledgers.
- DSM artifacts and review-controlled graph proposals.
- Historical back-test substrate, baselines, evaluations, labels, readiness snapshots, and promotion candidates.
- Generic SignalOps algorithm substrate and the v0 adapter pack.
- Algorithm result persistence, proposal review, preflight, and materialization ledgers.
- Options chain storage, options distribution storage, feature materialization, bounded coverage runner, and quality-aware proposal gating.
- Syncratic bounded context windows and Ask integration without ingesting MarketOps data into Syncratic core.
- Frontend views for assets, DSM workbench, algorithms, backtests, options quality, proposal review, materialization, and Syncratic reasoning.

This baseline is a strong operational substrate. It preserves evidence, review boundaries, quality gates, and replayability. The main limitation is that it still presents much of the system through events, algorithm rows, proposals, and ledgers rather than a canonical market-state model.

## Functional Outcome Evaluation

| Outcome | Current state | Gap |
| --- | --- | --- |
| Daily canonical market state | Not implemented as a first-class object. Current data exists across normalized events, options distributions, algorithm results, DSM artifacts, and UI panels. | Analysts cannot inspect one authoritative asset/session state with completeness, quality, lineage, and evaluable hypotheses. |
| State-transition observations | Partially implied by detector metrics, options rolling windows, and algorithm scores. | Material changes are not persisted as reusable state transitions with lookback, rarity, persistence, divergence, or migration semantics. |
| Versioned hypothesis registry | Not implemented. DSM detectors and algorithm proposal policies are versioned, but H001-H007 are not registered hypotheses. | The system cannot govern, evaluate, calibrate, or promote market hypotheses as first-class research objects. |
| Evidence-first signal generation | Partially implemented through DSM artifacts, graph proposals, algorithm results, and proposal review. | There is no common evidence ledger connecting features, transitions, hypotheses, opportunities, Syncratic context, and outcomes. |
| Opportunity ranking | Not implemented. Current UI exposes alerts, insights, proposals, algorithms, and options evidence separately. | Analysts still need to manually synthesize related evidence into one actionable market opportunity. |
| Closed-loop outcome evaluation | Partially implemented for detector back-test metrics and calibration workflows. | Outcomes are not tied to hypothesis-triggered signals and opportunities across forward horizons. |
| Insight distinction from alert | Conceptually documented, not fully realized. | Insights can still appear too close to one persisted signal or incident; the design expects insights/opportunities to summarize multi-event state changes. |
| Syncratic explainability | Implemented as bounded Ask over SignalOps context. | Once states, transitions, hypotheses, and opportunities exist, Syncratic context should shift from raw event summaries toward evidence-pure market-state bundles. |

## Design Gap Evaluation

The target architecture is additive and aligns with existing SignalOps principles. It does not require replacing the raw ledger, normalized ledger, DSM detectors, proposal workflow, graph review, or Syncratic boundary.

Critical design gaps:

- No `marketops_feature_definitions` or generic MarketOps feature-observation registry exists.
- No canonical `marketops_market_states` model exists.
- No persisted `marketops_state_transitions` model exists.
- No shared `marketops_evidence` ledger exists.
- No `marketops_hypothesis_definitions` or `marketops_hypothesis_evaluations` registry exists.
- No `marketops_opportunities` queue exists.
- No `marketops_signal_outcomes` materializer exists.
- No API surface exists for `/v1/marketops/features`, `/states`, `/transitions`, `/evidence`, `/hypotheses`, `/opportunities`, or `/outcomes`.
- No worker topology exists for feature materialization, state building, transition calculation, evidence generation, hypothesis evaluation, opportunity building, or outcome materialization.
- No frontend Market State view, Surface matrix, Transition timeline, Hypothesis workbench, or Opportunity queue exists as defined by the architecture.

The design does not conflict with the implemented algorithm substrate. The algorithm layer should become one producer of feature observations or evidence, while hypothesis evaluation remains the governed path to market-state signal proposals.

## Algorithm And Options Assessment

The implemented algorithm pack is useful infrastructure, but it does not yet make insights materially more useful by itself. The algorithms currently score feature rows and generate quality-gated proposals. That is necessary but insufficient for market-state intelligence because:

- an anomaly result is not the same as a market hypothesis;
- a z-score on one options ratio does not explain whether a broader state transition exists;
- proposal quality gating blocks bad evidence but does not rank good evidence into opportunities;
- reviewed materialization proves governance, not predictive value;
- the system needs forward outcomes and calibration by hypothesis version before confidence can imply historical reliability.

The current options work is well aligned with the target design. It establishes bounded provider usage, stored contract evidence, distribution snapshots, feature rows, and explicit quality states. The next design step is to normalize options evidence into stable DTE, delta, moneyness, premium, IV, OI, liquidity, and quality feature observations rather than relying only on raw contracts or aggregate ratio rows.

## Recommended Path Forward

### G136: Market State Foundation

Add first-class schemas and read surfaces for:

- feature definitions;
- feature observations;
- market states;
- state transitions;
- evidence records.

This gate should be storage/API focused and should not introduce automatic scheduling, production signal materialization, or broad Top 50 fanout.

### G137: AAPL Vertical Slice

Build one narrow asset/session path from persisted equity and options evidence to:

- underlying momentum features;
- selected IV, premium, positioning, liquidity, and quality features;
- one canonical market state;
- state transitions;
- reusable evidence.

The vertical slice should prove lineage, idempotency, quality blocking, and deterministic rebuild behavior.

### G138: Hypothesis Registry And Evaluator

Implement a small research-status hypothesis pack from the target architecture:

- H001: overbought downside-hedging expansion.
- H004: volatility term-structure regime shift.
- H006: premium-price divergence.
- H007: delta-bucket unusual OI accumulation.

The evaluator should persist triggered and non-triggered results with reason codes and should feed the existing proposal workflow only where lifecycle status and quality policy permit.

### G139: Opportunity Layer

Introduce analyst-facing opportunities that group compatible hypothesis evaluations by asset, session, direction, and horizon.

This is the likely point where insights become meaningfully different from alerts:

- alerts remain event-level records;
- opportunities summarize corroborated market-state changes;
- Syncratic explains the opportunity using bounded, evidence-pure context.

### G140: Outcome Evaluation

Materialize forward outcomes for hypothesis evaluations, materialized signals, and opportunities after the forecast horizon matures.

This closes the loop between hypothesis, evidence, signal, opportunity, and realized market behavior.

## Risks And Controls

- Provider fanout risk: keep all acquisition bounded by explicit symbols, maximum symbols, limits, and pages.
- Data-quality risk: preserve existing options quality states and map them into the shared quality model without treating zero or missing values as valid evidence.
- Look-ahead risk: all features, states, transitions, hypotheses, and outcomes need point-in-time rules before they are used for calibration.
- Analyst overload risk: do not surface every feature change as an alert; route state changes through hypotheses and opportunities.
- Model overconfidence risk: confidence must mean the transition exists and satisfies the hypothesis, not certainty of future price movement.
- Governance risk: hypothesis evaluations must not write production signals directly; existing proposal review, preflight, materialization, and graph review remain mandatory.

## Bottom Line

The current MarketOps system has the right control plane, evidence ledgers, algorithm substrate, options foundation, and explainability boundary. The major missing layer is the market-state intelligence model itself.

The definitive path forward is not to add more isolated detectors or more raw options data first. The next workstream should create the state, transition, evidence, hypothesis, opportunity, and outcome layers that convert existing evidence into fewer, more meaningful, historically testable analyst decisions.
