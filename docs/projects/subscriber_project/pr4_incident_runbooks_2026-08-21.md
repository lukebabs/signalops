# PR-4 Production Incident Runbooks — 2026-08-21

Status: source runbooks ready. These runbooks define the operator response pattern for controlled pilot and production-readiness gates. They do not by themselves authorize provider polling, destructive database action, or broad shell access.

## Operating rules

- Prefer the constrained deployment agent over direct `systemctl` or shell scripts.
- Treat the dedicated MarketOps databases as authoritative for MarketOps data freshness and scheduled-job status.
- Do not delete, reset, or rebuild live databases as a recovery shortcut.
- Do not widen provider polling during incident response unless a named approval explicitly permits it.
- Capture the git revision, command output, timestamps, and the final verification result in the build journal or incident record.

## Common triage commands

```bash
sudo -n signalops-deploy-agent scheduler-status
sudo -n signalops-deploy-agent operations-monitor-run
curl -fsS https://signalops.syncratic.io/readyz
```

Use Admin Workbench first when available:

- Admin > System / Operations Health
- Scheduled Jobs
- Data Freshness
- Subscription Administration for access/tier issues

## Runbook 1 — stale Dashboard or cross-view freshness drift

Detection:

- Admin Operations Health shows Dashboard, Market State, Risk/Reward, SRI, SAF, Assets analytical coverage, Intraday, or FMP annual financials as `stale`, `partial`, or `missing`.
- Browser views disagree on latest completed session.
- User reports stale dates after a completed market session.

Owner:

- MarketOps production operator.

First response:

1. Confirm public availability:

   ```bash
   curl -fsS https://signalops.syncratic.io/readyz
   ```

2. Confirm scheduler state:

   ```bash
   sudo -n signalops-deploy-agent scheduler-status
   ```

3. Run the operations monitor once:

   ```bash
   sudo -n signalops-deploy-agent operations-monitor-run
   ```

Recovery action:

- If the failed source is a scheduled job with an approved run-now action, run only that constrained job:

  ```bash
  sudo -n signalops-deploy-agent scheduler-run-now:<job_id>
  ```

- If post-close artifacts are incomplete, use the post-close recovery guard instead of manually starting individual internal stages:

  ```bash
  sudo -n signalops-deploy-agent scheduler-run-now:marketops-postclose-recovery
  ```

- If the systemd failed state is stale but the dedicated database proves the work recovered, use the constrained reconcile action:

  ```bash
  sudo -n signalops-deploy-agent marketops-postclose-systemd-reconcile
  ```

Verification:

- `scheduler-status` returns clean.
- Admin Operations Health freshness rows are current or explicitly degraded with reasons.
- Dashboard, Assets, Market State, Risk/Reward, SRI, and SAF show the same latest completed-session date where applicable.

Rollback criteria:

- If recovery worsens freshness or produces contradictory session dates, stop run-now actions and leave the system in read-only observation until the specific failed job logs and database evidence are reviewed.

## Runbook 2 — failed daily post-close or recovery guard

Detection:

- `scheduler-status` shows failed `marketops-daily-postclose` or `marketops-postclose-recovery`.
- Admin Scheduled Jobs shows failed/degraded post-close status.
- Dashboard remains on the prior completed session after the expected post-close window.

Owner:

- MarketOps production operator.

First response:

1. Check if the date is a non-trading day. Weekend/holiday skips are expected.
2. Confirm the failure is not only stale systemd state by reviewing Admin Operations Health and scheduler status.

Recovery action:

- Run the bounded recovery guard:

  ```bash
  sudo -n signalops-deploy-agent scheduler-run-now:marketops-postclose-recovery
  ```

- If the database evidence shows post-close, Risk/Reward, and SRI recovered but systemd still reports failed, reconcile failed state:

  ```bash
  sudo -n signalops-deploy-agent marketops-postclose-systemd-reconcile
  ```

Verification:

- `marketops-daily-postclose`, `marketops-postclose-recovery`, `marketops-risk-reward`, and `marketops-sri-refresh` status are `succeeded`, `degraded`, or otherwise explicitly allowed by the reconcile guard.
- Admin Operations Health no longer reports unexplained Dashboard drift.

Rollback criteria:

- Do not manually reset failed systemd state unless the constrained reconcile action accepts the database evidence.
- If reconcile refuses, treat the refusal as the source of truth and investigate the underlying failed job.

## Runbook 3 — provider outage or provider schema drift

Detection:

- Provider-backed jobs degrade/fail across many symbols.
- FMP annual financial tasks defer/fail in clusters.
- Massive/FMP provider responses show timeout, entitlement, rate-limit, or unexpected payload shape errors.

Owner:

- MarketOps data-plane operator.

First response:

1. Confirm whether failures are broad or symbol-specific.
2. Do not increase retry count or widen polling scope during the incident.
3. Preserve provider error output and task counts.

Recovery action:

- Let failure-isolated workers finish as `degraded` when possible.
- Run only approved retry/recovery actions after provider availability returns:

  ```bash
  sudo -n signalops-deploy-agent scheduler-run-now:marketops-task-retry
  ```

- For FMP annual financials, prefer waiting for the next scheduled run unless the business needs current fundamentals and named approval exists for run-now.

Verification:

