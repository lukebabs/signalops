# Subscriber Project — Production Readiness Path

Status: active production-readiness control.

Last reviewed: 2026-08-21.

## Readiness position

The Subscriber Project is **controlled pilot-ready**, not full external production-ready.

The current platform has enough structure to keep validating with controlled tenants and named approvals: dedicated MarketOps databases are live, web and gateway are serving, SAF viability analytics are visible, and the global-data projection work has advanced materially.

Production readiness is still blocked by operational consistency gaps. The most important blocker is not feature breadth; it is the ability to prove every scheduled job completed, every view refreshed from the same authoritative MarketOps data plane, and every access/subscription decision was enforced consistently.

## Current evidence snapshot — 2026-08-21

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Web and Gateway availability | Ready | `signalops-web-1` and `signalops-gateway-1` were running; public `/readyz` returned `200`; public `/marketops/signal-assurance` returned `200`. |
| Dedicated MarketOps data boundary | Ready | `signalops-marketops-postgres-1` and `signalops-marketops-timescaledb-1` were healthy. MarketOps is intended to read/write the dedicated MarketOps databases, not the old shared MarketOps tables. |
| Completed-session global data | Mostly ready | SAF, SRI, Market State, and Risk/Reward global projections showed latest completed-session data for 2026-08-20. |
| Intraday freshness | Schedule-ready, market-idle | Latest intraday snapshot was 2026-08-20 22:15 UTC. At the review time, no 2026-08-21 market-session intraday data was expected yet. |
| Scheduled jobs | Ready / observing | `scheduler-status` returned clean on 2026-08-21 05:17 UTC. The next acceptance point is the natural 2026-08-21 post-close cycle. |
| Daily post-close | Observing | The stale failed systemd state was reconciled only after dedicated MarketOps DB evidence showed recovered completion. The next eligible post-close cycle must complete without manual reconcile before PR-0/PR-1 exit is final. |
| FMP annual financial job | Partial / PR-4 | FMP annual enrichment has a central worker, Admin-visible job status, freshness row, run-now action, and disabled Saturday schedule hook. PR-4 must decide when to enable recurring capture. |
| Deployment automation | Mostly ready | Production route checks and constrained Playwright smokes now pass. PR-1 Admin freshness acceptance corrected the false `/marketops/admin` check to the real `/admin/system` route. |
| SAF operational viability | Pilot-ready | SAF progression chart, 10/20-day filters, and inline drill-down are live. Historical viability is currently strongest for the tenant-local 132-asset legacy cohort and should continue maturing naturally unless a separate backtest gate is approved. |
| Subscription/access controls | Ready for configured QA identities | PR-2 closed tenant isolation, private-list owner projection, tier-enforcement canary, restoration, and Subscription Administration governance-surface browser evidence. |
| Backup/restore | Deferred risk | Dedicated pgBackRest backup and isolated restore rehearsal previously passed. PR-3 current re-verification is intentionally deferred by product decision and remains a known readiness risk. |

## Production gates

### P0 — Operational correctness gates

These must close before wider pilot or paid production.

1. **Daily post-close must end cleanly**
   - Fix the current service failure where completed core evidence is followed by fatal SRI canonical-normalization failure.
   - Either make SRI normalization complete from persisted canonical inputs without provider polling, or classify missing optional normalization as a warning while preserving explicit admin evidence.
   - Acceptance:
     - `sudo -n signalops-deploy-agent scheduler-status` returns clean status.
     - The next eligible post-close run exits successfully.
     - Admin status shows the job result, timestamps, and any warnings.
     - Dashboard, Market State, Risk/Reward, SRI, and SAF read the same completed market session.

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
   - Make job health, data freshness, and recovery controls visible in Admin Workbench.
   - Acceptance:
     - Admin can see each job's last run, next run, result, data session, row counts, warning/error reason, and run-now eligibility.
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
     - Admins can govern enrolled users, tenant membership, tier assignment, entitlement state, quota state, default-list policy, and audit evidence.
     - Billing/provider integration remains separated until explicitly approved.

