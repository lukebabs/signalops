# G147 Market State Analyst Experience UI Specification

Status: proposed - frontend-agent specification ready

Date: 2026-07-20

## Purpose

G147 hands the frontend agent an implementation-ready contract for making the G137-G146 state, transition, hypothesis, calibration, opportunity, and disposition ledgers usable in the existing MarketOps application.

## Specification

Frontend-agent specification:

- `../../../../frontend/marketops_market_state_analyst_experience_spec.md`

## Scope

The frontend should:

- add one MarketOps-only `/marketops/state` asset/date workbench;
- provide overview, seven-cell DTE/delta surface, material transition, and hypothesis tabs;
- compare the selected state with the nearest earlier persisted session using exact feature identity;
- show current evaluation evidence and exact-version G145 calibration without implying promotion readiness;
- extend the existing opportunity detail with calibration, evidence-quality limits, and append-only G146 analyst dispositions;
- keep URL selection, bounded reads, explicit missingness, responsive behavior, and accessible non-color semantics;
- retain provider acquisition, state construction, research evaluation, lifecycle promotion, proposal review, signal materialization, graph review, and opportunity disposition as distinct controls.

## Backend Readiness

The current backend contract is sufficient for the bounded read experience and opportunity disposition action. G147 composes the existing state/lineage, transition, definition/evaluation, evidence, outcome, calibration-summary, opportunity, and disposition APIs. It must not add a frontend-only proxy, invent a composite endpoint, or reinterpret sparse data as readiness.

Current evidence may legitimately produce incomplete states, no transitions, non-triggered/ineligible evaluations, insufficient calibration samples, no hypothesis proposals, and no opportunities. These are required first-class states.

## Explicitly Out Of Scope

- New backend endpoints.
- Provider acquisition, scheduling, historical synthesis, or broad cohort rollout.
- Browser-triggered state, feature, evaluation, opportunity, or outcome calculation.
- Hypothesis promotion or automatic calibration decisions.
- Proposal review or signal materialization inside the Market State route.
- Graph review/mutation, Syncratic Ask, narrative generation, or G148 cohort rollout.
- Trading or portfolio actions.
- Full-chain options visualization beyond the persisted seven selected cells.

## Result

The frontend agent can implement G147 without crossing the G145 advisory-calibration boundary or the G146 reviewed-proposal, fail-closed materialization, and append-only disposition boundaries.

## Links

- Canonical architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- Architecture reconciliation: `../architecture/market_state_intelligence_evaluation.md`
- Existing opportunities workbench: `../../../../frontend/marketops_opportunities_workbench_spec.md`
- G145 calibration: `G145_hypothesis_backtest_and_calibration.md`
- G146 governance: `G146_hypothesis_proposal_and_opportunity_governance.md`
