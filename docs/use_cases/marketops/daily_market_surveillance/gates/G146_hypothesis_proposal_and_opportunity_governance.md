# G146 Hypothesis Proposal And Opportunity Governance

Status: implemented and accepted backend governance contract on 2026-07-20.

## Objective

Bridge only eligible hypothesis lifecycle versions into the existing reviewed signal-proposal workflow, preserve a fail-closed materialization boundary, and add auditable analyst dispositions without conflating them with computed opportunity lifecycle.

## Proposal bridge

`signalops-marketops-hypothesis-proposal-generator` reads exact-version hypothesis definitions and evaluations for a bounded AAPL date range. It creates a proposal only when the evaluation is eligible, triggered, non-invalidated, scored, and joined to an exact `candidate` or `approved` definition.

Migration `000034` generalizes the existing `algorithm_signal_proposals` ledger with an explicit `proposal_source` while preserving algorithm-result rows. Hypothesis rows carry the exact evaluation, key, version, lifecycle, evidence references, research flag, materialization eligibility, and an immutable eligibility snapshot. They do not invent algorithm result or execution identifiers.

| Definition state | Proposal behavior |
| --- | --- |
| `draft`, `research`, `backtest_ready`, `calibration`, `paused`, `retired` | No proposal |
| `candidate` | Reviewable research-only proposal; never materialization-eligible |
| `approved` without approval actor/time | No proposal |
| `approved` with audit but production policy false | Reviewable research-only proposal |
| `approved` with audit and explicit production policy true | Reviewable proposal marked materialization-eligible |

All created rows begin as `proposed` and reuse the existing `proposed`, `reviewed`, `rejected`, and `superseded` decision workflow. Existing proposal list, detail, summary, and preflight endpoints expose source and hypothesis filters.

G146 does not write a signal. The existing algorithm materializer rejects a hypothesis source with HTTP `409 unsupported_materialization_source`, and preflight reports `hypothesis_materialization_adapter_unavailable`. This prevents a materialization-eligible flag from bypassing the requirement for a dedicated, reviewed hypothesis signal adapter.

## Opportunity disposition

`marketops_opportunity_dispositions` is an append-only analyst audit ledger. The allowed dispositions are `watch`, `advance`, `needs_more_evidence`, `dismiss`, and `resolved`. Each row records actor, note, metadata, opportunity, tenant, and time.

- `GET /v1/marketops/opportunities/{opportunity_id}/dispositions`
- `POST /v1/marketops/opportunities/{opportunity_id}/dispositions`

Disposition does not rewrite `marketops_opportunities.lifecycle_status`. Computed lifecycle and analyst judgment remain separate, and no alert or insight row is synthesized.

## Validation and observed effectiveness

Focused proposal, PostgreSQL, API, and CLI tests pass. Tests cover exact-version joins, lifecycle admission, approval audit, production-policy gating, deterministic identity, false algorithm-lineage rejection, review visibility, fail-closed materialization, and disposition validation/create/list behavior. Migration `000034` applied to local PostgreSQL.

The real G146 run scanned 24 persisted AAPL evaluations and built zero proposals because all 24 were non-triggered or ineligible. All four registered definitions remain `research`. This is the correct operational result: G146 is effective as a governance boundary, but there is not yet real eligible evidence with which to demonstrate a hypothesis proposal or analyst disposition.

## Boundaries retained

G146 does not promote definitions, infer approval from G145 reports, manufacture triggers, lower quality thresholds, call a provider, create production signals, mutate opportunity lifecycle, generate alerts/insights, write graph state, schedule work, or add frontend behavior.

## Next gate

G147 Market State Analyst Experience should build the specified state, surface, transition, hypothesis, and opportunity views over these controls. Proposal review, future hypothesis materialization, graph review, and opportunity disposition must remain distinct actions.
