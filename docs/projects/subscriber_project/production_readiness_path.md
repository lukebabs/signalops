# Subscriber Project — Production Readiness Path

Status: controlled pilot-ready checkpoint advanced; PR-0 and PR-1 are closed after the August 21, 2026 ET post-close acceptance cycle. Syncratic Ask live smoke is now passing as a production-readiness QA control.

Last reviewed: 2026-08-22.

## Readiness position

The Subscriber Project is **controlled pilot-ready**, not full external production-ready.

The current platform has enough structure to keep validating with controlled tenants and named approvals: dedicated MarketOps databases are live, web and gateway are serving, SAF viability analytics are visible, and the global-data projection work has advanced materially.

Production readiness is still blocked by expansion and recovery-evidence gaps. The core operational-consistency loop for the tenant-local pilot scope has now passed one natural post-close acceptance cycle after the MarketOps database decoupling.

## Current evidence snapshot — 2026-08-22

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Web and Gateway availability | Ready | Public `/readyz` returned `200`; Admin, access-control, and Dashboard freshness Playwright smokes passed. |
| Dedicated MarketOps data boundary | Ready | `signalops-marketops-postgres-1` and `signalops-marketops-timescaledb-1` were healthy. MarketOps is intended to read/write the dedicated MarketOps databases, not the old shared MarketOps tables. |
| Completed-session global data | Ready for pilot scope | Market State and Risk/Reward showed 132/132 tenant-local symbols for the 2026-08-21 session; SRI showed 16/16 segments for 2026-08-21; SAF matured evidence remains governed by the configured maturation window. |
| Intraday freshness | Ready / market-idle | Latest intraday snapshot was 2026-08-21 22:15 UTC with 132 tenant-local symbols. Saturday, August 22 has no EOD/intraday trading session. |
| Scheduled jobs | Ready for pilot scope | `scheduler-status` returned clean after the August 21 post-close window. Daily post-close, Risk/Reward, recovery, SRI, holdings, and intraday completed successfully. Warm EOD returned governed `degraded` with `bounded_provider_gap`. |
| Daily post-close | Closed for PR-0 | The August 21 post-close cycle completed without requiring stale-systemd reconciliation. |
| FMP annual financial job | Active / observing | Option B selected and activated: weekly Saturday 02:30 ET recurring capture. Live scheduler status shows `signalops-marketops-boundary-fmp-annual-financial.timer active=active next=Sat 2026-08-29 06:30:00 UTC`. |
| Deployment automation | Mostly ready | Production route checks and constrained Playwright smokes now pass, including the controlled Syncratic Ask live smoke after AI Gateway price-catalog propagation. PR-1 Admin freshness acceptance corrected the false `/marketops/admin` check to the real `/admin/system` route. |
| SAF operational viability | Pilot-ready | SAF progression chart, 10/20-day filters, and inline drill-down are live. Historical viability is currently strongest for the tenant-local 132-asset legacy cohort and should continue maturing naturally unless a separate backtest gate is approved. |
| Subscription/access controls | Ready for configured QA identities | PR-2 closed tenant isolation, private-list owner projection, tier-enforcement canary, restoration, and Subscription Administration governance-surface browser evidence. |
| Backup/restore | Deferred risk | Dedicated pgBackRest backup and isolated restore rehearsal previously passed. PR-3 current re-verification is intentionally deferred by product decision and remains a known readiness risk. |

## Production gates

### P0 — Operational correctness gates

These must close before wider pilot or paid production.

1. **Daily post-close must end cleanly**
   - Status: closed for the current pilot path after the August 21, 2026 ET post-close acceptance cycle.
   - Evidence:
     - `sudo -n signalops-deploy-agent scheduler-status` returned clean tracked service state.
     - The next eligible post-close run exited successfully without another stale-systemd reconcile.
     - Dedicated MarketOps status rows showed job result, timestamps, and the governed `marketops-warm-eod` bounded-provider-gap warning.
     - Market State, Risk/Reward, SRI, and intraday aligned to the August 21 completed session; SAF remained consistent with its configured maturation window.

