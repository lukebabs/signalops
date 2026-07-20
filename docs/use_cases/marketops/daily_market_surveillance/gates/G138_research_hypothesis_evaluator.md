# G138 Research Hypothesis Evaluator

Status: implemented and live-validated on 2026-07-20.

## Purpose

G138 adds the first governed MarketOps hypothesis registry and deterministic evaluator over the G137 state substrate. It records why each hypothesis was eligible, triggered, not triggered, or rejected without converting research results into operational signals.

## Research Pack

- H001 v1: overbought downside-hedging expansion.
- H004 v1: volatility term-structure regime shift.
- H006 v1: premium-price divergence.
- H007 v1: delta-bucket unusual open-interest accumulation.

All four definitions have lifecycle status `research`. Their calibration policy sets `production_materialization_allowed=false` and requires at least 100 samples plus walk-forward validation before promotion can be considered.

## Implemented Boundary

- Tenant-aware, versioned hypothesis-definition ledger.
- Idempotent hypothesis-evaluation ledger linked to one canonical market state.
- Deterministic evaluation IDs and keys independent of run ID.
- Explicit eligibility, trigger, score, evidence, and reason-code fields.
- Persistence of both eligible and rejected/non-triggered evaluations.
- Bounded AAPL-only CLI with inclusive session dates, a maximum-session cap, and dry-run mode.
- Read-only definition and evaluation APIs.
- Positive unit fixtures for all four hypotheses and negative sparse-input coverage.

G138 does not write algorithm proposals, graph proposals, opportunities, alerts, insights, or production signals. It does not call Massive, schedule work, or promote a hypothesis lifecycle status.

## Live Result

The bounded AAPL window from 2026-07-01 through 2026-07-20 contained six G137 states. Each run evaluated four hypotheses per state:

- definitions: 4;
- evaluations: 24;
- eligible: 0;
- triggered: 0;
- rejected: 24.

The result is expected. State completeness was 0-24%, quality was `missing` or `partial`, and the required IV, premium, OI-change, RSI, surface-coverage, and transition-persistence inputs were absent or unusable. The evaluator retained those causes as reason codes rather than treating missing values as zero or manufacturing a trigger.

Two write runs with different run IDs left exactly 4 definitions and 24 evaluations, proving logical idempotency.

## Acceptance

- Migration `000029_marketops_hypothesis_research` applies cleanly.
- Registry contents are exactly H001, H004, H006, and H007 v1 in `research` status.
- Each state produces one durable evaluation per hypothesis version.
- Incomplete inputs produce explicit rejection reasons and no trigger score.
- Complete positive fixtures trigger each hypothesis deterministically.
- Reruns update run lineage without duplicating logical evaluations.
- Read APIs expose definitions and evaluations without adding a mutation route; authenticated live reads returned 4 definitions and 24 evaluations.
- No production workflow ledger is written by the evaluator.

## Next Gate

G139 should introduce the analyst-facing opportunity layer. It should group compatible, eligible hypothesis evaluations by asset, session, direction, and horizon while preserving research/production lifecycle controls. Sparse or rejected G138 evaluations must not become opportunities.

## Links

- Architecture evaluation: `../architecture/market_state_intelligence_evaluation.md`
- Target architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- G137 vertical slice: `G137_aapl_market_state_vertical_slice.md`
- Operations: `../operations/hypothesis_evaluation.md`
- API contract: `../../../../api.md`
