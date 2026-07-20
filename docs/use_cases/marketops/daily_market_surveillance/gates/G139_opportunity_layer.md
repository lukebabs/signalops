# G139 Opportunity Layer

Status: implemented, live-validated, and frontend-deployed on 2026-07-20.

## Purpose

G139 adds the first analyst-facing MarketOps opportunity object. It groups compatible triggered hypothesis evaluations by asset, session, resolved direction, and horizon while retaining hypothesis contributions, evidence lineage, overlap suppression, conflicts, scores, and research status.

## Implemented Boundary

- Tenant-aware `marketops_opportunities` ledger with deterministic versioned identities.
- AAPL-bounded `signalops-marketops-opportunity-builder` with inclusive dates, a 50-session cap, explicit run lineage, and dry-run mode.
- Contribution grouping by asset, session, direction, and `5_to_20_sessions` v1 horizon.
- Strongest-evaluation selection within one hypothesis domain to avoid correlated double counting.
- Opposing upside/downside evaluations retained as conflicts instead of silently discarded.
- Opportunity, confidence, domain-diversity, and conflict scores bounded to 0-1.
- `emerging` lifecycle for a single independent contribution and `active` for at least two independent domains without dominant conflict.
- Concise deterministic summary plus linked evaluation, evidence, suppressed-overlap, conflict, and invalidation references.
- Read-only list/detail APIs.
- G138 evaluation payload compatibility fields for resolved direction, horizon, and hypothesis family.
- UX-first analyst workbench with a dense queue, immediate detail, lazy evidence drill-down, and sparse-data diagnostics.

Every G139 row is `research_only=true`. Lifecycle `active` means independently corroborated within the research queue; it does not mean approved, tradable, or materialized.

## Scoring Semantics

The v1 score combines the strongest trigger, mean quality, mean persistence, domain diversity, and corroborating contribution count, then subtracts an opposing-direction conflict penalty. Only the strongest member of each hypothesis domain receives full contribution.

Confidence represents confidence that the grouped market-state evidence satisfies its hypotheses. It is not a forecast probability or expected return.

## Live Result

The bounded AAPL window from 2026-07-01 through 2026-07-20 contains 24 G138 evaluations:

- eligible: 0;
- triggered: 0;
- ineligible/skipped: 24;
- opportunities: 0.

Dry-run and two write runs produced the same zero-opportunity result. The empty ledger is expected and proves that sparse or rejected hypothesis evaluations do not become analyst-facing opportunities.

The G138 compatibility refresh retained all 24 deterministic evaluations and added the shared `5_to_20_sessions` horizon. H001 resolves to downside and H004 to non-directional; H006/H007 remain conditional until required trigger evidence resolves their direction.

## Safety Boundary

G139 does not add:

- opportunity review or lifecycle mutation APIs;
- signal, proposal, alert, insight, artifact, or graph writes;
- trade or portfolio actions;
- Syncratic Ask or generated opportunity narratives;
- provider acquisition, Top 50 fanout, or scheduling;
- browser-triggered opportunity builds;
- hypothesis promotion or runtime policy deployment.

## Frontend Workbench

The MarketOps-only `/marketops/opportunities` route implements a dense work queue with immediate detail, contribution/conflict/evidence drill-down, research-only visibility, and a useful empty-queue diagnostic based on the existing hypothesis-evaluation API. It does not expose review, build, trade, or materialization controls.

The current live ledger has no opportunity rows, so the primary rendered path reports the bounded supporting evaluation counts and rejection reasons. This preserves the distinction between no source evaluations and evaluations blocked by coverage or quality.

Specification: `../../../../frontend/marketops_opportunities_workbench_spec.md`.

## Next Gate

G140 should materialize forward outcomes after forecast horizons mature. Outcome rows must remain separate from evaluations and opportunities and must preserve point-in-time lineage without changing the original research record.

## Links

- Target architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- Architecture evaluation: `../architecture/market_state_intelligence_evaluation.md`
- G138 evaluator: `G138_research_hypothesis_evaluator.md`
- Operations: `../operations/opportunity_building.md`
- API contract: `../../../../api.md`
