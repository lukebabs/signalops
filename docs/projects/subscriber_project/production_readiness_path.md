# Subscriber Project — Production Readiness Path

Status: controlled paid-pilot readiness is advanced but not complete. PR-0, PR-1, PR-2, PR-4, and PR-5 are closed for the configured QA identities. The September 4, 2026 UTC follow-up confirms the tenant-local B2C enrollment path, no-MFA enrollment posture, Pricing/Stripe readiness surface, and enrollment-only subscription activation gate are deployed and tested.

Last reviewed: 2026-09-04.

## Readiness position

The Subscriber Project is **controlled paid-pilot ready with first real paid activation evidence**, not full external production-ready.

The current platform has enough structure to keep validating with controlled tenants and named approvals: dedicated MarketOps databases are live, web and gateway are serving, SAF viability analytics are visible, and the global-data projection work has advanced materially.

Production readiness is still blocked by current backup/restore re-verification, Stripe Customer Portal/self-management, tax/invoice evidence review, and cleanup of known historical test artifacts. First real paid-flow activation evidence is now retained. The core operational-consistency loop for the tenant-local pilot scope has passed natural post-close acceptance after the MarketOps database decoupling.

## Current evidence snapshot — 2026-09-04

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Web and Gateway availability | Ready | Gateway and Web were rebuilt/restarted through constrained deployment-agent actions after the B2C activation gate. Built-in gateway smoke passed: `marketops_read_cutover_gateway_verified`; subscriber pilot UI smoke passed: `2 passed`. |
| Dedicated MarketOps data boundary | Ready | `signalops-marketops-postgres-1` and `signalops-marketops-timescaledb-1` were healthy. MarketOps is intended to read/write the dedicated MarketOps databases, not the old shared MarketOps tables. |
| Completed-session global data | Ready for pilot scope | Dedicated MarketOps DB shows 2026-08-31 latest rows: Market State 132, Risk/Reward 132, SRI 16, SAF 99, Annual VC 988, Annual DOSM 988. |
| Intraday freshness | Ready / market-idle | Latest intraday scheduled service result is success. Freshness remains governed by market-window semantics rather than requiring live snapshots outside the active monitor window. |
| Scheduled jobs | Ready with governed warning | `scheduler-status` returned active timers and success results for tracked services. `marketops-warm-eod` remained governed `degraded` with `bounded_provider_gap`, while post-close, intraday, recovery, SRI, holdings, and FMP annual completed successfully. |
| Daily post-close | Closed for PR-0 | The August 31 post-close cycle completed: `marketops-daily-postclose` succeeded at 2026-08-31 22:35:52 UTC and `marketops-postclose-recovery` succeeded at 2026-09-01 03:00:01 UTC. |
| FMP annual financial job | Active / current | Latest governed annual run succeeded on 2026-09-01 UTC after global EOD catch-up. Current user-facing annual VC/DOSM output skips assets without usable annual data instead of projecting partial ineligible rows. |
| Deployment automation | Mostly ready | Production route checks and constrained Playwright smokes now pass, including the controlled Syncratic Ask live smoke after AI Gateway price-catalog propagation. PR-1 Admin freshness acceptance corrected the false `/marketops/admin` check to the real `/admin/system` route. Syncratic Ask readiness is tracked in [Syncratic Ask Readiness Checklist](syncratic_ask_readiness_checklist.md), and Admin Operations Health now has a dedicated Syncratic Ask row that passed production browser validation on 2026-08-23. |
| Mobile subscriber UX | Closed for configured QA identities | `scripts/run_subscriber_mobile_ui_smoke.sh` validates production login and core subscriber routes at 375px, 390px, and 430px phone widths: Dashboard, Watchlists, Assets, Sector Rotation, Opportunities, Signal Assurance, Syncratic Intelligence, Pricing, Dashboard-to-Syncratic handoff, Assets mobile card drilldown, Opportunities mobile drilldown, SAF mobile drilldown, and SRI ETF progression/makeup drilldown. Read-only enrollment smoke also passed. The named mobile gated-route subscription-enforcement canary passed and restored production state. Latest result: mobile `24 passed in 253.48s` on 2026-09-01 after adding EEOM current/history regression coverage; enrollment `1 passed in 0.93s`; canary `subscription_enforcement_canary_verified` then `subscription_enforcement_canary_restored`. |
| SAF operational viability | Pilot-ready | SAF progression chart, 10/20-day filters, and inline drill-down are live. Historical viability is currently strongest for the tenant-local 132-asset legacy cohort and should continue maturing naturally unless a separate backtest gate is approved. |
| Subscription/access controls | Ready for configured QA identities / first paid activation closed | PR-2 closed tenant isolation, private-list owner projection, tier-enforcement canary, restoration, and Subscription Administration governance-surface browser evidence. B2C users remain in `tenant-local`; self-enrolled subjects without an effective subscription resolve to `subscription_missing` and route to Pricing. Pricing reads the configured Stripe product catalog, displays governed human-readable prices instead of raw Stripe Price IDs, Checkout-start is operational, and the first live Explorer monthly payment reconciled through a signed webhook into `explorer active` with B2C browser state `marketops_ready`. Admin-governed refund intake is now live: users can request refunds, admins triage/record disposition, and actual refund execution remains in Stripe Dashboard. Stripe Customer Portal self-service is implemented for active Stripe-backed subscriptions, with webhook-authoritative entitlement updates retained. |
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
   - Current status: ready for configured QA identities, including the `000160` upgrade-intent closure and the September 4 B2C activation-gate closure. Playwright verified pilot Explorer pricing, Stripe price display, authenticated upgrade-interaction persistence, tenant-filtered Admin Upgrade funnel evidence, Stripe webhook fail-closed/signed-canary behavior, Checkout-start readiness, and B2C subscription-missing routing.
   - Re-run enforcement with tenant-local admin and tenant-pilot-b subscriber identities.
   - Validate Explorer, Professional, and Institutional entitlement behavior.
   - Validate cross-tenant denial, admin-only settings, tenant-default list behavior, and private-list ownership.
   - Acceptance:
     - No tenant can read or mutate another tenant's private objects.
     - Non-admin users cannot perform admin duties.
     - Subscription enforcement can be enabled, tested, and restored without residue.

