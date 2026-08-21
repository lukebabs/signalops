# PR-1 Admin Operations Visibility — 2026-08-21

Status: implemented in source; live deployment and browser verification remain next.

## Scope

This PR-1 slice makes MarketOps operational freshness visible to administrators without adding provider calls, migrations, or broad shell access.

The Administration operations-health API now returns a read-only `data_freshness` section built from the dedicated MarketOps data plane. Admin Workbench renders the section as a compact table under MarketOps Operations Health.

## Views covered

The freshness table covers:

- Dashboard — derived completed-session alignment across Market State, Risk/Reward, Sector Rotation Intelligence, and Signal Assurance.
- Assets coverage — active tenant-local symbols with active global EOD baseline coverage.
- Market State — latest global Market State projection session and row count.
- Risk/Reward — latest global Risk/Reward projection session and row count.
- Sector Rotation Intelligence — latest platform-global SRI snapshot session and active segment count.
- Signal Assurance — latest matured SAF observation session and observation count.
- Intraday conditions — latest tenant-local intraday snapshot timestamp and current 30-minute row count.
- FMP annual financials — latest global annual-financial workflow completion against its task count.

## Status semantics

- `current`: latest evidence exists and row/alignment checks pass.
- `partial`: latest session has fewer rows than expected, Dashboard components are not aligned to one completed session, asset EOD coverage is incomplete, or the latest FMP annual workflow has incomplete/non-succeeded tasks.
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
3. Dashboard, Market State, Risk/Reward, SRI, SAF, and Intraday rows are present.
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

Remaining live verification:

- Log in as `luke@strategiclabs.io` and confirm Admin Workbench renders the data-freshness table under MarketOps Operations Health.
- Confirm rows are present for Dashboard, Assets coverage, Market State, Risk/Reward, Sector Rotation Intelligence, Signal Assurance, Intraday conditions, and FMP annual financials.


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
https://signalops.syncratic.io/marketops/admin  -> 200
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

Remaining PR-1 browser acceptance:

- Log in as `luke@strategiclabs.io`.
- Open Administration -> Admin Workbench.
- Confirm the MarketOps Operations Health freshness table includes all eight rows: Dashboard, Assets coverage, Market State, Risk/Reward, Sector Rotation Intelligence, Signal Assurance, Intraday conditions, and FMP annual financials.

