# PR-1 Admin Operations Visibility — 2026-08-21

Status: implemented in source; live deployment and browser verification remain next.

## Scope

This PR-1 slice makes MarketOps operational freshness visible to administrators without adding provider calls, migrations, or broad shell access.

The Administration operations-health API now returns a read-only `data_freshness` section built from the dedicated MarketOps data plane. Admin Workbench renders the section as a compact table under MarketOps Operations Health.

## Views covered

The freshness table covers:

- Dashboard — derived completed-session alignment across Market State, Risk/Reward, Sector Rotation Intelligence, and Signal Assurance.
- Market State — latest global Market State projection session and row count.
- Risk/Reward — latest global Risk/Reward projection session and row count.
- Sector Rotation Intelligence — latest platform-global SRI snapshot session and active segment count.
- Signal Assurance — latest matured SAF observation session and observation count.
- Intraday conditions — latest tenant-local intraday snapshot timestamp and current 30-minute row count.

## Status semantics

- `current`: latest evidence exists and row/alignment checks pass.
- `partial`: latest session has fewer rows than expected, or Dashboard components are not aligned to one completed session.
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
- Confirm rows are present for Dashboard, Market State, Risk/Reward, Sector Rotation Intelligence, Signal Assurance, and Intraday conditions.
