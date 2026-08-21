# PR-4 Production Expansion Controls — 2026-08-21

Status: started.

## Decision boundary

PR-3 backup/restore rehearsal is intentionally deferred by product decision. Prior dedicated MarketOps pgBackRest backup and isolated restore rehearsal evidence remains useful, but it is not current after the latest PR-1/PR-2 changes. This is accepted as a known readiness risk, not closed recovery evidence.

PR-4 now focuses on production expansion controls that can be improved without widening provider polling or tenant access.

## Scope

PR-4 covers four areas:

1. **FMP annual financial lifecycle**
   - Keep the annual enrichment centrally governed.
   - Ensure it is visible as an Admin job/freshness row.
   - Keep it failure-isolated so one bad symbol or endpoint does not fail the full workflow.
   - Respect the current FMP Starter 300-calls/minute entitlement while retaining a conservative worker throttle unless changed by explicit release decision.

2. **Trading-calendar correctness**
   - Move from weekend-only guards toward a canonical US market-session calendar.
   - Use the same calendar for job eligibility and UI 10/20 trading-day interpretation.
   - Explicitly prevent EOD/intraday jobs on weekends and market holidays, except maintenance jobs.

3. **Subscriber administration**
   - PR-2 already closed the governance-surface smoke for configured production QA identities.
   - Remaining PR-4 work is polish/operationalization, not core access-control closure.

4. **Incident runbooks**
   - Maintain concise runbooks for stale dashboard data, failed post-close, provider outage, access-control regression, failed deployment smoke, and backup/restore.
   - Each runbook must include detection, owner, first response, recovery action, verification, and rollback criteria.

## Current source state

- `marketops-fmp-annual-financial` exists as an Admin-visible scheduled job and deployment-agent run-now target.
- The FMP annual worker records per-asset task status and workflow coverage, and marks the workflow `degraded` instead of failing the entire run when individual tasks fail/defer.
- The installer supports `--enable-fmp-annual` and would enable the Saturday 02:30 America/New_York timer.
- `scripts/lib/marketops_trading_calendar.sh` currently provides a weekend guard only. It allows maintenance jobs and FMP annual on weekends.

## First implementation target

Add a reusable MarketOps calendar primitive with a static US market-holiday list for the current operational range, then wire scheduler guards and tests around it. This is safer than immediately enabling new provider schedules.

Acceptance for the first slice:

- Calendar helper returns non-trading day for weekends and configured US market holidays.
- EOD/intraday/SRI/post-close jobs skip on non-trading days.
- Maintenance and FMP annual jobs remain permitted on weekends by explicit allowlist.
- Documentation records how to update the holiday list.

## First slice implementation — trading-calendar guard

Implemented an explicit MarketOps trading-calendar helper in `scripts/lib/marketops_trading_calendar.sh`:

- supports weekend detection;
- includes static 2026 and 2027 US market-holiday closures;
- exposes `marketops_is_trading_day`;
- exposes `marketops_non_trading_reason`;
- preserves the explicit maintenance/FMP weekend allowlist.

`scripts/marketops_scheduled_job.sh` now uses the unified non-trading-day predicate. EOD, intraday, SRI, warm-EOD, retry, and post-close jobs skip on weekends and configured market holidays. Maintenance jobs and `marketops-fmp-annual-financial` remain explicitly permitted on weekends.

Added `scripts/test_marketops_trading_calendar.sh` to prove:

- 2026-08-21 is a trading day;
- 2026-08-22 is a weekend non-trading day;
- 2026-09-07, 2026-11-26, and 2027-07-05 are market holidays;
- FMP annual is weekend-permitted;
- daily post-close is not weekend-permitted.

Verification:

```text
bash -n scripts/lib/marketops_trading_calendar.sh scripts/marketops_scheduled_job.sh scripts/test_marketops_trading_calendar.sh
scripts/test_marketops_trading_calendar.sh
marketops_trading_calendar_tests_passed
scripts/run_subscription_admin_ui_smoke.sh
...                                                                      [100%]
3 passed in 3.20s
```

Operational note: the holiday list is intentionally static and must be updated annually from the official exchange calendar before enabling a new production year.

## Second slice implementation — SAF UI trading-day filters

Implemented a frontend MarketOps trading-calendar helper in `web/src/lib/marketopsTradingCalendar.ts` and wired the SAF daily progression 10/20-day filters to it.

The SAF progression chart previously interpreted trading-day windows as weekdays only. That was inconsistent with the PR-4 scheduler guard because market holidays would be skipped by jobs but still counted by the UI filter. The UI now excludes both weekends and configured 2026/2027 US market holidays when selecting trailing 10/20 trading-day observation windows.

Verification:

```text
npm --prefix web test -- marketopsTradingCalendar.test.ts
✓ src/lib/marketopsTradingCalendar.test.ts (2 tests)

npm --prefix web test
Test Files  37 passed (37)
Tests  440 passed (440)

npm --prefix web run build
✓ built
```

Operational note: `web/src/lib/marketopsTradingCalendar.ts` and `scripts/lib/marketops_trading_calendar.sh` intentionally duplicate the same static holiday list for now. They must be updated together until a later shared calendar source is introduced.

## Third slice decision — FMP annual recurring cadence

Decision: Option B is selected. `marketops-fmp-annual-financial` should run as a governed recurring weekly job.

Production cadence:

