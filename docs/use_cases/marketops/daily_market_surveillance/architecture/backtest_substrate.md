# Back-Test Substrate Architecture

Status: MVP implemented
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

The implemented G081 MVP uses this flow:

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

G081 adds this isolated storage boundary:

- `marketops_backtest_runs`
- `marketops_backtest_signals`
- `marketops_backtest_artifacts`
- `marketops_backtest_graph_proposals`
- `marketops_backtest_policy_results`

Aggregate run metrics are stored on the `marketops_backtest_runs` row rather than a separate metrics table.

The MVP schema preserves:

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

Back-test rows are keyed by `run_id` plus generated object ids, so repeated runs remain isolated from production rows and from each other. This follows the same deterministic-id philosophy as the G079 `proposal_id` convention, while preserving experiment-level isolation.

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


## Implemented Interfaces

- Operator command: `cmd/marketops-backtest`.
- Detector adapter: `python/signalops_workers/backtest_detector.py`.
- Read-only APIs: `/v1/marketops/backtests`, `/v1/marketops/backtests/{run_id}`, `/v1/marketops/backtests/{run_id}/signals`, and `/v1/marketops/backtests/{run_id}/graph-proposals`.
- Reusable DSM helpers: `internal/marketops/dsm` for artifact extraction, graph proposal extraction, and deterministic policy classification.

## G082 Calibration Summary Snapshots

G082 adds `marketops_backtest_calibration_summaries` as a persisted aggregate layer over isolated back-test runs.

A summary snapshot is created explicitly from a run filter and stores the selected run ids, aggregate run metrics, recommendation distribution, dominant recommendation, and the filter/parameter payload used to create the snapshot.

This is intentionally separate from detector execution and production ledgers:

- back-test runs still own generated experimental outputs;
- calibration summaries aggregate completed run metrics;
- production signal, artifact, graph proposal, alert, and insight rows are not mutated;
- policy promotion and named baseline management remain future gate work.

## G083 Baselines And Evaluation Direction

G083 adds named calibration baselines and stored comparison records over the G082 persisted summary snapshots.

The baseline layer treats calibration summaries as immutable evidence. A baseline points to a selected summary; a comparison captures deterministic deltas between a candidate summary and the baseline summary.

G080 operator graph-proposal decisions can become evaluation labels only after they are normalized into a separate label substrate with a stable graph fact key. Accepted, rejected, superseded, proposed, and restored states should not be collapsed into binary truth without documented rules.

Policy promotion remains deferred. Baseline comparisons can emit recommendations such as `needs_more_data`, `regression_candidate`, `improvement_candidate`, `neutral_candidate`, or `manual_review_required`, but those recommendations are advisory only.

## G086 Promotion Review Direction

G086 should introduce an auditable review layer above G083 baseline comparisons and G085 label-aware evaluations.

A promotion candidate should reference immutable comparison and evaluation evidence, compute conservative readiness status, and capture an operator decision. Approval in this layer is permission to plan a later deployment gate; it is not a runtime policy change, detector threshold edit, graph writeback, or model release.

## G087 Deployment Planning Direction

G087 should add deployment plan records above approved G086 promotion candidates. A deployment plan describes target type, environment, rollout strategy, preflight checks, rollback plan, and evidence. It should not execute a deployment or mutate runtime detector, policy, graph, signal, alert, or insight state.
