# PR-0 Scheduler Reconcile Evidence — 2026-08-21

Status: reconcile completed; PR-0 remains pending next eligible post-close proof.

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

## Remaining PR-0 acceptance

PR-0 is not fully closed until the next eligible post-close cycle exits cleanly without requiring stale-systemd reconciliation.

Required next evidence:

1. `marketops-daily-postclose` completes with systemd `result=success` after the next eligible market close.
2. `sudo -n signalops-deploy-agent scheduler-status` remains clean after that run.
3. Dashboard, Market State, Risk/Reward, SRI, and SAF all report the same completed market session.
