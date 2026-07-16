# G124 Algorithm Materialized Signal Lifecycle Policy

Status: proposed - lifecycle decision recorded
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G124 records the lifecycle decision for production signals created from reviewed `algorithm_signal_proposals` by the G122 materialization endpoint.

G122 intentionally writes one production signal ledger row through `UpsertSignalLedger` only. It does not directly create alerts, insights, graph proposals, Syncratic context windows, or downstream lifecycle records.

## Decision

Keep algorithm materialization signal-ledger-only for the current gate sequence.

Do not call alert, insight, graph, or Syncratic lifecycle writers directly from the G122 POST route.

The next runtime lifecycle step should be a separate policy gate that consumes materialized production signals and decides whether they should enter alert, insight, graph, or reasoning workflows.

## Rationale

The materialization action is an operator-confirmed bridge from algorithm output into canonical `signal.v1` storage. It should remain narrow, auditable, and idempotent.

Directly invoking lifecycle fanout from the mutation route would couple proposal review, signal creation, alert semantics, insight semantics, graph semantics, and reasoning semantics into one endpoint. That would make retries, duplicate handling, severity policy, and operator trust harder to reason about.

A separate lifecycle policy lets SignalOps distinguish these concepts cleanly:

- `algorithm_signal_proposals`: candidate signal intent from algorithm results.
- `algorithm_signal_materializations`: audit ledger for operator-confirmed materialization attempts.
- `signals`: canonical production signal evidence.
- `alerts`: event-level operator notification records.
- `insights`: synthesized multi-event or multi-signal interpretation records.
- `graph proposals`: reviewed entity/relation writeback candidates.
- `Syncratic Ask`: on-demand reasoning over curated context windows.

## Recommended Future Runtime Path

When this moves beyond design, prefer a lifecycle policy processor over direct fanout inside the materialization endpoint.

Recommended behavior:

1. Select canonical signals where:
   - `source_id=algorithm_signal_materialization`;
   - `domain=algorithms`;
   - `use_case=algorithm_signal_materialization`;
   - lifecycle policy has not already processed the signal id.
2. Apply a versioned lifecycle policy such as `algorithm_signal_lifecycle.v1`.
3. Persist lifecycle decisions in a first-class audit row before creating downstream records.
4. Create alerts only for event-level operator notification criteria.
5. Create insights only when the policy sees enough related evidence across more than one signal/event or an explicitly configured single-signal insight rule.
6. Create graph proposals only through the existing reviewed graph proposal workflow.
7. Trigger Syncratic Ask only on demand or under an explicit bounded reasoning policy, never as unbounded batch fanout.

## Policy Inputs

A future lifecycle policy should consider:

- signal id, type, severity, confidence, and signal time;
- algorithm id, algorithm version, execution request id, algorithm result id, and proposal id;
- materialization id, requested by, requested at, policy version, and idempotency key;
- source event ids and artifact ids;
- duplicate status and duplicate signal id;
- existing alerts, insights, and graph proposals for the same signal/entity/window;
- operator review status and decision metadata;
- tenant-level lifecycle configuration.

## Required Guarantees

A future lifecycle implementation must preserve:

- idempotency by signal id and lifecycle policy version;
- no duplicate alert/insight/graph creation on retry;
- clear audit trail from algorithm result to proposal to materialization to lifecycle decision;
- explicit policy versioning;
- operator/admin authorization for manual lifecycle actions;
- no secret, bearer token, prompt, or raw upstream payload leakage in UI/API errors.

## Rejected Alternatives

### Direct fanout in G122

Rejected for now. It would make the materialization endpoint too broad and would blur the line between producing a canonical signal and deciding operational consequences.

### Treat every materialized signal as an alert and an insight

Rejected. Alerts should represent event-level operator notification. Insights should represent synthesized interpretation across enough supporting context, unless a future policy explicitly defines a high-confidence single-signal insight class.

### Skip lifecycle audit rows

Rejected. Algorithm-generated production state needs a durable policy trail so operators can explain why a materialized signal did or did not become an alert, insight, graph proposal, or reasoning context.

## Out Of Scope

- Implementing lifecycle tables or worker code.
- Changing G122 materialization behavior.
- Bulk materialization.
- Runtime policy deployment.
- Alert/insight/graph writeback.
- Syncratic integration changes.
- Frontend implementation.

## Acceptance Criteria For A Future Implementation Gate

A later implementation gate should not start until it can define:

- the lifecycle decision storage schema;
- the policy version and decision outcomes;
- the processor trigger model;
- duplicate prevention by signal id and policy version;
- alert versus insight criteria;
- graph proposal criteria;
- UI read model for lifecycle decisions;
- positive and negative tests for retry, duplicate signal, low severity, and insufficient context.

## Result

G124 keeps G122 narrow and production-safe while defining the recommended lifecycle boundary for materialized algorithm signals.
