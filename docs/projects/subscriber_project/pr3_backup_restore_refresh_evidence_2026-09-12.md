# PR-3 backup/restore refresh evidence — 2026-09-12

Status: passed for the dedicated MarketOps production data boundary.

## Scope

This refresh used the constrained deployment-agent recovery actions against the dedicated MarketOps databases. It did not move production traffic, alter tenant data, invoke providers, or change Kubernetes production cutover status.

Validated stores:

- `marketops-primary`
- `marketops-temporal`

## Actions executed

```text
sudo -n signalops-deploy-agent backup-run
sudo -n signalops-deploy-agent restore-rehearsal-run
sudo -n signalops-deploy-agent operations-monitor-run
sudo -n signalops-deploy-agent scheduler-status
```

## Backup evidence

The constrained `backup-run` action completed successfully on 2026-09-12 UTC. The command produced no secret-bearing console output and returned exit code `0`.

## Restore rehearsal evidence

The isolated restore rehearsal archive-checked and restored both dedicated MarketOps stanzas.

Primary store evidence:

```text
marketops-primary check completed successfully
repo1 restored backup set 20260901-024502F_20260912-224602D
restore size = 2.3GB, file total = 2584
Restore rehearsal passed for marketops-primary: isolated database started and accepted a validation query.
```

Temporal store evidence:

```text
marketops-temporal check completed successfully
repo1 restored backup set 20260901-024647F_20260912-224652D
restore size = 1.3GB, file total = 2830
Restore rehearsal passed for marketops-temporal: isolated database started and accepted a validation query.
Dedicated MarketOps restore rehearsal passed. Temporary containers and volumes will now be removed.
```

## Operations-monitor evidence

`sudo -n signalops-deploy-agent operations-monitor-run` exited successfully after the restore rehearsal.

A direct read of `/var/lib/signalops/marketops-operations/restore-rehearsal.json` and `/var/lib/signalops/marketops-operations/health.json` was not available through this non-root session because those durable evidence files are root-protected. The deployment-agent operation itself returned success, and the scheduler status below confirms `signalops-marketops-operations-monitor.service` result `success`.

## Scheduler status after recovery refresh

```text
timer=signalops-marketops-boundary-intraday.timer load=loaded active=active next=Mon 2026-09-14 13:30:00 UTC
timer=signalops-marketops-boundary-daily-postclose.timer load=loaded active=active next=Mon 2026-09-14 22:01:55 UTC
timer=signalops-marketops-boundary-postclose-recovery.timer load=loaded active=active next=Mon 2026-09-14 22:30:00 UTC
timer=signalops-marketops-boundary-sri-refresh.timer load=loaded active=active next=Tue 2026-09-15 00:07:00 UTC
timer=signalops-marketops-boundary-sri-holdings-refresh.timer load=loaded active=active next=Tue 2026-09-15 00:20:00 UTC
timer=signalops-marketops-boundary-warm-eod.timer load=loaded active=active next=Mon 2026-09-14 22:00:00 UTC
timer=signalops-marketops-boundary-fmp-annual-financial.timer load=loaded active=active next=Sat 2026-09-19 06:30:00 UTC
timer=signalops-marketops-operations-monitor.timer load=loaded active=active next=Sat 2026-09-12 23:00:00 UTC
service=signalops-marketops-operations-monitor.service load=loaded active=inactive result=success
```

## Result

PR-3 current backup/restore re-verification is closed for the dedicated MarketOps data boundary. Recovery readiness is no longer a deferred production-readiness risk for this cycle.

OpenBao HA remains resilience hardening, not a launch blocker. The production gate is recoverability: current backup/restore evidence, seal/unseal recovery, per-plane policies, audit logging, and rollback controls.
