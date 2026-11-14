# MarketOps Dagster Orchestration Migration v1.0

## Objective

Replace independent time-based MarketOps CronJobs with a dependency-aware Dagster control
plane. Kubernetes remains the execution substrate and the MarketOps database remains the
system of record for asset-list revisions, task evidence, feature watermarks, and
projections.

Dagster must orchestrate existing workers first; it must not duplicate algorithm or
provider logic. The initial migration is therefore an orchestration migration, not a
business-logic rewrite.

## Current failure this addresses

Risk/Reward currently starts at a fixed time after post-close starts, rather than after
post-close feature materialization completes. On 2026-10-14 the post-close writer ran
21:35–22:05 UTC while Risk/Reward ran 21:45–21:47 UTC. It observed 81/225 ready assets,
created 144 unavailable snapshots, and exited successfully through the bounded partial
grace path. The remaining features arrived after Risk/Reward had already finished.

The durable invariant is:

> A downstream projection may not publish until the current primary asset-list revision
> has complete, usable dependency evidence for every member.

## Target graph

The first graph is session-scoped and keyed by `(tenant_id, session_date,
asset_list_revision)`:

```text
market-session-open
        |
provider-capture (Massive/FMP, bounded and idempotent)
        |
normalization-and-EOD-evidence
        |
postclose-feature-materialization
        |
feature-watermark-gate (225/225, current list revision)
        +--> market-state
        +--> risk-reward
        +--> valuation / DOSM
        +--> EROC
        +--> EEOM
        +--> SRI / holdings
        +--> options projections
        |
algorithm-convergence-gate
        |
global-dashboard-projection
        |
SAF outcome/effectiveness materialization
        |
operations-monitor and notification
```

Each node has an explicit input watermark, output watermark, run id, image digest,
asset-list id/revision, and failure class. A node is `blocked` when its upstream
watermark is absent or incomplete; it is never marked `succeeded` with a partial cohort.

## Dagster deployment

Deploy Dagster into the `signalops-orchestration` namespace:

- Dagster webserver and daemon (HA can be added later; run state is persisted).
- Dagster PostgreSQL metadata database, separate from MarketOps data databases.
- Kubernetes run launcher, creating isolated worker Jobs in `signalops-marketops`.
- OpenBao-backed runtime credentials, read-only to Dagster and scoped worker secrets to
  the launched Job.
- Istio route restricted to the Admin Operations surface; no public user route.

Dagster metadata is operational observability only. MarketOps evidence and projections
remain in the MarketOps database.

## Required guards

1. **Dynamic asset-list snapshot**: every run reads and pins the current list id,
   revision, fingerprint, and expected member count before launching work.
2. **Watermark gate**: downstream assets query completion for the same session and list
   revision. Time-based sleeps are not dependencies.
3. **Fail-closed partial handling**: remove the Risk/Reward partial-grace success path.
   If the watermark is incomplete by the run deadline, the Dagster run is `blocked` or
   `retryable`; no unavailable rows are published as a successful projection.
4. **Idempotency**: all worker invocations use deterministic run/correlation ids and
   upserts, so retries cannot duplicate evidence.
5. **Single image contract**: each graph run records and verifies one immutable worker
   image digest across post-close, algorithms, and projection.
6. **Retry policy**: retries are bounded and dependency-aware. A failed provider call
   retries its asset only; a dependency failure retries the dependency before any
   downstream stage.
7. **Completeness assertion**: projection requires `projected_rows == expected_rows ==
   usable_rows` for the pinned list revision. Otherwise it remains blocked and alerts.

## Observability

The Admin Task Manager should display Dagster run state alongside the existing task
ledger:

- graph/run id and session date;
- current list revision and expected/ready/projected counts;
- upstream watermark and lag;
- active worker image digest;
- retry number and next retry time;
- failure class and the exact blocked dependency;
- links to logs, task ledger rows, and projection evidence.

The operations monitor should alert on:

- any blocked critical node past its SLA;
- list revision mismatch;
- partial cohort evidence;
- a projection attempt before its dependency watermark;
- a worker image mismatch within one graph;
- repeated retries or provider-budget exhaustion.

## Migration phases

### Phase 0 — contract and shadow graph

- Add Dagster definitions for the graph above.
- Read the existing task ledger and database watermarks.
- Launch no production workers; compare Dagster decisions with CronJob outcomes.
- Validate the 2026-10-14 failure reproduces as `blocked`, not `succeeded`.

### Phase 1 — supervised execution

- Run one approved post-close session through Dagster with provider polling disabled unless
  explicitly approved.
- Keep CronJobs suspended for that session.
- Verify all evidence, list revision, and projection invariants.

### Phase 2 — scheduler handoff

- Disable equivalent production CronJobs.
- Enable Dagster schedules/sensors.
- Retain the deployment-agent rollback action and a one-command pause switch.

### Phase 3 — stabilization

- Remove obsolete time-based dependency workarounds and duplicate recovery paths.
- Keep the existing task ledger as the application-facing audit contract, populated by
  Dagster run metadata and worker results.

## Rollback

Rollback is a control-plane action, not a data restore: pause Dagster schedules, resume the
previously pinned CronJobs, and keep all evidence written by either path. No database or
Docker-volume deletion is part of rollback.

## Acceptance criteria

- A downstream job cannot start with an incomplete upstream watermark.
- A partial cohort cannot be reported as a successful projection.
- A retry resumes from the failed dependency, not from the beginning of the graph.
- Admins can identify the exact blocked asset/dependency without reading pod logs.
- A completed session has one immutable list revision, one graph run id, and complete
  225/225 evidence before dashboard publication.

## Automatic failure reconciliation

Scheduled MarketOps runs now carry `marketops_job_id`, `marketops_scheduler=dagster`,
and `marketops_reconciliation_attempt` tags. The Dagster failure sensor watches only
MarketOps scheduled runs and dispatches one bounded retry through the existing
Kubernetes Job template. A retry is tagged with `marketops_reconciliation_attempt=1`;
the sensor skips any subsequent failure for that run, preventing retry loops.

This is deliberately separate from task-level retries. The worker remains responsible
for per-asset continuation and dependency guards, while Dagster handles a failed
workflow launch. Missed sessions still require an explicit reconciliation run because
Dagster does not infer historical schedule ticks after an outage.

### Session identity hardening

Every scheduled Dagster run now stamps `marketops_session_date` from its
scheduled execution time converted to `America/New_York`. The launcher passes
that exact value to the child Kubernetes Job as `MARKETOPS_SESSION_DATE`.
Manual retries can provide `session_date` explicitly. This prevents UTC
midnight and delayed-worker drift from moving a post-close job onto a different
trading session. The dashboard projection remains the final publish gate and
requires a ready watermark and complete Risk/Reward coverage for that same
session.

### Degraded dependency exception

The dependency watermark distinguishes a blocked session from a complete-but-degraded
session. A projection may publish when every asset in the authoritative list has a
persisted Risk/Reward row, including an explicit `unavailable` row for an asset whose
provider or technical inputs were unavailable. The watermark is recorded as `degraded`,
with excluded symbols remaining visible for retry and operations monitoring. A missing
row still blocks publication; this prevents silent omissions while ensuring one
unavailable asset cannot stall the entire dashboard.