5. **Admin operations status control**
   - Status: closed for the current pilot path after Admin Operations Health, Dashboard freshness, and the actionable freshness-contract browser smokes passed.
   - Deployment-agent expansion is installed: `scheduler-status` lists warm-EOD, FMP, retention, storage, and core MarketOps scheduled-service rows.
   - Acceptance evidence:
     - Admin can see core job/data freshness without shell access.
     - Each core freshness row now exposes its expected freshness contract, producing dependency job, dependency status/schedule, latest evidence, coverage count, reason/next-step guidance, and only the constrained run-now action allowed for that view.
     - Run-now uses constrained deployment-agent actions, not broad shell access. Unsupported actions, including SAF projection refresh, remain status-only in the UI.
     - Production validation on 2026-08-24 passed: targeted Go API tests, web build, gateway/web deploy, `/readyz`, subscriber smoke, and Admin Operations Health Playwright/API smoke.

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
     - Admins can govern enrolled users, tenant membership, tier assignment, entitlement state, quota state, default-list policy, audit evidence, user activity visibility, and refund-request intake.
     - Billing/provider integration remains separated until explicitly approved.
   - Current status: migration `000157_subscriber_user_activity_ledger` is applied in the dedicated MarketOps database and gateway/web are deployed. Subscription Administration now exposes append-only login/logout/feature-view/mutation activity through an Activity tab and per-user drilldown. Migration `000158_subscriber_user_activity_retention_policy` is applied and provides a 180-day dry-run retention policy for activity detail rows; enforcement remains a separate approval gate after product/legal retention and summarized activity needs are confirmed. Migration `000166_subscriber_refund_requests` adds admin-governed refund intake and email-first identity rendering for subscription operators; actual Stripe refunds remain manual/admin-only pending a separately approved refund executor/reconciliation sprint.

