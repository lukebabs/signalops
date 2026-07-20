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

This baseline is a strong operational substrate. It preserves evidence, review boundaries, quality gates, and replayability. G136-G141 now add a bounded canonical state, research-hypothesis, opportunity, analyst-workbench, point-in-time outcome path, and historical execution coordinator. Live AAPL equity coverage is 135 sessions; historical option analytics remain insufficient and correctly block triggered sources and outcomes.

## Functional Outcome Evaluation

| Outcome | Current state | Gap |
| --- | --- | --- |
| Daily canonical market state | G136 provides the ledger/API, G137 materializes AAPL states, and G141 has 135 persisted equity sessions available for strict historical execution. | Historical options analytics remain sparse; the target analyst Market State view is absent. |
| State-transition observations | G136/G137 persist one-session numeric transitions; G141 adds trailing-only persistence, z-score, and empirical percentile after 20 prior observations. | Multi-lookback divergence and migration semantics remain incomplete. |
| Versioned hypothesis registry | G138 implements tenant-scoped H001/H004/H006/H007 v1 definitions in `research` status plus deterministic evaluation rows. | Promotion, broad historical calibration, and approved production materialization remain future gates. |
| Evidence-first signal generation | G136/G137 provide a shared evidence ledger, G138 links eligible inputs/evidence to research evaluations, and G140 can measure triggered evaluations. | Opportunity-scoped Syncratic context and a governed hypothesis-to-signal proposal adapter remain absent. |
| Opportunity ranking | G139 implements deterministic research-only grouping, overlap suppression, conflict scoring, contribution/evidence lineage, list/detail APIs, and the deployed analyst workbench. | Historical calibration remains absent; G140 can measure opportunity outcomes once valid opportunity rows exist. |
| Closed-loop outcome evaluation | G140 adds deterministic 1/5/10/20-session outcomes; G141 supplies 135 live equity sessions and a strict coordinated execution path. | Zero analytics-ready option sessions produce zero live triggers/outcomes; materialized-signal adaptation still awaits an explicit hypothesis-to-signal link. |
| Insight distinction from alert | Conceptually documented, not fully realized. | Insights can still appear too close to one persisted signal or incident; the design expects insights/opportunities to summarize multi-event state changes. |
| Syncratic explainability | Implemented as bounded Ask over SignalOps context. | Once states, transitions, hypotheses, and opportunities exist, Syncratic context should shift from raw event summaries toward evidence-pure market-state bundles. |

## Design Gap Evaluation

The target architecture is additive and aligns with existing SignalOps principles. It does not require replacing the raw ledger, normalized ledger, DSM detectors, proposal workflow, graph review, or Syncratic boundary.

Critical design gaps:

- G136 provides `marketops_feature_definitions`, typed feature observations, canonical market states, state transitions, and reusable evidence ledgers; G137 materialization is currently bounded to AAPL.
- G138 now provides `marketops_hypothesis_definitions` and `marketops_hypothesis_evaluations`; current scope is bounded to research-only H001/H004/H006/H007 over AAPL G137 states.
- G139 provides the `marketops_opportunities` research queue and deployed workbench; current AAPL quality produces zero eligible rows, which the UI explains through bounded rejection diagnostics.
- G140 provides `marketops_signal_outcomes` and a bounded AAPL materializer; the current live ledger is empty because no evaluation is triggered and no opportunity exists.
- G141 adds exact-symbol bounded equity-history acquisition, trailing point-in-time transition statistics, and strict coordinated G137-G140 execution. Live equity coverage is 135 sessions; options preflight blocks writes at zero analytics-ready sessions.
- G136-G140 provide read APIs for features, states, transitions, evidence, hypotheses, evaluations, opportunities, and outcomes.
- State, hypothesis, opportunity, and outcome calculation remain explicit bounded CLIs rather than a scheduled worker topology.
- No frontend Market State view, Surface matrix, Transition timeline, Hypothesis workbench, or Outcome calibration view exists as defined by the architecture.

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

Implemented as a bounded AAPL point-in-time materializer for eligible triggered evaluations and opportunities at 1, 5, 10, and 20 sessions. The schema reserves materialized signals, but the v1 adapter does not infer a hypothesis relationship from unrelated signal rows.

This establishes the closed-loop storage and calculation contract. Broader historical coverage and actual triggered sources are required before outcome calibration can be trusted.

### G141: Historical Coverage And Outcome Population

Implemented as a strict AAPL historical coordinator plus bounded exact-symbol Massive equity acquisition. It computes persistence and rarity from prior transitions only, requires 60 equity sessions and 20 analytics-ready option sessions, and blocks all writes when coverage fails. The live equity requirement is satisfied at 135 sessions; historical IV-surface coverage remains the global blocker, while open-interest, premium, and distribution gaps continue to reject their dependent hypotheses.

## Risks And Controls

- Provider fanout risk: keep all acquisition bounded by explicit symbols, maximum symbols, limits, and pages.
- Data-quality risk: preserve existing options quality states and map them into the shared quality model without treating zero or missing values as valid evidence.
- Look-ahead risk: all features, states, transitions, hypotheses, and outcomes need point-in-time rules before they are used for calibration.
- Analyst overload risk: do not surface every feature change as an alert; route state changes through hypotheses and opportunities.
- Model overconfidence risk: confidence must mean the transition exists and satisfies the hypothesis, not certainty of future price movement.
- Governance risk: hypothesis evaluations must not write production signals directly; existing proposal review, preflight, materialization, and graph review remain mandatory.

## Bottom Line

MarketOps now has the state, transition, evidence, hypothesis, opportunity, and outcome substrate plus a bounded historical coordinator. The remaining critical path is source coverage: obtain point-in-time option analytics without synthetic reconstruction, produce real triggered evaluations, and only then add calibration summaries or promotion decisions.
