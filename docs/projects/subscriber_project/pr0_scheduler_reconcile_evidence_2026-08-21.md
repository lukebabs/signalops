# PR-0 Scheduler Reconcile Evidence — 2026-08-21

Status: closed after the next eligible post-close proof.

## Action performed

The root-owned SignalOps deployment-control agent was reprovisioned with the approved constrained action:

```text
marketops-postclose-systemd-reconcile
```

The installed agent output confirmed the action is available in the allowlist.

## Result

After running the reconcile flow, `sudo -n signalops-deploy-agent scheduler-status` returned clean service state for the MarketOps scheduler set.

Relevant result:

```text
service=signalops-marketops-boundary-schedule@marketops-daily-postclose.service load=loaded active=inactive result=success
service=signalops-marketops-boundary-schedule@marketops-postclose-recovery.service load=loaded active=inactive result=success
service=signalops-marketops-boundary-schedule@marketops-sri-refresh.service load=loaded active=inactive result=success
service=signalops-marketops-boundary-schedule@marketops-sri-holdings-refresh.service load=loaded active=inactive result=success
service=signalops-storage-monitor.service load=loaded active=inactive result=success
service=signalops-retention-governance.service load=loaded active=inactive result=success
```

Active timers remained loaded and active for intraday, daily post-close, post-close recovery, SRI refresh, SRI holdings refresh, and warm EOD. The FMP annual financial timer remained loaded but inactive, which is tracked separately under the FMP lifecycle gate.

## Control boundary

This was not a general suppression of scheduler failures. The reconcile action is constrained to reset stale post-close systemd failure state only after dedicated MarketOps DB evidence verifies post-close completion and recovery state.

## 2026-08-22 post-close acceptance evidence

The Friday, August 21, 2026 ET post-close window completed without requiring another stale-systemd reconcile.

Live evidence collected on 2026-08-22 UTC:

```text
sudo -n signalops-deploy-agent scheduler-status
```

returned clean tracked service state for the installed scheduler set. Timers remained active for intraday, warm EOD, daily post-close, post-close recovery, SRI refresh, and SRI holdings refresh, with next runs on Monday, August 24, 2026 ET.

Dedicated MarketOps status rows showed:

```text
marketops-daily-postclose      succeeded  completed_at=2026-08-21 22:22:48 UTC
marketops-risk-reward          succeeded  completed_at=2026-08-22 03:00:01 UTC
marketops-postclose-recovery   succeeded  completed_at=2026-08-22 03:00:01 UTC
marketops-sri-refresh          succeeded  completed_at=2026-08-22 00:07:14 UTC
marketops-sri-holdings-refresh succeeded  completed_at=2026-08-22 00:20:05 UTC
marketops-intraday             succeeded  completed_at=2026-08-22 00:00:35 UTC
marketops-warm-eod             degraded   reason=bounded_provider_gap
```

The `marketops-warm-eod` degraded result is the expected governed state for a bounded provider no-bar gap. It is no longer a hard scheduler failure.

Core completed-session ledgers aligned to the August 21, 2026 ET session:

```text
Market State  2026-08-21  132 symbols  latest_as_of=2026-08-21 22:04:31 UTC
Risk/Reward   2026-08-21  132 symbols  latest_observed=2026-08-21 22:19:58 UTC
SRI           2026-08-21   16 segments
Intraday      2026-08-21  132 symbols  latest_snapshot=2026-08-21 22:15:00 UTC
```

SAF benchmark rows were refreshed with matured-session evidence through August 20, 2026, which is correct for the configured signal-outcome maturation window.

The operations monitor had intermittent failed runs during the post-close/recovery window, but the current run-now check completed successfully:

```text
marketops-operations-monitor-20260822T060712Z succeeded exit_code=0
```

PR-0 is closed for the August 21 post-close acceptance cycle. Future monitor-cadence hardening remains a normal PR-4 operations-control improvement, not a PR-0 blocker.
