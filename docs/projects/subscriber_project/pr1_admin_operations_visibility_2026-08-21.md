# PR-1 Admin Operations Visibility — 2026-08-21

Status: closed after the August 21, 2026 ET post-close acceptance cycle.

## Scope

This PR-1 slice makes MarketOps operational freshness visible to administrators without adding provider calls, migrations, or broad shell access.

The Administration operations-health API now returns a read-only `data_freshness` section built from the dedicated MarketOps data plane. Admin Workbench renders the section as a compact table under MarketOps Operations Health.

## Views covered

The freshness table covers:

- Dashboard — derived completed-session alignment across Market State, Risk/Reward, Sector Rotation Intelligence, and Signal Assurance.
- Assets analytical coverage — active tenant-local selected symbols with current Market State analytical evidence.
- Market State — latest global Market State projection session and row count.
- Risk/Reward — latest global Risk/Reward projection session and row count.
- Sector Rotation Intelligence — latest platform-global SRI snapshot session and active segment count.
- Signal Assurance — latest matured SAF observation session and observation count.
- Intraday conditions — latest tenant-local intraday snapshot timestamp and current 30-minute row count.
- FMP annual financials — latest global annual-financial workflow completion against its task count.
- Syncratic Ask — latest completed daily-narrative Ask success for the current governed context, with failure category/context detail when unhealthy.

## Status semantics

- `current`: latest evidence exists and row/alignment checks pass.
- `partial`: latest session has fewer rows than expected, Dashboard components are not aligned to one completed session, asset EOD coverage is incomplete, the latest FMP annual workflow has incomplete/non-succeeded tasks, or Syncratic Ask lacks a completed response for the latest daily context.
- `stale`: intraday latest snapshot is outside the configured freshness window.
- `missing`: no rows exist for that view.

## Security boundary

This is read-only. It does not:

- start jobs;
- run provider polling;
- mutate operational status;
- broaden scheduler actions;
- bypass tenant or subscription controls.

Run-now behavior remains constrained to the existing deployment-agent allowlist.

## Files changed

- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/api/scheduled_jobs.go`
- `internal/api/scheduled_jobs_test.go`
- `web/src/types.ts`
- `web/src/routes/SystemRoute.tsx`

## Verification performed

```text
gofmt -w internal/storage/storage.go internal/storage/postgres/repository.go internal/api/scheduled_jobs.go internal/api/scheduled_jobs_test.go
go test ./internal/api ./internal/storage/postgres
npm --prefix web run build
```

The live read-only SQL smoke could not be executed from Codex because raw `sudo -n docker exec ...` still requires an interactive password on the host. Deployment verification should confirm the API response after Gateway/Web rebuild.

## Deployment verification

After deploying Gateway and Web, verify:

1. Admin Workbench loads MarketOps Operations Health.
2. The new data freshness table is visible.
3. Dashboard, Market State, Risk/Reward, SRI, SAF, Intraday, FMP annual financials, and Syncratic Ask rows are present.
4. Dashboard is `current` only when completed-session components align.
5. Failed/stale/partial states include a reason.
6. Existing scheduled-job run-now buttons still use only constrained deployment-agent actions.


## Live deployment evidence

Production deployment was executed through the constrained deployment-agent path after commit `248c85a`.

Evidence:

```text
signalops_public_production_deploy_verified
```

Public route checks returned `200` for:

```text
https://signalops.syncratic.io/readyz
https://signalops.syncratic.io/marketops/watchlists
https://signalops.syncratic.io/marketops/admin
```

`scheduler-status` remained clean after deployment. The first bundled subscriber pilot smoke failed on the watchlist login/heading check after reaching `chrome-error://chromewebdata/`, matching the known smoke harness/auth instability rather than a route outage. A single retry of the constrained smoke action passed:

```text
..                                                                       [100%]
2 passed in 5.83s
```

This route-level evidence was superseded by the automated Admin Workbench smoke and August 21 post-close acceptance evidence below.


## Assets and FMP extension

The second PR-1 source slice added the remaining required freshness rows:

- `assets`: compares active tenant-local symbols to active global `eod_baseline` coverage.
- `fmp_annual`: reports the latest `subscriber_global_annual_financial_workflows` session and succeeded task count. If no workflow exists yet, the row is still returned as `missing` so the administrator sees an explicit gap.

