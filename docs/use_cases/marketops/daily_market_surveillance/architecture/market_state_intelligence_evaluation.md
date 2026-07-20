# Market State Intelligence Reconciliation

Status: reconciled against canonical architecture
Date: 2026-07-20
Canonical design: `docs/design/syncratic_marketops_market_state_intelligence_architecture_v1.md`

## Executive Position

MarketOps has implemented the structural backbone of the Market State Intelligence architecture, but it has not completed the analytical system described by that architecture.

The strongest completed areas are deterministic storage, lineage, quality blocking, research isolation, review boundaries, and bounded execution. The dominant gap is source-to-feature completeness: current live options evidence cannot yet support reliable premium, surface-change, delta-bucket open-interest, liquidity, and historical calibration logic. As a result, the state, hypothesis, opportunity, and outcome ledgers are operationally real but mostly sparse or quality-blocked in live use.

The architecture is now the governing design criterion. A gate is not considered complete merely because its schema, API, or UI exists; it must advance the functional path from bounded point-in-time evidence to calibrated, explainable analyst opportunities.

## Implemented Baseline

The current system provides:

- Top 50 MarketOps asset metadata and explicit app/domain/use-case boundaries.
- Massive ingestion paths, raw and normalized ledgers, replay, idempotency, and source lineage.
- Deterministic DSM signals, alert/insight lifecycle records, artifacts, and review-controlled graph proposals.
- Generic SignalOps algorithm definitions, execution requests, immutable results, review proposals, preflight, and controlled materialization.
- Back-test runs, campaigns, baselines, comparisons, labels, evaluation scores, readiness snapshots, and promotion-planning records.
- Bounded Syncratic context and on-demand Ask integration without ingesting MarketOps data into Syncratic core.
- Options chain, distribution, quality, feature, and capture ledgers.
- First-class MarketOps feature definitions, feature observations, market states, transitions, evidence, hypothesis definitions/evaluations, opportunities, and outcomes.
- A bounded AAPL coordinator for state, hypothesis, opportunity, and outcome processing.
- A prospective options capture path with canonical same-session spot, provider-side DTE/moneyness bounds, a candidate cap, transient aggregation, and compact selected-contract evidence.

## Phase Reconciliation

| Architecture phase | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Phase 1: foundation and schemas | Substantially complete | Migrations `000028`-`000031`, deterministic identities, versioned ledgers, shared quality states, and read APIs exist. | Mutation/job APIs, durable run status for all logical workers, immutable approved-definition enforcement, and broader integration coverage remain. |
| Phase 2: AAPL vertical slice | Structurally complete, functionally partial | G137/G143 build 29 definitions, 39 slots, states, transitions, quote-derived surface evidence, and explicit quality; G138 evaluates H001/H004/H006/H007. | Historical options coverage is absent, live quote completeness is unproven, and several canonical feature domains remain incomplete. |
| Phase 3: backtest integration | Partial | Generic G081-G105 back-test/calibration substrate and G140/G141 point-in-time outcomes/coordinator exist. | No hypothesis-specific back-test adapter, comparison/walk-forward modes, regime/event segmentation, calibration report, or hypothesis-version promotion evidence exists. |
| Phase 4: proposal and opportunity integration | Partial | G139 opportunity grouping, overlap suppression, conflict scoring, read APIs, and analyst queue exist. | No hypothesis-evaluation-to-proposal adapter, lifecycle eligibility bridge, opportunity review/disposition, or calibrated ranking exists. |
| Phase 5: graph and Ask | Foundation only | Existing graph proposals remain review-controlled; G088-G093 provide bounded Ask and evidence-purity controls. | State, transition, hypothesis, opportunity, and outcome graph mappings and Market State Ask bundles are absent. |
| Phase 6: Top 50 bounded rollout | Acquisition controls partial | Asset cohorts, bounded ingestion, capped symbol execution, quality summaries, and G142 provider budgets exist. | State/hypothesis/outcome processing is AAPL-only; no staged cohort rollout, operational state dashboard, or per-symbol end-to-end readiness exists. |

## Component Conformance

