# PR-0 / PR-1 Post-Close Closure Evidence — 2026-08-22

Status: PR-0 and PR-1 closed for the August 21, 2026 ET post-close acceptance cycle.

## Evidence collected

Commands/checks executed from the SignalOps workspace:

```text
sudo -n signalops-deploy-agent scheduler-status
scripts/run_subscription_admin_ui_smoke.sh
scripts/run_subscriber_access_control_ui_smoke.sh
scripts/run_marketops_dashboard_freshness_ui_smoke.sh
curl -fsS https://signalops.syncratic.io/readyz
sudo -n signalops-deploy-agent scheduler-run-now:marketops-operations-monitor
```

Results:

```text
/readyz -> {"service":"signalops-gateway","status":"ready","time":"2026-08-22T06:04:44Z"}
Subscription/Admin UI smoke -> 3 passed
Subscriber access-control UI smoke -> 1 passed
Dashboard freshness UI smoke -> 1 passed
Manual operations-monitor run-now -> succeeded
```

## Scheduler and job status

Installed scheduler status was clean for the tracked scheduler set. Active timers were loaded for:

- intraday;
- warm EOD;
- daily post-close;
- post-close recovery;
- SRI refresh;
- SRI holdings refresh.

The next trading-cycle timers point to Monday, August 24, 2026 ET, which is correct because August 22, 2026 is a Saturday.

Dedicated MarketOps job-status evidence showed:

```text
marketops-daily-postclose      succeeded  completed_at=2026-08-21 22:22:48 UTC
marketops-risk-reward          succeeded  completed_at=2026-08-22 03:00:01 UTC
marketops-postclose-recovery   succeeded  completed_at=2026-08-22 03:00:01 UTC
marketops-sri-refresh          succeeded  completed_at=2026-08-22 00:07:14 UTC
marketops-sri-holdings-refresh succeeded  completed_at=2026-08-22 00:20:05 UTC
marketops-intraday             succeeded  completed_at=2026-08-22 00:00:35 UTC
marketops-warm-eod             degraded   reason=bounded_provider_gap
marketops-operations-monitor   succeeded  completed_at=2026-08-22 06:07:24 UTC
```

`marketops-warm-eod` is intentionally `degraded`, not failed. The bounded provider gap behavior was the source fix for small no-bar gaps in the 1000-asset warm cohort.

## Completed-session data evidence

Core ledgers aligned to the August 21, 2026 ET completed session:

```text
Market State  2026-08-21  132 symbols  latest_as_of=2026-08-21 22:04:31 UTC
Risk/Reward   2026-08-21  132 symbols  latest_observed=2026-08-21 22:19:58 UTC
SRI           2026-08-21   16 segments
Intraday      2026-08-21  132 symbols  latest_snapshot=2026-08-21 22:15:00 UTC
```

SAF benchmark observations showed matured-session evidence through August 20, 2026. That is consistent with the configured SAF maturation window and does not indicate a missed August 21 post-close run.

## Closure decision

PR-0 is closed because the next eligible post-close cycle completed without another stale-systemd reconcile.

PR-1 is closed because Admin Operations Health is browser-verified and the underlying data-freshness ledgers align to the latest completed session for the core MarketOps views.

## Remaining non-blocking items

- The deployment agent was subsequently reprovisioned and live `scheduler-status` now includes warm-EOD and FMP service rows.
- The FMP annual financial timer was subsequently activated for the selected weekly Saturday 02:30 ET cadence. First natural run is Saturday, August 29, 2026 at 06:30 UTC.
- PR-3 remains deferred by accepted product risk; current backup/restore evidence was intentionally not refreshed in this cycle.