10. **Incident runbooks**
    - Maintain runbooks for stale dashboard data, failed post-close, provider outage, access-control regression, failed deployment smoke, backup/restore, and FMP annual degradation.
    - Acceptance:
      - Each runbook includes detection, owner, first response, recovery action, verification, and rollback criteria.
    - Current PR-4 evidence: `pr4_incident_runbooks_2026-08-21.md` defines all required response paths using constrained deployment-agent actions and dedicated MarketOps database evidence.

11. **Mobile subscriber readiness**
   - Status: planned sprint; Admin remains out of scope.
   - The product must be validated for primarily mobile subscribers before paid production.
   - Acceptance:
     - Mobile Playwright suite validates subscriber routes at 375px and 430px widths.
     - Dashboard, Watchlists, Assets, Market State, Value Intelligence, Distressed Opportunity Intelligence, Earnings Opportunity Intelligence, Opportunities, SRI, SAF, Syncratic, Pricing, and enrollment flows have no blocking mobile usability defects.
     - No page-level horizontal overflow, clipped primary actions, unreadable chart labels, or inaccessible drilldown close/back controls.
     - Admin Workbench and Subscription Administration remain desktop/operator workflows unless a separate sprint is approved.

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

Implementation note — 2026-08-24:

- Admin Operations Health now includes an explicit freshness contract for each core view: expected freshness, dependency job, dependency status/schedule, latest evidence, coverage, reason/next-step explanation, and constrained action.
- Actionability is intentionally bounded. Dashboard/Assets/Market State use the post-close recovery guard, Risk/Reward uses the Risk/Reward run-now action, SRI uses the SRI run-now action, Intraday uses the intraday monitor, and FMP uses the annual-financial action. SAF and Syncratic Ask remain status-only because their refresh paths require separate named approval or narrative-specific controls.
- Production browser/API validation passed against `/admin/system` after gateway/web deployment.

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

### Sprint PR-5 — Mobile subscriber readiness

Status: closed for the configured production QA identities on 2026-09-01.

Scope:

- Add a dedicated mobile browser acceptance suite for subscriber-facing MarketOps routes.
- Remediate phone-width usability defects in Dashboard, Watchlists, Assets, Market State, Valuation/DOSM, EROC, EEOM, Opportunities, SRI, SAF, Syncratic, Pricing, and enrollment.
- Keep Admin and operator controls desktop-scoped.
- Preserve a future path to PWA/native app viability without starting native development.

Exit:

- The mobile suite passes in production against the configured subscriber QA identity at 375px and 430px widths.
- Failure-only HAR/trace/screenshots are retained under the protected artifact policy.
- Product readiness records subscriber mobile web as accepted or lists only non-blocking polish items.

Closure evidence:

- Mobile subscriber route/drilldown/pricing validation passed at 375x812, 390x844, and 430x932. The suite includes Earnings Opportunity Intelligence route coverage and EEOM current/history drill-down behavior.
- Latest regression rerun after the enrollment no-MFA policy update: `scripts/run_subscriber_mobile_ui_smoke.sh` returned `24 passed in 48.56s` on 2026-09-02.
- Read-only Keycloak enrollment smoke passed: `1 passed` on 2026-09-02.
- Authenticated B2C enrollment resolver smoke passed: `1 passed` on 2026-09-02, confirming the active production flow reaches the SignalOps enrollment resolver without SMS/MFA friction.
- Enrollment production-polish rerun passed after removing stale SMS language from the Keycloak registration theme: read-only registration smoke `1 passed`; authenticated B2C resolver smoke `1 passed`.
- Named temporary production subscription-enforcement canary passed at mobile viewport and restored production state: Explorer denied Value Intelligence, Sector Rotation remained open, Professional unlocked Value Intelligence, Signal Assurance remained Institutional-only, tenant-local Institutional/admin access remained valid, and wrapper emitted `subscription_enforcement_canary_verified` followed by `subscription_enforcement_canary_restored`.

## Standing readiness check

Each production-readiness review should record:

