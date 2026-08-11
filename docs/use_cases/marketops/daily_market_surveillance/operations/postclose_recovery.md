# MarketOps Post-Close Outage Recovery

Status: implemented guardrail; the guard is installed separately from the existing daily timer so it can be reviewed and rolled back independently.

## Objective

A service interruption must not silently leave a completed market session without Risk/Reward, snapshots, or the other evidence required by the universal completion gate.

The primary post-close service remains the authoritative scheduled workflow. The recovery guard is a bounded safety net; it does not create a parallel collection plane or invent a completed state.

## Recovery behavior

The signalops-marketops-postclose-recovery timer runs on weekdays every 15 minutes from 18:30 through 23:00 America/New_York.

For the current completed session it:

1. derives the active canonical universe and runs the existing universal completion gate;
2. records a marketops-risk-reward stage status with current result/snapshot counts;
3. exits without work when the gate has passed;
4. defers when the primary post-close service or its lock is active;
5. otherwise launches the same idempotent post-close workflow for that explicit session date;
6. reruns the completion gate and clears its attempt state only after success.

The guard writes at most two recovery attempts per session by default. Configure MARKETOPS_POSTCLOSE_RECOVERY_MAX_ATTEMPTS only as part of an operational capacity decision. Exhaustion remains visible as a failed recovery job and requires operator review; it does not loop indefinitely or conceal provider/data-quality failures.

## Installation and rollback

Install after the normal post-close timer is installed:

    ./scripts/install_marketops_postclose_recovery_user_timer.sh

Confirm it with:

    systemctl --user list-timers signalops-marketops-postclose-recovery.timer --no-pager

Rollback disables only the guard; it does not delete any evidence or primary timer:

    systemctl --user disable --now signalops-marketops-postclose-recovery.timer

## Observability

The recovery wrapper writes marketops-postclose-recovery status through the existing scheduled-job status contract. The guard also writes marketops-risk-reward stage status including session date, expected active symbols, persisted Risk/Reward result count, snapshot count, and whether recovery is needed, running, succeeded, or failed.

The Admin scheduled-job registry must include both statuses when the surrounding SRI worktree changes are committed. Until then, the status files provide the auditable runtime evidence and the primary post-close job remains the visible composite workflow.

A terminated primary workflow can leave its previous status file as running because it never reaches wrapper cleanup. The recovery guard relies on completion evidence and the systemd/lock state instead of trusting that stale status alone.