- Admin Operations Health shows degraded/succeeded with explicit counts.
- Failed/deferred symbols are visible and not silently omitted.
- Provider API use remains within entitlement policy.

Rollback criteria:

- Disable the affected recurring timer if the provider keeps failing and the failure creates noise or budget risk. For FMP annual:

  ```bash
  sudo -n signalops-deploy-agent scheduler-fmp-annual-disable
  ```

## Runbook 4 — access-control or subscription regression

Detection:

- Tenant-pilot-b can read tenant-local, or tenant-local can read pilot private objects.
- Non-admin users can access admin settings or Subscription Administration.
- Subscription enforcement denies an entitled user or permits a non-entitled feature.

Owner:

- Platform access-control operator.

First response:

1. Capture the affected route, tenant id, user email, status code, and error code.
2. Do not change Keycloak mappers or roles without recording the exact intended correction.
3. Run the read-only access smoke where configured:

   ```bash
   scripts/run_subscriber_access_control_ui_smoke.sh
   scripts/run_subscription_admin_ui_smoke.sh
   ```

Recovery action:

- If the issue is identity configuration, correct the user/role/tenant claim in Keycloak and retest.
- If the issue is application authorization, disable the affected feature flag or route exposure until the regression is fixed.
- If a temporary enforcement canary caused state drift, restore enforcement-off and the pilot Explorer baseline using the canary’s restoration path.

Verification:

- Cross-tenant requests return `403 tenant_mismatch`.
- Non-admin users receive `403 insufficient_role` for admin-only surfaces.
- Subscription tier behavior matches Explorer, Professional, and Institutional policies.

Rollback criteria:

- Roll back the gateway/web deployment or disable subscription enforcement if access behavior cannot be proven correct quickly.

## Runbook 5 — failed deployment smoke or 404 after deploy

Detection:

- `/readyz` fails.
- Public app route returns the SPA shell but renders Not Found for an expected authenticated route.
- Deployment-agent smoke fails.
- User reports 404 immediately after login.

Owner:

- Release operator.

First response:

1. Determine whether the failure is public routing, authenticated route authorization, or stale browser bundle.
2. Do not treat a route-shell `200` as a successful protected-page smoke by itself.

Recovery action:

- Use the constrained deployment path for the impacted component:

  ```bash
  sudo -n signalops-deploy-agent signalops-production-web-deploy
  sudo -n signalops-deploy-agent signalops-production-gateway-deploy
  ```

- If only MarketOps web assets are affected:

  ```bash
  sudo -n signalops-deploy-agent marketops-web-deploy
  ```

Verification:

- `/readyz` returns `200`.
- `/admin/system` and key MarketOps routes load for the configured QA identities.
- Playwright smoke validates rendered content, not only HTTP status.

Rollback criteria:

- Revert to the last known-good deployment if route rendering or authorization cannot be restored with a safe redeploy.

## Runbook 6 — backup/restore concern

Detection:

- pgBackRest timer fails.
- Backup freshness is unknown.
- A destructive migration or incident requires restore confidence.

Owner:

- Recovery operator.

First response:

1. Confirm current production impact before running any recovery action.
2. Do not restore over live databases during rehearsal.
3. Prefer isolated restore rehearsal.

Recovery action:

```bash
sudo -n signalops-deploy-agent backup-run
sudo -n signalops-deploy-agent restore-rehearsal-run
```

Verification:

- Backup completes and reports a current label.
- Restore rehearsal starts isolated databases and accepts validation queries.
- Live MarketOps containers remain untouched by rehearsal.

Rollback criteria:

- If backup or restore rehearsal fails, treat production readiness as blocked for wider rollout until recovery evidence is current.

## Runbook 7 — FMP annual financial degradation

Detection:

- Admin Operations Health shows `FMP annual financials` as `partial`, `missing`, or stale after the Saturday cadence.
- Annual financial workflow is `degraded` or has many failed/deferred tasks.

Owner:

- MarketOps fundamentals operator.

First response:

1. Confirm the recurring timer is enabled when Option B is active:

   ```bash
   sudo -n signalops-deploy-agent scheduler-status
   ```

2. Confirm whether failures are provider-wide or limited to ineligible symbols.

Recovery action:

- For a provider-wide issue, wait for provider recovery or disable the timer to stop repeated noise:

  ```bash
  sudo -n signalops-deploy-agent scheduler-fmp-annual-disable
  ```

- For isolated symbol failures, keep the workflow degraded and review deferred symbols before retrying.
- Re-enable the timer after provider or data-quality correction:

  ```bash
  sudo -n signalops-deploy-agent scheduler-fmp-annual-enable
  ```

Verification:

- Admin Scheduled Jobs shows the latest FMP run with explicit status.
- Operations Health shows FMP annual freshness and task coverage.
- No return to the old 240/day barrier or uncontrolled retry loop.

Rollback criteria:

- Disable recurring FMP annual capture if it repeatedly degrades due to provider or schema instability.

## Closure evidence required after any incident

Record:

- Incident start/end timestamps.
- Affected tenant, route, job, or provider.
- Git revision and deployment state.
- Detection signal.
- Commands/actions used.
- Final verification result.
- Any disabled timers or feature flags.
- Follow-up work item if the incident exposed a product or architectural gap.