1. Git revision and working-tree status.
2. Web, Gateway, dedicated MarketOps Postgres, and dedicated MarketOps Timescale container health.
3. Public `/readyz` and the key MarketOps browser routes, including the Syncratic Ask smoke when Syncratic code, Gateway deployment, or AI Gateway policy/catalog changed. Use [Syncratic Ask Readiness Checklist](syncratic_ask_readiness_checklist.md) as the control record.
4. `sudo -n signalops-deploy-agent scheduler-status`.
5. Latest completed market session and row counts for Dashboard, Assets, Market State, Risk/Reward, SRI, SAF, and FMP.
6. Latest intraday snapshot and hot-symbol count during market hours.
7. Subscription/access-control canary result.
8. Backup label and restore rehearsal result.
9. Open blockers classified as Ready, Partial, or Blocked.
10. Mobile subscriber smoke result when user-facing layout, navigation, enrollment, Syncratic, SAF, SRI, Dashboard, Assets, or Pricing changed.

## Next recommended action

Move to the remaining paid-pilot production work:

1. Complete one controlled Stripe paid-flow validation only when ready to create a real subscription: Checkout success, signed webhook reconciliation, effective subscription activation, and return-to-context behavior.
2. Refresh PR-3 backup/restore evidence before broader paid pilot expansion.
3. Decide whether to keep historical `tenant-b2c` test access rows as retained audit evidence or archive them through a governed cleanup path.
4. Harden the parent Keycloak SMS/MFA reconcile source so future auth deployments preserve the current no-MFA enrollment posture unless MFA is explicitly re-approved.

## 2026-08-21 05:17 UTC readiness update

- Public `/readyz` returned `200`.
- Public `/admin/system` returned `200`.
- `sudo -n signalops-deploy-agent scheduler-status` returned clean timer/service state.
- PR-1 Admin freshness browser acceptance passed through Playwright: the test logs in as `luke@strategiclabs.io`, opens `/admin/system`, asserts the operations-health API response, and verifies the required freshness labels render. The current contract includes nine rows after the Syncratic Ask operations-health extension.
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
- 2026-08-26 source update: the post-close global projection gate now also materializes and verifies core Valuation Intelligence (`signalops.algorithms.valuation_composite_v3`), Distressed Opportunity Intelligence (`signalops.algorithms.distressed_opportunity_scoring_v3`), and EEOM evidence for the exact completed session. The gate fails closed if `subscriber_gateway_global_valuation_results` or `subscriber_gateway_global_eeom_results` trails the tenant-local source for that session. This makes Valuation/DOSM/EEOM part of the same global analytical data-plane parity contract as Market State, Risk/Reward, SAF outcomes, and EROC.

## 2026-08-23 Syncratic Ask operations-health deployment validation

- Deployed the Gateway through the constrained deployment-agent action `signalops-production-gateway-deploy`. The action rebuilt the Gateway image and returned `signalops_public_production_deploy_verified`.
- The deployment-agent bundled subscriber smoke hit one transient Chromium `net::ERR_NETWORK_CHANGED` during initial navigation; public readiness and the targeted Admin smoke were used as the slice-specific acceptance evidence.
- Public `/readyz` returned `200` with service `signalops-gateway`.
- `scripts/run_subscription_admin_ui_smoke.sh` passed: `3 passed in 11.94s`. This validates the Administration Operations Health API/browser path and requires the `Syncratic Ask` freshness row.

## 2026-08-24 operations-monitor WAL probe closure

- Investigated repeated `marketops-operations-monitor` failures on August 24. The failure was `wal_temporal`: temporal archive age exceeded the 1800-second threshold even after the monitor requested a WAL switch.
- Root cause: the dedicated temporal database is low-write and can sit at a WAL segment boundary. `pg_switch_wal()` alone may not create a new archivable segment in that state, so the monitor observed stale `pg_stat_archiver.last_archived_time` despite healthy pgBackRest configuration and zero archive failures.
- Source fix: `scripts/marketops_operations_monitor.sh` now writes a minimal committed WAL heartbeat with `SELECT txid_current()` before `pg_switch_wal()` and polls archive freshness for up to 60 seconds.
- Validation: `bash -n scripts/marketops_operations_monitor.sh`, `sudo -n signalops-deploy-agent scheduler-run-now:marketops-operations-monitor`, DB status `marketops-operations-monitor=succeeded` at `2026-08-24 13:17:09 UTC`, and `sudo -n signalops-deploy-agent scheduler-status` returned clean.


