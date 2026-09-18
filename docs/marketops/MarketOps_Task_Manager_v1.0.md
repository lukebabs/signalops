# MarketOps Task Manager v1.0

The Task Manager is the operator-facing control plane for scheduled MarketOps work. It reads the existing scheduled-job status
ledger, evaluates a versioned dependency contract, and exposes a fail-closed state for each managed job:

- `ready`, `running`, `succeeded`, `deferred`, `blocked`, or `retryable`.
- A retry is offered only for an allow-listed failed job; dependency failures and stale prerequisites remain blocked.
- Retry execution delegates to the existing deployment-agent run-now action, preserving the existing authorization boundary.

The Admin Operations surface is available through `GET /v1/administration/marketops/task-manager` and the Retry action at
`POST /v1/administration/marketops/task-manager/{job_id}/retry`. Both routes require the tenant administrator primitive.

Migration `000177_marketops_task_manager` adds append-only dependency definition, evaluation, and retry-decision tables for the
next persistence slice. The current evaluator is deterministic and continues to operate from the existing status ledger while
those audit rows are integrated into the scheduled workers.

Initial dependency scope covers warm EOD, intraday, post-close, Risk/Reward, SRI, SRI holdings, SAF evaluation, and recovery.
Provider polling behavior and scheduler timing are unchanged by this release.