2. **Scheduler/data-plane contract must be uniform**
   - Every MarketOps scheduled job must load the same dedicated MarketOps database configuration and write status to the dedicated MarketOps operations tables.
   - No routine job should silently fall back to the old shared MarketOps tables.
   - Acceptance:
     - Intraday, EOD, Risk/Reward, SRI, SAF projection, FMP annual, operations monitor, retention governance, and storage monitor all report through the same status layer.
     - Failed jobs expose root cause and recovery action without requiring HAR files or manual log archaeology.

3. **Deployment smoke must be trustworthy**
   - Fix the deployment-agent web smoke false-positive `404`.
   - Acceptance:
     - `signalops-production-web-deploy` succeeds only when local and public route checks pass.
     - Protected routes validate with the configured QA identities.
     - A deploy failure means a real release risk, not a test harness mismatch.

### P1 — Secure pilot gates

These are required before expanding beyond tightly controlled users.

4. **Subscription and access-control canaries**
   - Re-run enforcement with tenant-local admin and tenant-pilot-b subscriber identities.
   - Validate Explorer, Professional, and Institutional entitlement behavior.
   - Validate cross-tenant denial, admin-only settings, tenant-default list behavior, and private-list ownership.
   - Acceptance:
     - No tenant can read or mutate another tenant's private objects.
     - Non-admin users cannot perform admin duties.
     - Subscription enforcement can be enabled, tested, and restored without residue.

5. **Admin operations status control**
   - Status: closed for the current pilot path after Admin Operations Health and Dashboard freshness browser smokes passed.
   - Deployment-agent expansion is now installed: `scheduler-status` lists warm-EOD and FMP scheduled-service rows.
   - Acceptance evidence:
     - Admin can see core job/data freshness without shell access.
     - Run-now uses constrained deployment-agent actions, not broad shell access.

6. **Backup and restore re-verification**
   - Run a current dedicated MarketOps pgBackRest backup and isolated restore rehearsal.
   - Acceptance:
     - Backup completes against the current repository path and current schema.
     - Restore rehearsal starts an isolated database and accepts validation queries.
     - The runbook records owner, command, timestamp, backup label, restore outcome, and rollback boundary.

### P2 — Production expansion gates

These are required before broader commercial production.

7. **FMP annual financial lifecycle**
   - Convert FMP annual enrichment into a governed recurring/admin-visible job.
   - Keep central polling budgeted and failure-isolated.
   - Acceptance:
     - A single failed symbol or endpoint does not fail the full task.
     - Coverage, staleness, and skipped/failed symbols are visible.
     - The task honors the paid 300-calls/minute plan without returning to a 240/day assumption.

8. **Trading-calendar correctness**
   - Replace approximate weekday windowing with a canonical market calendar for UI filters and job eligibility.
   - Acceptance:
     - 10/20 trading-day filters match actual US market sessions.
     - Weekend and holiday guards prevent EOD/intraday jobs from running except explicit maintenance.
   - Current PR-4 evidence: scheduler eligibility now uses a reusable MarketOps calendar helper with explicit 2026/2027 US market-holiday closures; SAF UI 10/20-day progression filters now use the matching frontend calendar helper.

9. **Subscriber administration**
   - Expand Subscription Administration into a complete governance surface.
   - Acceptance:
     - Admins can govern enrolled users, tenant membership, tier assignment, entitlement state, quota state, default-list policy, audit evidence, and user activity visibility.
     - Billing/provider integration remains separated until explicitly approved.
   - Current status: migration `000157_subscriber_user_activity_ledger` is applied in the dedicated MarketOps database and gateway/web are deployed. Subscription Administration now exposes append-only login/logout/feature-view/mutation activity through an Activity tab and per-user drilldown. Detail-retention automation for the 180-day target remains a follow-up operations-control item.

