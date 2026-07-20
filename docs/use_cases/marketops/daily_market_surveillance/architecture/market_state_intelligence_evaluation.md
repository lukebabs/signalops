# Market State Intelligence Reconciliation

Status: reconciled against canonical architecture
Date: 2026-07-20
Canonical design: `docs/design/syncratic_marketops_market_state_intelligence_architecture_v1.md`

## Executive Position

MarketOps has implemented the structural backbone of the Market State Intelligence architecture, but it has not completed the analytical system described by that architecture.

The strongest completed areas are deterministic storage, lineage, quality blocking, research isolation, review boundaries, bounded execution, and the initial provider-to-longitudinal-feature contract. The dominant gap is genuine source history and calibration: live options evidence has not accumulated enough eligible sessions for reliable rarity, outcome, comparison, or walk-forward claims, and no point-in-time earnings-calendar producer exists. The state, hypothesis, opportunity, and outcome ledgers are operationally real but remain sparse or quality-blocked in live use.

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
- A state-v2 materializer for explicit cohorts capped at 10 symbols, plus an AAPL-only coordinator for hypothesis, opportunity, and outcome processing.
- A prospective options capture path with canonical same-session spot, provider-side DTE/moneyness bounds, a candidate cap, transient aggregation, and compact selected-contract evidence.

## Phase Reconciliation

| Architecture phase | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Phase 1: foundation and schemas | Substantially complete | Migrations `000028`-`000031`, deterministic identities, versioned ledgers, shared quality states, and read APIs exist. | Mutation/job APIs, durable run status for all logical workers, immutable approved-definition enforcement, and broader integration coverage remain. |
| Phase 2: initial hypothesis slice | Structurally complete, functionally partial | G137/G143/G144 build 44 definitions, 69 state-v2 slots, longitudinal and event context, multi-window transitions, bounded explicit-symbol execution, quote-derived surface evidence, and explicit quality; G138 evaluates H001/H004/H006/H007. | Historical options coverage is absent, live quote/event completeness is unproven, and broader surface/concentration/migration domains remain incomplete. |
| Phase 3: backtest integration | Substantially complete, evidence sparse | G145 adapts exact hypothesis/outcome versions into existing isolated runs and calibration summaries with single/comparison/walk-forward modes, point-in-time event/regime segmentation, confidence metrics, and sample warnings. | Live matured samples remain insufficient; cross-sectional ranking, ablation, 3/60-session outcomes, bootstrap intervals, reviewed-label metrics, and promotion evidence remain absent. |
| Phase 4: proposal and opportunity integration | Substantially complete | G139 groups opportunities; G146 adds exact-version lifecycle-gated hypothesis proposals through the existing review ledger plus append-only analyst dispositions. | A governed hypothesis signal materialization adapter, calibrated ranking contribution, and mature live population remain absent. |
| Phase 5: graph and Ask | Foundation only | Existing graph proposals remain review-controlled; G088-G093 provide bounded Ask and evidence-purity controls. | State, transition, hypothesis, opportunity, and outcome graph mappings and Market State Ask bundles are absent. |
| Phase 6: Top 50 bounded rollout | Acquisition and state controls partial | Asset cohorts, bounded ingestion, G142 provider budgets, quality summaries, and G144 explicit state cohorts capped at 10 symbols exist. | Hypothesis/outcome coordination remains AAPL-only; no universe-resolved staged rollout, operational state dashboard, or per-symbol end-to-end readiness exists. |

## Component Conformance