10. **Incident runbooks**
    - Maintain runbooks for stale dashboard data, failed post-close, provider outage, access-control regression, failed deployment smoke, and backup/restore.
    - Acceptance:
      - Each runbook includes detection, owner, first response, recovery action, verification, and rollback criteria.

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
- Live evidence now shows the stale `marketops-daily-postclose` systemd failure was reconciled and `scheduler-status` returned clean service states. This closes the stale-state cleanup, but PR-0 still requires the next eligible post-close cycle to complete with `result=success` without needing reconcile.

### Sprint PR-1 — Make operations visible

Scope:

- Expose status control in Admin Workbench.
- Show job result, freshness, warnings, and run-now actions.
- Keep run-now behind constrained deployment-agent actions.

Exit:

- An administrator can determine whether Dashboard, Assets, Market State, Risk/Reward, SRI, SAF, and FMP data are fresh without shell access.

Implementation note — 2026-08-21:

- The first source slice added read-only data-freshness visibility to the Administration operations-health API and Admin Workbench for Dashboard alignment, Market State, Risk/Reward, SRI, SAF, and Intraday.
- The follow-on source slice added Assets coverage and FMP annual financial workflow freshness. PR-1 source coverage is now complete. Browser acceptance is automated and passed through `scripts/run_subscription_admin_ui_smoke.sh`; final live exit still requires post-close freshness evidence from the next eligible scheduled cycle.

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

- Govern FMP annual enrichment lifecycle.
- Add trading-calendar correctness. Scheduler non-trading-day skips and SAF UI 10/20-day filters now use matching 2026/2027 US market-holiday semantics.
- Complete subscriber administration workflow.
- Finalize incident runbooks.

Exit:

- Platform can support paid pilot tenants with secure access, consistent data, visible operations, and documented recovery paths.

Implementation note — 2026-08-21:

- PR-4 starts with controls, not broader provider polling. The first implementation added a reusable MarketOps trading-calendar primitive and wired scheduled jobs to skip configured US market holidays as well as weekends, while preserving the explicit maintenance/FMP weekend allowlist.

## Standing readiness check

Each production-readiness review should record:

1. Git revision and working-tree status.
2. Web, Gateway, dedicated MarketOps Postgres, and dedicated MarketOps Timescale container health.
3. Public `/readyz` and the key MarketOps browser routes.
4. `sudo -n signalops-deploy-agent scheduler-status`.
5. Latest completed market session and row counts for Dashboard, Assets, Market State, Risk/Reward, SRI, SAF, and FMP.
6. Latest intraday snapshot and hot-symbol count during market hours.
7. Subscription/access-control canary result.
8. Backup label and restore rehearsal result.
9. Open blockers classified as Ready, Partial, or Blocked.

## Next recommended action

Observe the next natural 2026-08-21 post-close cycle. If `marketops-daily-postclose`, `marketops-postclose-recovery`, Risk/Reward, SRI, SAF, Dashboard, and Assets freshness all align without manual reconcile, close PR-0/PR-1. PR-2 access/subscription hardening is closed for the configured QA identities. PR-3 is deferred by accepted risk, so the next active implementation gate is **Sprint PR-4 — Production expansion controls**. If it fails, treat the failure as the highest-priority production blocker.

## 2026-08-21 05:17 UTC readiness update

- Public `/readyz` returned `200`.
- Public `/admin/system` returned `200`.
- `sudo -n signalops-deploy-agent scheduler-status` returned clean timer/service state.
- PR-1 Admin freshness browser acceptance passed through Playwright: the test logs in as `luke@strategiclabs.io`, opens `/admin/system`, asserts the operations-health API response, and verifies all eight freshness labels render.
- The remaining time-gated acceptance item is the next natural post-close cycle at 2026-08-21 22:01:55 UTC, followed by recovery/SRI timers through the controlled post-close window.