## 2026-08-30 production-readiness update — deploy recovery and FMP annual queue

Status: application availability restored; FMP annual lifecycle remains partial until the queued annual-financial tasks drain or are explicitly classified.

Evidence:

- Gateway deploy recovered through the constrained deployment-agent path and returned `marketops_read_cutover_gateway_verified`.
- Gateway startup evidence confirmed MarketOps reads use the dedicated MarketOps data boundary.
- Web/proxy deploy recovered through `marketops-web-deploy`.
- Public `/readyz` returned `200`.
- Subscriber pilot UI smoke passed with `2 passed`.
- FMP annual v2 evidence exists in the dedicated MarketOps database, but the task queue still contains queued/running/deferred work.

Control update:

- Added source action `marketops-fmp-systemd-reconcile` to avoid raw operator `systemctl daemon-reload` / `reset-failed` for the FMP annual service.
- The action is DB-evidence-gated and resets only the FMP annual scheduler service after the dedicated MarketOps database proves an FMP annual workflow and v2 evidence records exist.

Remaining acceptance before marking this slice fully production-ready:

1. Reprovision the root-owned deployment agent from the current source.
2. Run `sudo -n signalops-deploy-agent marketops-fmp-systemd-reconcile`.
3. Run `sudo -n signalops-deploy-agent scheduler-status` and confirm no failed service state remains.
4. Continue the FMP annual task workflow until queued/running tasks are drained or classified as provider quota/no-data/terminal failure.


## 2026-08-30 production-readiness update — FMP annual identity fix verified

The FMP annual lifecycle blocker was reduced from scheduler failure to governed data-quality/provider exceptions.

Root cause fixed:

- Annual financial v2 evidence IDs did not include algorithm version/session identity, which caused collisions against earlier v1 immutable evidence when FMP returned unchanged payloads.
- Annual VC/DOSM materialized evidence IDs did not include observation date, which caused cross-session collisions when valuation payloads repeated.

Validation:

- Targeted Go package tests passed for the FMP annual task worker, annual valuation materializer, and FMP adapter.
- `sudo -n signalops-deploy-agent marketops-fmp-annual-run` completed successfully after the fixes.
- `sudo -n signalops-deploy-agent scheduler-status` returned clean.
- Public `/readyz` returned `200`.
- Dedicated MarketOps evidence for `2026-08-28`: `1000` `fundamental_annual` v2 records, `879` annual VC records, and `879` annual DOSM records.

Remaining FMP annual acceptance:

- Review the six provider quota deferrals and one no-data symbol as governed data-quality/provider exceptions.
- Keep the next recurring run scheduled for Saturday, September 5, 2026 at 06:30 UTC / 02:30 America/New_York.


## 2026-08-30 production-readiness update — FMP class-share normalization verified

The residual FMP annual exceptions were reduced from seven to one.

Root cause fixed:

- FMP expects common share-class request symbols with hyphen notation, while the MarketOps catalog keeps exchange-style dot notation. The adapter now converts dot notation to hyphen notation only at the FMP request boundary.

Validation:

- FMP adapter tests passed for annual and TTM fundamentals class-share normalization.
- The seven known exception tasks were narrowly requeued for workflow `subglobalannualworkflow-20260828`.
- `BF.A`, `BF.B`, `BRK.B`, `HEI.A`, `MOG.A`, and `MOG.B` succeeded after the adapter fix.
- `BXBL` remains the only skipped symbol because FMP returned an empty annual-financial response.
- Workflow coverage is now `999` succeeded and `1` skipped no-data.
- `sudo -n signalops-deploy-agent scheduler-status` returned clean, and public `/readyz` returned `200`.

Remaining FMP annual acceptance:

- Decide whether `BXBL` should remain eligible with a provider no-data classification or be excluded/normalized through a separate catalog-governance policy.

## 2026-09-01 production-readiness update — annual valuation and catalog hygiene follow-up