| Component | Current conformance | Critical gap |
| --- | --- | --- |
| Market Feature Pipeline | Partial | Deterministic observations and quality metadata exist, but the catalog lacks realized-volatility features, most IV/surface cells, quote-derived premium, delta/expiry OI migration, concentration, liquidity spreads, and event context. |
| Market State Builder | Partial | One versioned state per AAPL session with exact feature lineage exists. Completeness is low in live data, and Top 50/general-symbol execution is not implemented. |
| State Transition Engine | Partial | One-session differences plus trailing persistence, z-score, and percentile are persisted point-in-time. Multi-window acceleration, divergence, migration, concentration, curve, and regime operators are missing. |
| Evidence Ledger | Partial | Reusable return, OI-ratio, and ATM-IV-change claims exist with lineage. Evidence coverage is too narrow for the target hypothesis pack. |
| Hypothesis Registry | Partial | H001, H004, H006, and H007 v1 are versioned and research-only. H002, H003, and H005 are absent; no audited lifecycle mutation or approved immutability path exists. |
| Hypothesis Evaluator | Partial | Deterministic trigger/non-trigger/rejection rows and reason codes exist. Live evaluations are blocked by sparse inputs, and no proposal bridge exists. |
| Opportunity Engine | Partial | Grouping, domain overlap suppression, conflicts, scores, and research lifecycle exist. There is no review disposition, calibration contribution, or mature live opportunity population. |
| Outcome Evaluator | Partial | Immutable 1/5/10/20-session outcomes with source lineage and maturity states exist. No live triggered sources currently populate them, and no calibration aggregation by regime/confidence is implemented. |
| Data Acquisition | Partial | G142/G143 are analytically bounded and compact. Request IDs, page completeness, selected quotes/contract metadata, quote freshness, selection lineage, and candidate rejection metrics are captured; rate-limit telemetry and durable scheduling are absent. |
| API surface | Read-heavy partial | Core list/detail reads exist. Feature/state build, transition calculation, hypothesis mutation/evaluation, opportunity build/review, outcome materialization, coverage, calibration, and durable job-request APIs are absent. |
| Worker topology | CLI prototype | State materialization combines feature/state/transition/evidence stages; separate hypothesis, opportunity, outcome, and history CLIs exist. Durable job records, partial-failure ledgers, consistent version metrics, and approved scheduling are absent. |
| Frontend | Partial | Asset/options, algorithms, backtests, proposal controls, Syncratic views, and the opportunity queue exist. The specified Market State, DTE-delta surface, transition timeline, hypothesis workbench, and outcome calibration views do not. |
| Syncratic Ask | Partial | Bounded on-demand reasoning and evidence purity exist. Context is still signal/event oriented rather than a compact state/transition/hypothesis/opportunity bundle. |
| Observability | Partial | CLIs emit run summaries and G142 records capture metrics. Stage-level durable metrics, latency, incompatible-version counts, proposal rejection rates, opportunity dispositions, and calibration drift are missing. |
| Security and governance | Strong foundation | Existing auth, proposal review, graph review, and research/production separation are preserved. Audited hypothesis promotion and opportunity disposition controls remain absent. |

## Feature Catalog Reconciliation

Implemented or partially implemented:

- Underlying returns for 1/5/10/20 sessions, RSI 14, SMA-20 distance, volume ratio, gap, and ATR.
- ATM IV at 30/60/90 DTE and 30/60-DTE 25-delta put/call IV, quote-derived 30-DTE wing premium/spread, IV term slopes, 30/60-DTE risk reversal, and selection confidence.
- Put/call OI ratio, aggregate one-session ratio change, dimensioned 30-DTE wing OI change, put/call volume ratio, and transient moneyness/expiry/DTE/delta OI-volume buckets.
- Usable-contract, missing-IV, missing-Greeks, surface-coverage, and OI-quality observations.
- Deterministic source-contract, quote, provider-request, selection-cell, policy-version, and score lineage for seven selected surface cells.

Missing or not yet analytically usable:

- Realized volatility catalog (`rv_10d`, `rv_20d`, `rv_60d`, percentiles, acceleration).
- The broader 21/30/45/60/90/180/365-DTE by 0.15/0.25/0.35/ATM delta surface.
- IV daily/5-day changes, z-scores, percentiles, rank, IV-realized-volatility spread, broader skew/surface curvature/dispersion, and classified term-structure state beyond the initial G143 slopes and risk reversals.
- Straddles, expected move, broader premium ratios, and multi-session premium changes.
- Longitudinal bucket OI/volume changes, concentration, and migration beyond the current point-in-time transient distributions.
- Spread-weighted liquidity and broader standard-contract enforcement; G143 records quote staleness, crossed/zero markets, eligibility reasons, and selection confidence.
- Earnings, ex-dividend, corporate-action, and other point-in-time event context.

## Complete-v1 Acceptance Check

| # | Criterion | Status |
| --- | --- | --- |
| 1 | Existing MarketOps validations continue to pass | Pass |
| 2 | AAPL normalized data to market state | Pass structurally; live options completeness remains blocked |
| 3 | Deterministic, versioned, traceable feature observations | Pass |
| 4 | DTE/delta cells retain selected-contract lineage | Pass for seven G143 cells |
| 5 | Transitions are persisted | Pass |
| 6 | Quality failures block hypotheses without deleting audit evidence | Pass |
| 7 | Four v1 hypotheses evaluate historically | Partial: provider-shaped G143 path makes all four eligible; live historical options evidence is insufficient |
| 8 | Evaluations cannot directly write production signals | Pass |
| 9 | Proposal review/materialization controls remain mandatory | Pass as a governance rule; hypothesis proposal bridge is absent |
| 10 | Opportunities avoid correlated double counting | Pass in deterministic research logic |
| 11 | Outcomes materialize without source mutation | Pass |
| 12 | Backtests enforce point-in-time correctness | Partial: G141/G140 do; hypothesis back-test modes are incomplete |
| 13 | Graph changes remain proposal-based | Pass |
| 14 | Ask receives bounded evidence-pure context | Partial: generic context is bounded; state/opportunity context is absent |
| 15 | Top 50 execution is explicitly bounded | Pass for acquisition; analytical execution remains AAPL-only |
| 16 | Lineage can reproduce any signal proposal | Partial: algorithm proposals have lineage; hypothesis proposals do not exist |
| 17 | Confidence is not presented as guaranteed performance | Pass |
| 18 | Semantic definitions are versioned | Partial: ledgers are versioned; acquisition/selection and some scoring policies need explicit semantic versions |

Complete-v1 status: not complete.

## G143 Resolved Cross-Layer Blocker

G143 resolves the immediate G137/G138 key-and-dimension incompatibility. The provider adapter, selected evidence ledger, and state materializer now produce 30/60-DTE 25-delta put/call IV, quote-derived 30-DTE premium/spread, and dimensioned 30-DTE OI change. Request, pagination, quote, contract, selection-policy, and selected-contract lineage are explicit.

The production-shaped acceptance test starts with Massive HTTP payloads and makes H001, H004, H006, and H007 eligible through generated G137 observations and transitions. The isolated G138 positive fixtures are no longer the only compatibility proof.

The remaining blocker is longitudinal source coverage, not the feature-contract seam. G141 still needs at least 20 genuine analytics-ready option sessions before live calibration/outcome claims are meaningful. G143 does not manufacture that history or lower the threshold.

## Reconciled Gate Sequence

### G143: Options Surface Evidence v1

Status: implemented 2026-07-20.

Implemented the smallest evidence set required by H001, H004, H006, and H007:

- parse provider bid, ask, quote timestamp, exercise style, shares per contract, and request lineage where available;
- persist quote and eligibility fields only for deterministic selected source contracts;
- produce versioned 30/60/90-DTE ATM and 30/60-DTE 25-delta put/call observations where justified by the four hypotheses;
- aggregate delta/DTE/expiry positioning and quality from bounded transient candidates rather than storing the full chain;
- calculate midpoint, extrinsic premium, spread, IV term slopes, risk reversal, bucket OI/volume, and selection confidence;
- record request ID, pages, candidate completeness, eligibility counts, and rejection reasons;
- reconcile G137 output keys/dimensions exactly with the G138 required-feature contract;
- preserve point-in-time session and source lineage and keep provider calls outside analytical calculators.

