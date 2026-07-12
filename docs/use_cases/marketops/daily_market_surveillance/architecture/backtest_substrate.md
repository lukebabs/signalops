# Back-Test Substrate Architecture

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Purpose

The MarketOps back-test substrate should let operators evaluate DSM detector output and graph-proposal review policy over historical data without affecting production operational records.

This is separate from the existing SignalOps replay subsystem (replay API at `/v1/replay`; `replay_jobs` and `replay_worker_heartbeats` tables). Replay republishes historical ledger payloads through broker topics for operational recovery, compatibility checks, or pipeline validation. Back-testing should execute a bounded experiment, isolate generated outputs, and produce comparable metrics for policy calibration.

## Why Not Use Operational Replay Alone

Operational replay is durable and useful, but it is not sufficient as the back-test substrate:

- Replay publishes back into operational topics.
- Replay is designed around source payload reproduction, not experiment isolation.
- Replay results focus on published/scanned/failed counts.
- Back-tests need run-scoped generated signals, artifacts, proposals, and policy metrics.
- Back-tests must compare detector or policy versions without confusing operators with production records.

The back-test substrate may reuse replay concepts such as windows, max records, source filters, and worker status, but it needs separate storage and run semantics.

## Architecture Boundary

The proposed G082 implementation should use this flow:

```text
normalized_event_ledger
  -> MarketOps DSM detector
  -> DSM signal artifact extraction
  -> graph proposal extraction
  -> deterministic policy evaluator
  -> isolated back-test ledgers
  -> aggregate run metrics
```

The back-test runner must not publish generated records to production Kafka topics by default. It must not persist generated outputs into production ledgers.

## Isolation Rules

Back-test output must be keyed by a back-test run id and stored separately from production tables.

Do not write back-test generated outputs to:

- `signal_ledger`
- `alert_ledger`
- `insight_ledger`
- `marketops_dsm_artifacts`
- `marketops_dsm_graph_proposals`

Do not materialize graph writes from a back-test run. Accepted proposals from G080 are review labels and possible training/evaluation data; they are not automatic graph-write permission.

## Conceptual Storage Model

Future implementation should add a small isolated storage boundary:

- `marketops_backtest_runs`
- `marketops_backtest_signals`
- `marketops_backtest_artifacts`
- `marketops_backtest_graph_proposals`
- `marketops_backtest_policy_results`

Aggregate run metrics are stored on the `marketops_backtest_runs` row rather than a separate metrics table.

The exact schema should be decided in G082, but the interface must preserve:

- run identity
- source window and filters
- detector identity/version
- policy identity/version
- generated output payloads
- aggregate metrics
- error and skip accounting

## Determinism

Back-tests should be reproducible for the same:

- normalized input set
- detector version
- policy version
- run options

Generated ids should include the back-test run id or a deterministic run namespace so that repeated runs do not collide with production ids. This follows the same deterministic-id philosophy as the G079 `proposal_id` convention, where replaying the same source data is idempotent.

## Policy Calibration

The first policy evaluator should be deterministic and explainable.

It should classify generated graph proposals into:

- `auto_accept_candidate`
- `auto_reject_candidate`
- `manual_review_required`
- `supersede_candidate`

The evaluator should emit reasons for each classification. These reasons are part of the back-test output and should be available for audit and future UI work.

## Relationship To G080 Decisions

G080 operator decisions are useful label data, but they are not perfect truth:

- An accepted proposal means an operator approved the proposal in context.
- A rejected proposal means an operator did not want that graph fact materialized.
- A superseded proposal means a newer or better graph fact replaced the older candidate.
- A restored proposal means the proposal needs review again.

Back-tests should use G080 labels as evaluation data only after enough decisions exist and after label policy is documented.

## Initial Success Criteria

A thin MVP is successful when it can:

- read a bounded historical normalized-event window
- run MarketOps DSM logic in an isolated context
- generate run-scoped signal/artifact/proposal outputs
- classify proposals with a deterministic policy pack
- persist aggregate metrics
- prove no operational ledgers were mutated

## Documentation Links

- Gate note: `../gates/G081_backtest_substrate.md`
- Operations: `../operations/backtest_substrate.md`
- G079 graph proposal acceptance (deterministic proposal id precedent and label data source): `graph_proposal_acceptance.md`
- G080 operator review workflow: `../gates/G080_operator_graph_proposal_review.md`
- Current signal and artifact ledger semantics: `signal_artifact_persistence.md`
- Replay API contract: `../../../../api.md`