Status: annual financial refresh completed successfully; global EOD catch-up and annual valuation rematerialization restored VC/DOSM revenue-growth evidence; tactical projection duplicate inflation is fixed.

Evidence:

- The corrected `000162_subscriber_global_valuation_tactical_projection` view was applied to the dedicated MarketOps database.
- Tactical posture projection for the latest materialized valuation session now emits one row per canonical symbol: `131` rows for `2026-08-31`, with `0` duplicate symbol rows and `max_rows_per_symbol=1`.
- The duplicate root cause was catalog identity drift: some canonical symbols exist more than once in `subscriber_global_assets`. The projection now deterministically selects one canonical global asset per uppercase symbol so downstream Valuation/DOSM cards are not inflated while the catalog is cleaned.
- `marketops-fmp-annual-run` completed for workflow `subglobalannualworkflow-20260901` with coverage `{"succeeded": 999, "skipped_no_data": 1}`.
- The stale global EOD evidence plane was the root cause for annual v4 `insufficient_data`: before catch-up, global `eod_bar` evidence stopped at `2026-08-14`, so the annual materializer could not derive market cap from close × shares.
- `subscriber-global-eod-history-materialize` caught the global EOD plane up through `2026-08-31` for `999` assets and inserted `10969` missing EOD records.
- Rerunning the governed annual action after EOD catch-up produced `997` annual VC rows and `997` annual DOSM rows for `2026-08-31`; `956` of each include `revenue_cagr_3y_annual`.
- Browser/API validation as `luke@strategiclabs.io` confirmed `/marketops/valuation` loads, tenant-local valuation API returns platform-global AAPL annual DOSM v4 with CAGR and updated revenue-growth score, tactical posture is present, and the rendered page showed `Unknown` count `0`.
- Raw compose execution remains intentionally constrained by protected MarketOps database secrets. The deployment-agent `render-cutover-env` action currently references a missing installed library path and must be repaired before it can replace all manual protected-env compose paths.

Remaining acceptance:

1. Monitor the remaining governed FMP no-data/partial-source exceptions; user-facing annual VC/DOSM projections now skip assets without usable annual data instead of displaying partial analytical rows.


## 2026-09-01 production-readiness update — catalog identity projection cleanup

Status: duplicate canonical-symbol projection is governed without deleting catalog rows or rewriting immutable evidence.

Evidence:

- Applied migration `000163_subscriber_global_catalog_identity_projection` to the dedicated MarketOps database.
- The existing `subscriber_global_asset_identity_resolutions` table already resolved all 100 duplicated canonical symbols to one canonical global asset ID per symbol.
- Added `subscriber_gateway_global_canonical_assets` as the gateway-facing canonical catalog projection. It exposes one row per canonical symbol and classifies duplicate groups as `deduplicated`.
- Updated `subscriber_gateway_global_valuation_results` to resolve valuation evidence through canonical asset identity before projection.
- Annual v4 Valuation/DOSM projection now excludes ineligible or non-usable annual rows. Immutable partial evidence remains stored for audit, but assets without usable annual data are skipped in user-facing annual analytical output.
- Verification: canonical projection returned `1402` canonical rows and `100` deduplicated rows, with `0` duplicate projected symbols. Existing `2026-08-31` annual DOSM evidence retained `9` partial records, while projected partial annual rows were `0`.

Remaining acceptance:

1. Continue observing the next natural post-close cycle to confirm canonical projections remain aligned across Dashboard, Assets, Market State, Risk/Reward, SAF, EROC, EEOM, and Material Events.


## 2026-09-01 production-readiness update — subscriber gateway canonical projection expansion

Status: core subscriber global gateway projections now use canonical asset identity resolution.

Evidence:

- Applied migration `000164_subscriber_global_gateway_canonical_projection` to the dedicated MarketOps database.
- Updated Market State, EROC, EEOM, Material Events, Options distributions, Risk/Reward, Intraday Current State, Signal Assurance observations, and global evidence coverage to resolve source asset IDs through the canonical projection.
- No catalog rows or immutable evidence rows were deleted or rewritten.
- Verification returned `0` duplicate projected symbols across all updated views.
- Existing subscriber pilot Playwright smoke passed: `2 passed`.