| Component | Current conformance | Critical gap |
| --- | --- | --- |
| Market Feature Pipeline | Partial | G144 adds realized volatility, IV/RV spread, normalized IV/premium/OI changes, curve state, and point-in-time earnings context. Broader surface cells, concentration/migration, weighted liquidity, event producers, and 252-session normalizations remain absent. |
| Market State Builder | Partial | One state-v2 row per explicit asset/session has 69 exact feature references and required-versus-total completeness. CLI cohorts are capped at 10, but universe-resolved analytical rollout and durable jobs are absent. |
| State Transition Engine | Partial | 1/3/5/10/20-session differences, point-in-time rarity/persistence, selected-feature acceleration, curve-slope change, and term-regime transitions persist. Explicit divergence, migration, concentration, curvature, and sign-consistency operators remain incomplete. |
| Evidence Ledger | Partial | Reusable return, OI-ratio, and ATM-IV-change claims exist with lineage. Evidence coverage is too narrow for the target hypothesis pack. |
| Hypothesis Registry | Partial | H001, H004, H006, and H007 v1 are versioned and research-only. H002, H003, and H005 are absent; no audited lifecycle mutation or approved immutability path exists. |
| Hypothesis Evaluator | Partial | Deterministic trigger/non-trigger/rejection rows and reason codes exist; G146 bridges eligible candidate/approved exact versions into reviewed proposals. | Live evaluations remain sparse and no hypothesis signal materialization adapter exists. |
| Opportunity Engine | Partial | Grouping, domain overlap suppression, conflicts, scores, research lifecycle, and append-only analyst dispositions exist. | Calibration contribution and a mature live opportunity population remain absent. |
| Outcome Evaluator | Partial | Immutable 1/5/10/20-session outcomes feed G145 horizon/asset/year/event/term-regime and confidence-band summaries with as-of maturity filtering. Live triggered sources remain too sparse for readiness claims. |
| Data Acquisition | Partial | G142/G143 are analytically bounded and compact. Request IDs, page completeness, selected quotes/contract metadata, quote freshness, selection lineage, and candidate rejection metrics are captured; rate-limit telemetry and durable scheduling are absent. |
| API surface | Read-heavy partial | Core reads, source-aware proposal review/preflight, and opportunity disposition create/list APIs exist. | Feature/state build, transition calculation, hypothesis mutation/evaluation, opportunity build, outcome materialization, coverage, calibration, and durable job-request APIs remain absent. |
| Worker topology | CLI prototype | State materialization combines feature/state/transition/evidence stages; separate hypothesis, opportunity, outcome, and history CLIs exist. Durable job records, partial-failure ledgers, consistent version metrics, and approved scheduling are absent. |
| Frontend | Partial | Asset/options, algorithms, backtests, proposal controls, Syncratic views, and the opportunity queue exist. The specified Market State, DTE-delta surface, transition timeline, hypothesis workbench, and outcome calibration views do not. |
| Syncratic Ask | Partial | Bounded on-demand reasoning and evidence purity exist. Context is still signal/event oriented rather than a compact state/transition/hypothesis/opportunity bundle. |
| Observability | Partial | CLIs emit run summaries, G142 captures acquisition metrics, and G146 dispositions are durable. | Stage-level metrics, latency, incompatible-version counts, proposal rejection rates, and calibration drift remain missing. |
| Security and governance | Strong foundation | Existing auth, proposal review, graph review, research/production separation, source-aware eligibility snapshots, and opportunity dispositions are preserved. | Audited hypothesis promotion and a dedicated hypothesis materialization adapter remain absent. |

## Feature Catalog Reconciliation

Implemented or partially implemented:

- Underlying returns for 1/5/10/20 sessions, RSI 14, SMA-20 distance, volume ratio, gap, and ATR.
- ATM IV at 30/60/90 DTE and 30/60-DTE 25-delta put/call IV, quote-derived 30-DTE wing premium/spread, IV term slopes, 30/60-DTE risk reversal, and selection confidence.
- Put/call OI ratio, aggregate one-session ratio change, dimensioned 30-DTE wing OI change, put/call volume ratio, and transient moneyness/expiry/DTE/delta OI-volume buckets.
- Usable-contract, missing-IV, missing-Greeks, surface-coverage, and OI-quality observations.
- Deterministic source-contract, quote, provider-request, selection-cell, policy-version, and score lineage for seven selected surface cells.
- Annualized 10/20/60-session realized volatility, 5-session 20-D RV change, 30-DTE ATM IV-minus-RV, and IV/RV ratio.
- One/five-eligible-option-session IV changes across all seven selected cells; one/five-session 30-DTE wing premium changes; and five-session wing OI changes.
- Classified 30/60/90 term-structure state plus point-in-time days-to/since earnings and earnings-window state when a normalized event was known by session end.
- One/three/five/ten/twenty-session transitions, selected-feature acceleration, and curve-regime changes; state execution accepts only explicit cohorts capped at 10.

Missing or not yet analytically usable:

- Realized-volatility percentiles, 252-session normalization, and intraday-range percentiles; G144 implements 10/20/60-session levels and 5-session 20-D RV change.
- The broader 21/30/45/60/90/180/365-DTE by 0.15/0.25/0.35/ATM delta surface.
- IV level z-scores, 252-session percentiles/rank, broader skew/surface curvature/dispersion, and 180-DTE slope; G144 implements 1/5-session selected-cell changes, IV/RV comparisons, and 30/60/90 term state.
- Straddles, expected move, broader premium ratios, and premium acceleration beyond G144 30-DTE wing 1/5-session midpoint changes.
- Broader delta/expiry OI-volume changes, concentration, and migration beyond G144 30-DTE wing 1/5-session OI changes and current transient distributions.
- Spread-weighted liquidity and broader standard-contract enforcement; G143 records quote staleness, crossed/zero markets, eligibility reasons, and selection confidence.
- An event-calendar ingestion producer, ex-dividend, corporate-action, and broader point-in-time event context; G144 consumes earnings events when already normalized.

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
| 9 | Proposal review/materialization controls remain mandatory | Pass: G146 reuses review and preflight, and unsupported hypothesis materialization fails closed |
| 10 | Opportunities avoid correlated double counting | Pass in deterministic research logic |
| 11 | Outcomes materialize without source mutation | Pass |
| 12 | Backtests enforce point-in-time correctness | Pass for G145 hypothesis modes: exact versions, as-of outcome maturity, point-in-time segments, and chronological folds are tested |
| 13 | Graph changes remain proposal-based | Pass |
| 14 | Ask receives bounded evidence-pure context | Partial: generic context is bounded; state/opportunity context is absent |
| 15 | Top 50 execution is explicitly bounded | Pass for acquisition and explicit analytical cohorts capped at 10; universe-resolved staged rollout remains absent |
| 16 | Lineage can reproduce any signal proposal | Pass for algorithm and G146 hypothesis proposals through exact source, evaluation, definition version, evidence, and eligibility snapshots |
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

Status: implemented 2026-07-20.

G144 adds 44 definitions and 69 state-v2 slots, retaining the original 39 slots as the required completeness denominator. Realized volatility, IV/RV comparisons, normalized 1/5-session IV and premium change, 5-session wing OI change, curve state, point-in-time earnings context, multi-window differences, selected-feature acceleration, and curve-regime transitions are persisted with explicit missingness and lineage. H006 retains its evaluator-level premium/price divergence classification, and H001 now invalidates known earnings windows. The materializer accepts only explicit cohorts capped at 10 and adds no scheduling, universe resolution, provider acquisition, or history synthesis.

### G145: Hypothesis Backtest And Calibration

Status: implemented 2026-07-20.

G145 reuses existing isolated back-test runs and calibration summaries for exact hypothesis/outcome versions. It adds single, comparison, and expanding walk-forward modes; horizon/asset/year/event/term-regime segmentation; confidence-band precision and Brier error; as-of maturity guards; hard read/cohort bounds; and explicit sample warnings. Reports set `promotion_allowed=false` and do not change lifecycle or deployment state.

### G146: Hypothesis Proposal And Opportunity Governance

Status: implemented 2026-07-20.

G146 adds exact-version candidate/approved hypothesis proposals to the existing reviewed proposal ledger without false algorithm lineage. Candidate and policy-blocked approved rows remain research-only; hypothesis materialization is explicitly unsupported and fail-closed until a dedicated adapter exists. Append-only opportunity dispositions preserve computed lifecycle separately. The live validation correctly built zero proposals from 24 non-triggered/ineligible evaluations.

### G147: Market State Analyst Experience

Status: implemented and accepted 2026-07-20.

Implemented the asset/date state view, persisted seven-cell DTE-delta surface, bounded multi-session transition timeline, exact-version hypothesis/evidence/calibration workbench, and calibrated opportunity detail. Provider acquisition, research calculation, proposal review, signal materialization, graph review, and opportunity disposition remain distinct controls.

Implementation record: `../gates/G147_market_state_analyst_experience_ui_spec.md` and `../../../../frontend/marketops_market_state_analyst_experience_spec.md`.

### G148: Graph, Ask, And Cohort Rollout

Status: proposed; backend implementation specification ready 2026-07-20.

Implement three ordered slices: generalize the existing graph review ledger for non-signal Market State records; extend existing Syncratic contexts and Ask with exact state/session/version evidence purity; then stage explicit cohorts capped at 10 symbols with durable aggregate readiness. Provider acquisition, graph decisions, Ask, proposal review, lifecycle promotion, and unsupported hypothesis materialization remain separate controls.

Implementation handoff: `../gates/G148_graph_ask_and_cohort_rollout.md`.

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

MarketOps is beyond a prototype database: it has genuine deterministic state, longitudinal features, hypothesis, opportunity, outcome, and version-isolated calibration infrastructure with strong governance. It is not yet a useful complete market-state intelligence product because genuine longitudinal options/event coverage and statistically meaningful calibration evidence remain incomplete.

G147 now makes the governed state, transition, hypothesis, calibration, opportunity, and disposition ledgers usable without manufacturing readiness. The definitive next product gate is G148 Graph, Ask, and Cohort Rollout; live research remains coverage-blocked until genuine prospective option sessions accumulate.

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
- G144 feature and transition completion: `../gates/G144_market_feature_and_transition_completion.md`
- G145 hypothesis backtest and calibration: `../gates/G145_hypothesis_backtest_and_calibration.md`
- State feature materializer: `../../../../../internal/marketops/state/materializer.go`
- Hypothesis feature contract: `../../../../../internal/marketops/hypotheses/registry.go`
