# MarketOps Production Engineering Handbook

Status: production-readiness draft for operators and engineers.

## Operating objective

MarketOps production must provide tenant-safe subscriber UX over one centrally governed MarketOps data plane. The system should avoid tenant-specific provider polling, duplicate asset storage, and manual recovery paths that hide data drift.

## Source of truth

| Domain | Production source |
|---|---|
| MarketOps operational data | Dedicated MarketOps PostgreSQL and TimescaleDB databases. |
| Scheduled-job state | Dedicated MarketOps database scheduled-job status tables. |
| MarketOps browser/API reads | Gateway projections over dedicated MarketOps data. |
| Watchlists/subscriptions | Subscriber tables in the dedicated MarketOps primary database. |
| Runtime operator actions | Constrained `signalops-deploy-agent` actions. |
| Provider data | Central workers only; never browser-triggered. |

The shared SignalOps database remains for non-MarketOps platform/CyberOps data and rollback evidence. It is not the active MarketOps analytical source after the dedicated MarketOps cutover.

## Required production invariants

1. MarketOps scheduled jobs run against the dedicated MarketOps primary and temporal stores.
2. Continuous writers for MarketOps are running and restartable.
3. Provider polling is centrally deduplicated.
4. Warm assets receive EOD processing; hot assets receive intraday processing.
5. Watchlist membership drives user scope but does not create duplicate data collection.
6. Subscription and RBAC checks both pass before feature access is granted.
7. Weekends and configured market holidays skip EOD/intraday work.
8. Admin Operations Health explains stale, partial, degraded, or skipped states.
9. Recovery actions use constrained deployment-agent commands, not broad manual shell intervention.
10. Every production readiness change is documented, committed, and pushed.

## Scheduler and job model

Current controlled timers:

| Timer | Purpose |
|---|---|
| `signalops-marketops-boundary-intraday.timer` | Hot-list intraday monitoring during market/session window. |
| `signalops-marketops-boundary-warm-eod.timer` | Warm 1,000-asset EOD baseline. |
| `signalops-marketops-boundary-daily-postclose.timer` | Tenant-local legacy hot post-close algorithm chain. |
| `signalops-marketops-boundary-postclose-recovery.timer` | Bounded recovery and reconciliation after post-close. |
| `signalops-marketops-boundary-sri-refresh.timer` | Sector Rotation Intelligence refresh. |
| `signalops-marketops-boundary-sri-holdings-refresh.timer` | ETF holdings/makeup refresh. |
| `signalops-marketops-boundary-fmp-annual-financial.timer` | Weekly FMP annual fundamentals enrichment. |

Baseline status command:

```bash
sudo -n signalops-deploy-agent scheduler-status
```

Expected healthy state:

- all required timers loaded and active;
- latest tracked one-shot services inactive with `result=success`;
- degraded jobs have bounded, documented reasons;
- non-trading-day skips are explicit, not silent.

## Post-close completion contract

The daily post-close workflow is not complete until both source tables and global subscriber projections are current.

Required post-close projection classes:

- options distributions;
- Risk/Reward snapshots;
- Market State evidence;
- SAF outcome observations;
- EROC valuation evidence for `signalops.algorithms.eroc_v6`.

The EROC production issue fixed on 2026-08-22 was a projection failure, not an algorithm failure: tenant-local EROC results were current, but `subscriber_gateway_global_eroc_results` stopped after 2026-08-14. The permanent fix adds constrained EROC valuation materialization to `scripts/marketops_global_dashboard_projection.sh` and fails the post-close gate when global EROC symbol coverage is lower than tenant-local source coverage for the same session.

## Operations Health acceptance

Admin Operations Health should expose at least:

- Dashboard;
- Assets analytical coverage;
- Market State;
- Risk/Reward;
- Sector Rotation Intelligence;
- Signal Assurance;
- Intraday conditions;
- FMP annual financials.

The row status must be actionable:

- `current`: expected evidence exists or the market is correctly idle;
- `partial` / `degraded`: bounded issue with counts/reason;
- `stale`: behind expected session;
- `missing`: required evidence absent;
- `skipped`: non-trading day or explicitly allowed schedule skip.

## Access and subscription controls

Production access requires both:

1. tenant/use-case authorization from the signed OIDC principal; and
2. subscription feature entitlement where enforcement is enabled.

Expected failures:

| Failure | Expected code |
|---|---|
| Cross-tenant route | `403 tenant_mismatch` |
| Missing SignalOps role | `403 insufficient_role` |
| Missing subscription | `402 subscription_required` |
| Plan lacks feature | `402 subscription_feature_required` |
| Backend entitlement resolver unavailable | `503`, never permissive fallback |

Subscription Administration belongs in Administration, not MarketOps.

## Validation checklist

Use this checklist before declaring a production-readiness gate closed.

1. `git status --short` is clean.
2. `/readyz` returns 200.
3. Web and Gateway routes render for the configured tenant-local and pilot identities.
4. `sudo -n signalops-deploy-agent scheduler-status` is clean.
5. Admin Operations Health rows are current or explicitly degraded/skipped.
6. Latest completed-session counts align across Dashboard, Market State, Risk/Reward, SRI, SAF, EROC, and intraday where applicable.
7. Cross-tenant and subscription smokes pass.
8. No provider polling occurred outside approved schedules/actions.
9. New migrations/scripts are documented.
10. Changes are committed and pushed to `subscribers`.

## Common commands

```bash
curl -fsS https://signalops.syncratic.io/readyz
sudo -n signalops-deploy-agent scheduler-status
sudo -n signalops-deploy-agent operations-monitor-run
sudo -n signalops-deploy-agent scheduler-run-now:marketops-postclose-recovery
sudo -n signalops-deploy-agent marketops-postclose-systemd-reconcile
```

Use direct `docker`/`systemctl` only when a constrained deployment-agent action does not exist and the action is explicitly approved.

## Current production-readiness open items

As of 2026-08-22:

1. Observe the next natural trading-day post-close run and confirm EROC global projection advances through the standard parity/materializer path.
2. Observe the first scheduled FMP annual financial run on 2026-08-29 at 02:30 America/New_York.
3. Decide whether to refresh PR-3 backup/restore evidence before broader paid pilot expansion.
4. Continue hardening operations-monitor cadence to reduce transient false failures during long post-close windows.

