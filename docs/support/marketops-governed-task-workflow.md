# MarketOps Governed Task Workflow

## Purpose

MarketOps scheduled work is controlled as durable, per-session task state rather than a single shell exit code. The task ledger records each asset's outcome, provider failure class, attempts, next retry time, and safe operator-facing reason. The Administration Workbench exposes incomplete work under **MarketOps task control**.

## Failure classification

| Condition | Task state | Behavior | Operator action |
|---|---|---|---|
| Vendor access or entitlement denial (`401`, `403`) | `blocked_entitlement` | No retry. Independent assets and dependent covered work continue. | Confirm the provider plan/entitlement and rerun the affected scope after access is granted. |
| FMP paid-plan or rate limit (`402`, `429`) | `deferred_quota` | No further same-window financial calls. FMP poll state records the HTTP status and next eligible time. | Wait for the next budget window or increase plan capacity. |
| Massive retryable rate limit, timeout, network failure, or `5xx` | `retry_scheduled` | The task retries its provider call twice in-process, then is persisted for the governed retry worker. | Review only if retries exhaust. |
| Valid but unavailable source data (`404`, no completed technical series) | `skipped_no_data` | No automatic retry; dependent work is withheld for that asset. | Reconcile after the provider publishes the completed session. |
| Invalid payload, malformed request, or retries exhausted | `failed_terminal` | The asset is excluded from dependent work and the workflow is degraded. | Inspect the recorded reason and correct configuration or code. |

## Tactical Posture

Tactical Posture requires completed-session `return_5d` and `distance_sma_50_pct` from Market State plus same-session Massive RSI-14, SMA-200, and 21 SMA-50 observations. A missing prerequisite is recorded explicitly; it is never rendered as a generic processing delay.

The post-close run processes every asset independently. A post-close stage wrapper treats a non-zero tactical process as recoverable only when the durable task ledger contains a classified outcome for every active asset; otherwise the job remains failed. Retry-only invocations roll the parent workflow up from all persisted session tasks, so they cannot erase an earlier skipped, deferred, blocked, or terminal state. A provider failure for one symbol does not terminate the remaining universe. Due `retry_scheduled` Tactical Posture tasks are selected by `scripts/marketops_tactical_retry.sh` and rerun by the `signalops-marketops-task-retry` user timer every 15 minutes during the post-close retry window.

A newly onboarded analyst asset remains part of the unified `all_active` universe for quote and EOD coverage immediately. It becomes part of `all_workflow_ready` only after a successful 50-session normalized equity backfill; strategic workflows select that readiness scope. Provider-bound options and intraday work are executed in deterministic batches, so no fixed universe cap silently omits an active asset.

## Workflow semantics

A MarketOps workflow is `succeeded` only when all task items complete. It is `degraded` when blocked, skipped, deferred, or terminal task items remain. Downstream algorithms run only for assets that have their prerequisite evidence; incomplete assets are not represented as current tactical evidence.

## Operations

1. Inspect **Administration → System → MarketOps task control** for the symbol, failure class, and retry time.
2. Do not retry entitlement-blocked or no-data items until the external condition changes.
3. Allow transient retries to run before manual action.
4. Re-run a specific asset with `marketops-tactical-valuation-runner --session-date YYYY-MM-DD --symbols SYMBOL` after correcting a terminal condition.
5. Verify that the task row is `succeeded` and Tactical Posture is no longer unavailable in Valuation & DOSM.