10. **Incident runbooks**
    - Maintain runbooks for stale dashboard data, failed post-close, provider outage, access-control regression, failed deployment smoke, backup/restore, and FMP annual degradation.
    - Acceptance:
      - Each runbook includes detection, owner, first response, recovery action, verification, and rollback criteria.
    - Current PR-4 evidence: `pr4_incident_runbooks_2026-08-21.md` defines all required response paths using constrained deployment-agent actions and dedicated MarketOps database evidence.

## Efficient and secure path to production

### Sprint PR-0 — Stabilize scheduled execution

Scope:

- Fix daily post-close failure semantics/root cause.
- Verify all scheduled jobs load the dedicated MarketOps data-plane configuration.
- Ensure scheduler status returns clean after the next eligible run.

Exit:

- No failed MarketOps scheduled services.
- Admin/job-status evidence identifies latest completed market session across all core views.

Implementation note — 2026-08-21:

- The approved `marketops-postclose-systemd-reconcile` deployment-agent action was added in source. It is intentionally narrow: it resets stale failed post-close systemd state only after the dedicated MarketOps database proves post-close, recovery, Risk/Reward, and SRI status are in allowed recovered states. It does not make `scheduler-status` ignore real failures.
- Live evidence now shows the stale `marketops-daily-postclose` systemd failure was reconciled and `scheduler-status` returned clean service states.

Implementation note — 2026-08-22:

- The next eligible post-close cycle completed without requiring another stale-systemd reconcile. `marketops-daily-postclose`, `marketops-risk-reward`, `marketops-postclose-recovery`, `marketops-sri-refresh`, `marketops-sri-holdings-refresh`, and `marketops-intraday` succeeded. `marketops-warm-eod` returned the expected governed `degraded` state for a bounded provider no-bar gap. PR-0 is closed for the current pilot path.

### Sprint PR-1 — Make operations visible

Scope:

- Expose status control in Admin Workbench.
- Show job result, freshness, warnings, and run-now actions.
- Keep run-now behind constrained deployment-agent actions.

Exit:

- An administrator can determine whether Dashboard, Assets, Market State, Risk/Reward, SRI, SAF, and FMP data are fresh without shell access.

Implementation note — 2026-08-21:

- The first source slice added read-only data-freshness visibility to the Administration operations-health API and Admin Workbench for Dashboard alignment, Market State, Risk/Reward, SRI, SAF, and Intraday.
- The follow-on source slice added Assets analytical coverage and FMP annual financial workflow freshness. PR-1 source coverage is now complete. Browser acceptance is automated and passed through `scripts/run_subscription_admin_ui_smoke.sh`.

### Sprint PR-2 — Harden access and subscriptions

Scope:

- Validate tier enforcement with real tenant-local and tenant-pilot-b identities.
- Expand Subscription Administration for user and tier governance.
- Record cross-tenant/private-list tests.

Exit:

- Access-control evidence is repeatable and subscription enforcement can be enabled safely.

Implementation note — 2026-08-21:

- Added `scripts/run_subscriber_access_control_ui_smoke.sh` and `python/tests/test_subscriber_access_control_ui.py` as a read-only real-OIDC PR-2 smoke.
- Live result: `1 passed in 3.55s`.
- The smoke proves tenant-pilot-b and tenant-local tokens are denied from each other’s tenant-bearing MarketOps routes with `403 tenant_mismatch`, while Subscription Administration remains available only to the platform subscription-admin identity.
- The temporary production subscription-enforcement canary ran under named approval and restored successfully. It verified Explorer denial, Professional access, Professional denial for Institutional-only SAF analytics, and tenant-local Institutional/admin access. Post-restore route, scheduler, tenant-isolation, and Admin browser smokes passed. The closing PR-2 browser smokes now also verify private-list owner-subject projection and the Subscription Administration governance surface. PR-2 is closed for the configured production QA identities; a future second same-tenant adversarial identity can deepen, but does not block, the current gate.