- Timer: `signalops-marketops-boundary-fmp-annual-financial.timer`
- Schedule: Saturday 02:30 America/New_York
- Job: `marketops-fmp-annual-financial`
- Worker: `scripts/marketops_annual_financial_task_worker.sh`
- Provider scope: FMP annual financial enrichment only
- Failure posture: task-level failures/deferred rows are recorded; the workflow may finish degraded instead of failing catastrophically on one symbol
- Operations surface: Admin scheduled jobs and operations freshness expose status/freshness

Control path:

- The installer supports `--enable-fmp-annual`.
- The deployment agent now exposes constrained actions:
  - `scheduler-fmp-annual-enable`
  - `scheduler-fmp-annual-disable`
- These actions only enable/disable `signalops-marketops-boundary-fmp-annual-financial.timer`; they do not grant general `systemctl` access.

Recommended live enablement command after deployment-agent reprovision/deploy:

```bash
sudo -n signalops-deploy-agent scheduler-fmp-annual-enable
sudo -n signalops-deploy-agent scheduler-status
```

Rollback command:

```bash
sudo -n signalops-deploy-agent scheduler-fmp-annual-disable
sudo -n signalops-deploy-agent scheduler-status
```

Acceptance:

- Timer is loaded and active.
- Next run is Saturday 02:30 America/New_York.
- Admin Scheduled Jobs lists FMP annual financial capture.
- Operations Health contains `FMP annual financials` freshness.
- The next natural run completes as `succeeded` or `degraded` with per-task evidence, not an untracked failure.

## Fourth slice implementation — incident runbooks

Added `pr4_incident_runbooks_2026-08-21.md` with production operator runbooks for:

- stale Dashboard or cross-view freshness drift;
- failed daily post-close or recovery guard;
- provider outage or provider schema drift;
- access-control or subscription regression;
- failed deployment smoke or post-login 404;
- backup/restore concern;
- FMP annual financial degradation.

Each runbook includes detection, owner, first response, recovery action, verification, and rollback criteria. The runbooks prefer constrained deployment-agent actions and dedicated MarketOps database evidence over manual shell intervention.

## Live activation attempt — FMP annual recurring timer

Attempted to activate Option B after source deployment-agent support was pushed.

Observed baseline:

```text
sudo -n signalops-deploy-agent scheduler-status
...
timer=signalops-marketops-boundary-fmp-annual-financial.timer load=loaded active=inactive next=n/a
```

The installed deployment agent did not yet include the new constrained action:

```text
sudo -n signalops-deploy-agent scheduler-fmp-annual-enable
Unsupported deployment-agent action: scheduler-fmp-annual-enable
```

Reprovisioning the root-owned deployment agent from source requires an interactive sudo password in the current host session:

```text
sudo -n ./scripts/provision_signalops_deployment_agent.sh adminalien
sudo: a password is required
```

No timer state changed. Scheduler status remained clean and FMP annual remained inactive.

Required one-time operator action:

```bash
cd /home/adminalien/docker/syncratic-core/subsystems/signalops
sudo ./scripts/provision_signalops_deployment_agent.sh adminalien
sudo -n signalops-deploy-agent scheduler-fmp-annual-enable
sudo -n signalops-deploy-agent scheduler-status
```

Acceptance remains: FMP annual timer must be `active=active` with next run at Saturday 02:30 America/New_York.

## Pause checkpoint — pending August 21 ET EOD acceptance

Active PR-4 implementation is paused at the controlled pilot-ready checkpoint until the next natural post-close cycle completes for the Friday, August 21, 2026 ET trading session.

Clarification: SRI refresh and SRI holdings refresh run after midnight UTC on Saturday, August 22, 2026, but they are part of the August 21 ET trading-session acceptance window. They should not be interpreted as a separate August 22 EOD trading day.

Acceptance after the window:

```bash
sudo -n signalops-deploy-agent scheduler-status
```

Then verify Dashboard, Assets coverage, Market State, Risk/Reward, SRI, SAF, and Admin Operations Health align to the same completed-session evidence without manual reconcile.

FMP annual recurring activation remains source-ready but not live-active until the deployment agent is reprovisioned and `scheduler-fmp-annual-enable` is accepted by the installed root-owned agent.

## Fifth slice implementation — operations freshness correction

Implemented the recommendations from the Aug 20 freshness review:

- `marketops-warm-eod` is now a first-class scheduled job in the Admin/API job list.
- The deployment-agent source scheduler-status now checks `marketops-warm-eod` and FMP annual scheduled services; this becomes live after the root-owned deployment agent is reprovisioned.
- Warm EOD now treats bounded provider no-bar gaps as `degraded` using exit code `10`, while the scheduler wrapper records `degraded` and exits cleanly to avoid stale systemd failure for acceptable small provider gaps.
- Admin Operations Health now reports `Assets analytical coverage` from current Market State evidence instead of one-time coverage activation metadata.
- Intraday freshness now distinguishes live-market staleness from after-hours market-idle completed-session evidence.
- SRI and SAF rows now explain that their latest as-of timestamp is provenance/materialization time while session date is the freshness authority.

Live gateway deployment evidence:

```text
signalops_public_production_deploy_verified
2 passed in 7.33s
curl -fsS https://signalops.syncratic.io/readyz
{"service":"signalops-gateway","status":"ready",...}
scripts/run_subscription_admin_ui_smoke.sh
3 passed in 3.05s
scripts/run_subscriber_access_control_ui_smoke.sh
1 passed in 3.59s
```