Verification performed:

```text
gofmt -w internal/storage/postgres/repository.go
go test ./internal/api ./internal/storage/postgres
npm --prefix web run build
```

## Assets and FMP live deployment evidence

Production deployment for the Assets/FMP freshness extension was completed from commit `41e3110`.

Verification after deployment:

```text
https://signalops.syncratic.io/readyz           -> 200
https://signalops.syncratic.io/admin/system      -> 200
```

Scheduler status after deployment remained clean:

```text
timer=signalops-marketops-boundary-intraday.timer active=active
timer=signalops-marketops-boundary-daily-postclose.timer active=active
timer=signalops-marketops-boundary-postclose-recovery.timer active=active
timer=signalops-marketops-boundary-sri-refresh.timer active=active
timer=signalops-marketops-boundary-sri-holdings-refresh.timer active=active
timer=signalops-marketops-boundary-warm-eod.timer active=active
service=signalops-marketops-boundary-schedule@preflight.service result=success
service=signalops-marketops-boundary-schedule@marketops-intraday.service result=success
service=signalops-marketops-boundary-schedule@marketops-daily-postclose.service result=success
service=signalops-marketops-boundary-schedule@marketops-postclose-recovery.service result=success
service=signalops-marketops-boundary-schedule@marketops-sri-refresh.service result=success
service=signalops-marketops-boundary-schedule@marketops-sri-holdings-refresh.service result=success
service=signalops-storage-monitor.service result=success
service=signalops-retention-governance.service result=success
```

Constrained production UI smoke passed:

```text
..                                                                       [100%]
2 passed in 5.60s
```

PR-1 browser acceptance is now automated and passed. The Playwright smoke logs in as `luke@strategiclabs.io`, opens Administration -> System, waits for `/v1/administration/marketops/operations-health`, asserts HTTP 200, verifies the API `data_freshness` payload includes all eight required labels, and verifies each label is rendered in the browser.

```text
scripts/run_subscription_admin_ui_smoke.sh
..                                                                       [100%]
2 passed in 2.29s
```


## 2026-08-22 post-close closure evidence

The August 21, 2026 ET post-close acceptance window completed and was verified on August 22, 2026 UTC.

Browser and API evidence:

```text
curl -fsS https://signalops.syncratic.io/readyz
{"service":"signalops-gateway","status":"ready","time":"2026-08-22T06:04:44Z"}

scripts/run_subscription_admin_ui_smoke.sh
3 passed in 3.26s

scripts/run_subscriber_access_control_ui_smoke.sh
1 passed in 3.83s

scripts/run_marketops_dashboard_freshness_ui_smoke.sh
1 passed in 5.08s
marketops_dashboard_freshness_ui_smoke_passed
```

Dedicated MarketOps ledgers aligned to the August 21, 2026 ET completed session:

```text
Market State  2026-08-21  132 symbols  latest_as_of=2026-08-21 22:04:31 UTC
Risk/Reward   2026-08-21  132 symbols  latest_observed=2026-08-21 22:19:58 UTC
SRI           2026-08-21   16 segments
Intraday      2026-08-21  132 symbols  latest_snapshot=2026-08-21 22:15:00 UTC
```

Scheduler/job-status evidence showed `marketops-daily-postclose`, `marketops-risk-reward`, `marketops-postclose-recovery`, `marketops-sri-refresh`, `marketops-sri-holdings-refresh`, and `marketops-intraday` succeeded for the acceptance window. `marketops-warm-eod` reported the expected governed `degraded` state for a bounded provider gap.

PR-1 is closed for the current pilot-readiness path. FMP recurring activation remains tracked under PR-4, not PR-1.

## Syncratic Ask operations-health extension — 2026-08-23

The operations-health freshness table now includes a ninth row, `Syncratic Ask`. The row is read-only and sourced from persisted daily narrative context windows, completed Ask insight metrics, and failed/retryable Syncratic intelligence jobs. It does not call the AI Gateway, mutate context windows, or enqueue work.

Healthy state means at least one completed Ask insight exists for the latest active daily narrative session and no newer failed Ask job supersedes that success. Partial state exposes the latest failure category and context-window id when available; missing state means no active daily narrative context exists.

The Subscription Administration browser smoke now requires the `Syncratic Ask` label in the operations-health table.