### Sprint PR-3 — Rehearse recovery

Status: deferred by product decision.

Scope:

- Run current backup and isolated restore rehearsal.
- Confirm recovery-control scripts never rebuild or disturb live databases during routine backup/rehearsal.

Exit:

- Backup/restore evidence is current after the latest production schema and runtime changes.

Deferral note — 2026-08-21:

- Product chose to skip PR-3 for now because a dedicated backup and isolated restore rehearsal passed a few days earlier. This does not convert PR-3 to closed; it records accepted risk that recovery evidence is not current after PR-1/PR-2 changes.

### Sprint PR-4 — Production expansion controls

Status: started.

Scope:

- Govern FMP annual enrichment lifecycle. Option B selected: weekly Saturday 02:30 ET recurring timer, controlled through constrained deployment-agent enable/disable actions.
- Add trading-calendar correctness. Scheduler non-trading-day skips and SAF UI 10/20-day filters now use matching 2026/2027 US market-holiday semantics.
- Complete subscriber administration workflow.
- Finalize incident runbooks. Source runbooks now cover stale data, post-close failure, provider outage, access regression, deployment smoke/404, backup/restore, and FMP annual degradation.

Exit:

- Platform can support paid pilot tenants with secure access, consistent data, visible operations, and documented recovery paths.

Implementation note — 2026-08-21:

- PR-4 starts with controls, not broader provider polling. The first implementation added a reusable MarketOps trading-calendar primitive and wired scheduled jobs to skip configured US market holidays as well as weekends, while preserving the explicit maintenance/FMP weekend allowlist.

## Standing readiness check

Each production-readiness review should record:

1. Git revision and working-tree status.
2. Web, Gateway, dedicated MarketOps Postgres, and dedicated MarketOps Timescale container health.
3. Public `/readyz` and the key MarketOps browser routes, including Syncratic Ask smoke when AI Gateway policy/catalog changed.
4. `sudo -n signalops-deploy-agent scheduler-status`.
5. Latest completed market session and row counts for Dashboard, Assets, Market State, Risk/Reward, SRI, SAF, and FMP.
6. Latest intraday snapshot and hot-symbol count during market hours.
7. Subscription/access-control canary result.
8. Backup label and restore rehearsal result.
9. Open blockers classified as Ready, Partial, or Blocked.

## Next recommended action

Move to the remaining PR-4/production-expansion work:

1. Observe the first scheduled FMP annual run on Saturday, August 29, 2026 at 02:30 ET / 06:30 UTC and verify task coverage/degradation reporting.
2. Decide whether to refresh PR-3 backup/restore evidence before wider paid pilot expansion.
3. Continue monitor-cadence hardening so operations-monitor checks do not produce avoidable transient failures during long post-close/recovery windows.

## 2026-08-21 05:17 UTC readiness update

- Public `/readyz` returned `200`.
- Public `/admin/system` returned `200`.
- `sudo -n signalops-deploy-agent scheduler-status` returned clean timer/service state.
- PR-1 Admin freshness browser acceptance passed through Playwright: the test logs in as `luke@strategiclabs.io`, opens `/admin/system`, asserts the operations-health API response, and verifies all eight freshness labels render.
- The remaining time-gated acceptance item is the next natural post-close cycle at 2026-08-21 22:01:55 UTC, followed by recovery/SRI timers through the controlled post-close window.

## 2026-08-21 06:10 UTC freshness semantics correction