Acceptance includes one end-to-end fixture that starts with Massive-shaped bounded records, passes through selected evidence and G137 materialization, and makes each intended G138 hypothesis eligible without injecting feature observations directly. Validation also verifies bounded dry-run behavior, no bulk candidate persistence, stable reruns, and explicit missing/partial behavior. The gate does not widen to the full architecture surface without a registered hypothesis need.

### G144: Market Feature And Transition Completion

Add the remaining inputs needed by the initial four hypotheses: realized volatility, IV/premium/OI changes, curve transitions, delta-bucket accumulation, persistence, divergence, and relevant event context. Generalize the materializer from AAPL to explicit bounded symbol cohorts without enabling scheduling.

### G145: Hypothesis Backtest And Calibration

Adapt hypothesis evaluations and outcomes into the existing isolated back-test substrate. Add comparison and walk-forward modes, sample-size warnings, regime/event segmentation, calibration summaries, and version isolation. Promotion remains advisory and operator-controlled.

### G146: Hypothesis Proposal And Opportunity Governance

Bridge only eligible lifecycle states into the existing proposal workflow, retain preflight/review/materialization controls, add opportunity analyst disposition, and keep alerts event-level while opportunities become the multi-evidence insight object.

### G147: Market State Analyst Experience

Build the specified asset/date state view, DTE-delta surface, transition timeline, hypothesis workbench, and calibrated opportunity detail. Keep provider acquisition, research calculation, proposal review, signal materialization, graph review, and opportunity disposition as distinct controls.

### G148: Graph, Ask, And Cohort Rollout

Add review-controlled graph mappings and bounded Market State Ask bundles, then stage explicit Top 50 cohorts with per-symbol quality/readiness dashboards. Do not schedule broad collection until provider budget, ownership, and operational thresholds are approved.

## Design Corrections Going Forward

- Do not use contract count or page count as evidence quality.
- Do not persist full chains by default for analytical state construction.
- Do not add a feature without a registered hypothesis or analyst question.
- Do not treat an algorithm anomaly as a production hypothesis or an insight.
- Do not make alerts and insights duplicate one-event lifecycle rows; opportunities are the intended multi-evidence analyst insight.
- Do not add UI ahead of usable state semantics and quality metadata.
- Do not lower quality or historical-coverage thresholds to manufacture triggers.
- Do not introduce hidden scheduling, automatic hypothesis promotion, direct signal writes, or direct graph mutation.

## Current Bottom Line

MarketOps is beyond a prototype database: it has genuine deterministic state, hypothesis, opportunity, and outcome infrastructure with strong governance. It is not yet a useful complete market-state intelligence product because the live options feature plane and longitudinal calibration evidence are incomplete.

G143 has closed the immediate provider-to-hypothesis compatibility gap while retaining bounded selected evidence. The definitive next step is G144 Market Feature And Transition Completion; live research remains coverage-blocked until genuine prospective option sessions accumulate.

## References

- Canonical architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- Functional component inventory: `functional_components.md`
- G136 foundation: `../gates/G136_market_state_foundation.md`
- G137 AAPL slice: `../gates/G137_aapl_market_state_vertical_slice.md`
- G138 hypothesis evaluator: `../gates/G138_research_hypothesis_evaluator.md`
- G139 opportunity layer: `../gates/G139_opportunity_layer.md`
- G140 outcomes: `../gates/G140_forward_outcome_evaluation.md`
- G141 history coordinator: `../gates/G141_historical_coverage_and_outcome_population.md`
- G142 prospective options capture: `../gates/G142_prospective_options_analytics_capture.md`
- G143 options surface evidence: `../gates/G143_options_surface_evidence_v1.md`
- State feature materializer: `../../../../../internal/marketops/state/materializer.go`
- Hypothesis feature contract: `../../../../../internal/marketops/hypotheses/registry.go`