Remaining acceptance:

1. Observe the next natural post-close cycle and confirm freshness/readiness remains aligned with the canonical projections.


## 2026-09-01 production-readiness update — deployment-agent render closure

Status: deployment-agent `render-cutover-env` live install gap is closed.

Evidence:

- Reprovisioned the root-owned deployment agent from the repaired source.
- `sudo -n signalops-deploy-agent render-cutover-env` succeeded and rendered `/etc/signalops/marketops-cutover.env`.
- `sudo -n signalops-deploy-agent scheduler-status` returned all tracked MarketOps timers active and all tracked services with `result=success`.

Remaining acceptance:

1. Observe the next natural post-close cycle and confirm freshness/readiness remains aligned with the canonical projections.

## 2026-09-01 production-readiness update — EEOM current/history boundary

Status: Earnings Opportunity Intelligence drift is fixed and the history boundary is explicit.

Evidence:

- Root cause: EEOM preserved point-in-time snapshots by earnings event ID, and provider event-date revisions could surface multiple rows for one canonical ticker in the default subscriber view. GTLB showed the failure mode: a newer `2026-09-01` bullish row and an older superseded `2026-09-02` bearish row.
- Gateway/API fix: default EEOM responses collapse to one current row per ticker. Historical rows remain available only through explicit `include_history=true` / `history=true` requests.
- Storage fix: tenant and subscriber-global EEOM readers deduplicate before `LIMIT`, preventing duplicate-heavy symbols from crowding out valid tickers.
- UI fix: the default Earnings Opportunity Intelligence table remains current-only, while selecting a row reveals an `Earnings setup evolution` panel with the preserved point-in-time history for that asset.
- Regression coverage: Go API tests validate the duplicate/conflicting-row collapse, and the subscriber Playwright smoke now asserts that the default EEOM response has no duplicate ticker rows and that the row-level evolution panel renders.
- Deployment evidence: gateway was rebuilt/restarted after commit `6a82d3a`; Playwright closure passed after commit `6ffe07b`; the current web slice passes local Vitest and production build before deployment.

Remaining acceptance:

1. After the next natural post-close cycle, rerun the subscriber Playwright smoke to confirm current-only EEOM rows and history drill-down remain aligned with refreshed data. The post-change mobile gate passed on 2026-09-01: `24 passed in 253.48s`.
2. If analysts need more than the compact inline history, add a dedicated EEOM evolution route or chart rather than expanding the default table.

## 2026-09-01 production-readiness update — remaining production gates

Status: controlled pilot-ready; not broad paid-production ready.

Remaining gated work:

1. **Post-close data freshness proof** — validate the September 1, 2026 ET post-close cycle after warm EOD, daily post-close, recovery guard, SRI refresh, and SRI holdings refresh complete.
2. **Subscription enforcement hardening** — rerun the temporary production enforcement canary when tier or gated-route behavior changes; keep automatic restoration to enforcement-off/pilot Explorer state for canaries.
3. **Enrollment and SMS MFA** — complete a full Keycloak-owned registration/MFA smoke with the production branding/disclaimer path and confirm existing users are routed to login rather than duplicate enrollment.
4. **Stripe billing** — validate one Stripe test-mode Explorer and one Professional subscription with automatic tax enabled, signed webhook reconciliation, and return-to-context behavior.
5. **Operations health in Admin** — keep expanding Admin-visible freshness/job status so routine validation does not require shell access or HAR handoffs. Current Sep 2 slice exposes `MarketOps operations monitor` as a constrained Admin run-now action, validates it through Playwright, and verifies the reprovisioned deployment-agent `scheduler-status` includes the operations-monitor timer/service.
6. **Mobile subscriber UX** — rerun the mobile Playwright suite after any Dashboard, Assets, EEOM, SAF, SRI, Syncratic Intelligence, Pricing, or enrollment layout change.
7. **Recovery evidence** — PR-3 remains a deferred risk until a current dedicated MarketOps backup and isolated restore rehearsal is rerun against the present schema/runtime.

## 2026-09-03 production-readiness update — Stripe Checkout runtime readiness