- Investigated the apparent Aug 20 freshness drift. Dashboard, Market State, Risk/Reward, SRI, SAF, and intraday all had Aug 20 completed-session evidence for the tenant-local 132-asset scope.
- Identified the real failed job as `marketops-warm-eod`: Aug 20 warm EOD normalized 997/1000 symbols and failed strict completion because provider no-bar responses were returned for a small number of symbols.
- Source fix: warm EOD now returns a governed `degraded` result for bounded provider gaps (`MARKETOPS_WARM_EOD_MAX_MISSING_SYMBOLS`, default `5`) and records the missing symbols in output instead of leaving systemd in a hard failed state for small no-bar gaps.
- Source fix: scheduler-status now includes `marketops-warm-eod` and FMP annual scheduled services, and Admin Scheduled Jobs exposes `MarketOps warm EOD baseline`. The installed root-owned deployment agent has now been reprovisioned and the new scheduler-status service list is live.
- Source/live gateway fix: Admin Operations Health now labels the row as `Assets analytical coverage`, uses current Market State evidence for the selected assets instead of one-time coverage activation metadata, treats after-hours intraday as market-idle current when the latest completed-session evidence is present, and adds explicit provenance notes for SRI/SAF as-of timestamps.
- Live gateway deployment completed and verified: `signalops_public_production_deploy_verified`, deployment smoke `2 passed`, `/readyz` returned `200`, Subscription/Admin smoke `3 passed`, and subscriber access-control smoke `1 passed`.

## 2026-08-22 06:15 UTC post-close closure update

- Public `/readyz` returned `200`.
- `scripts/run_subscription_admin_ui_smoke.sh` passed: `3 passed`.
- `scripts/run_subscriber_access_control_ui_smoke.sh` passed: `1 passed`.
- `scripts/run_marketops_dashboard_freshness_ui_smoke.sh` passed: `1 passed`.
- `sudo -n signalops-deploy-agent scheduler-status` returned clean tracked service state.
- Dedicated MarketOps status rows showed the August 21 post-close chain completed: daily post-close, Risk/Reward, post-close recovery, SRI refresh, SRI holdings refresh, and intraday succeeded. Warm EOD reported governed `degraded` with `bounded_provider_gap`.
- Market State and Risk/Reward aligned to the 2026-08-21 session with 132/132 symbols. SRI aligned to the 2026-08-21 session with 16 segments. Intraday had 132 symbols at `2026-08-21 22:15:00 UTC`.
- A run-now operations-monitor check completed successfully at `2026-08-22 06:07:24 UTC`.
- PR-0 and PR-1 are closed for the current pilot-readiness path.

## 2026-08-22 FMP recurring activation update

- The root-owned deployment agent was reprovisioned from the current source, and live `scheduler-status` now includes the warm-EOD and FMP annual scheduled-service rows.
- The FMP annual financial recurring timer is active:

```text
timer=signalops-marketops-boundary-fmp-annual-financial.timer load=loaded active=active next=Sat 2026-08-29 06:30:00 UTC
service=signalops-marketops-boundary-schedule@marketops-fmp-annual-financial.service load=loaded active=inactive result=success
```

- This closes the PR-4 activation blocker. The next evidence point is the first natural scheduled FMP annual run on Saturday, August 29, 2026 at 02:30 America/New_York.

## 2026-08-22 Review Queue / EROC projection closure

- Review Queue stale-date investigation found the composite queue was surfacing stale EROC reversal evidence from `subscriber_gateway_global_eroc_results`, whose latest session was `2026-08-14`.
- The EROC algorithm itself was not stale: `marketops_valuation_results` had 132 tenant-local EROC symbols for each trading session through `2026-08-21`. The stale layer was the subscriber global projection.
- Completed an append-only dedicated MarketOps catch-up for `2026-08-17` through `2026-08-21`, producing 132 global EROC symbols per session without provider polling or source-row mutation.
- The post-close global projection script now includes a constrained `valuation` projection for `signalops.algorithms.eroc_v6` and fails the post-close gate if `subscriber_gateway_global_eroc_results` has fewer symbols than the tenant-local EROC source for that session.
- The UI stale-evidence guard remains in place as a defensive presentation layer, but the pipeline now has a permanent EROC projection hook.
- Remaining validation: observe the next natural trading-day post-close run and confirm EROC global projection advances through the standard parity manifest/materializer path.