Status: Stripe Checkout-start is operational for Explorer monthly and Professional annual; full paid-flow activation remains gated on a controlled payment/webhook validation.

Evidence:

- Dedicated MarketOps database contains Explorer and Professional Stripe Product/Price mappings.
- Running Gateway has `STRIPE_WEBHOOK_SECRET` and Checkout success/cancel URLs present.
- Running Gateway has empty `STRIPE_API_KEY` and `STRIPE_RESTRICTED_API_KEY`, so Checkout startup returns `stripe_checkout_disabled` before any Stripe call.
- Gateway now exposes `checkout_enabled` from `/v1/marketops/subscription-products`, and Pricing disables Checkout controls with an explicit warning when the key is absent.
- Read-only Pricing readiness smoke passed: `scripts/run_stripe_checkout_readiness_ui_smoke.sh` returned `1 passed`.

Closure evidence — 2026-09-04:

- Stripe product/price IDs saved in Admin resolved under the live runtime key.
- After Stripe product tax-code configuration was corrected, `scripts/run_stripe_checkout_canary.sh` passed and created two unpaid Checkout Sessions.
- Dedicated MarketOps ledger rows show `checkout_started`, `checkout_url_returned=true`, populated Stripe session IDs, and empty `stripe_subscription_id`, proving Checkout-start without entitlement activation.

Remaining acceptance:

1. Complete one controlled paid-flow validation only when ready to create a real subscription.
2. Verify the signed Stripe webhook reconciles the matching opaque `checkout_ref` to an active subject subscription.
3. Verify Stripe invoice tax output in Stripe Dashboard.

## 2026-09-02 production-readiness update — Sep 1 post-close validation and warm-EOD observability

Status: September 1 post-close cycle completed; warm-EOD remains governed degraded, and the next natural run proved structured detail persistence.

Evidence:

- `scheduler-status` showed all tracked MarketOps timers active and tracked services inactive with `result=success`.
- Dedicated MarketOps job status showed successful September 1/2 UTC completion for daily post-close, post-close recovery, SRI refresh, SRI holdings refresh, intraday, and FMP annual financial.
- `marketops-warm-eod` completed as `degraded` with `reason=bounded_provider_gap`; this is the governed no-bar gap state previously accepted for the 1,000-symbol warm cohort.
- Observability gap: the warm-EOD status row had `detail={}`, so the Admin surface could not expose the specific normalized/expected/missing-symbol evidence.
- Source fix: `scripts/marketops_scheduled_job.sh` now captures warm-EOD normalization output and persists structured detail for degraded/incomplete runs. This does not change provider polling, success/failure semantics, or the bounded-provider-gap policy.
- Browser validation after the post-close cycle passed: `scripts/run_marketops_dashboard_freshness_ui_smoke.sh` returned `1 passed`; `scripts/run_subscriber_pilot_ui_smoke.sh` returned `2 passed`.

Closure evidence — 2026-09-03:

- The next natural warm-EOD run persisted structured detail for the September 2, 2026 session: `status=degraded`, `reason=bounded_provider_gap`, `normalized=996`, `expected=1000`, `missing=4`, `max_missing=5`, `session_date=2026-09-02`, and `missing_symbols=AVB,CRNX,EQR,WBS`.
- Admin Operations Health now renders a scheduled-job `Details` column, and the Admin Playwright smoke asserts the warm-EOD row exposes bounded-provider-gap details when the row is degraded.
- Validation passed: web build, Python syntax check, `go test ./internal/api`, production web deploy with subscriber smoke `2 passed`, and Admin Playwright smoke `3 passed, 1 skipped`.

Remaining policy:

1. Continue treating repeated bounded provider gaps as governed degraded unless missing count exceeds policy or affects user-facing analytical freshness.



### 2026-09-04 Stripe Customer Portal closure

Stripe Customer Portal self-service closes the customer-managed billing-session path for active Stripe-backed subscriptions. SignalOps creates portal sessions server-side, returns subscribers to Pricing, and relies on signed Stripe webhooks for any entitlement/status mutation. Manual/non-Stripe subscriptions fail closed instead of exposing a misleading self-service control.
