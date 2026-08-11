# Sprint S0 — Baseline and Controls

Status: implementation in progress. This sprint adds observation and release-control artifacts only. It does not change database ownership, API behavior, job scheduling, provider activity, or subscriber access.

## Objective

Capture a reproducible baseline of the current tenant-owned MarketOps plane before any Subscriber Project schema or traffic change. The baseline is evidence for later parity and rollback decisions; it is not a new data source or a production cutover.

## Implemented baseline utility

Run from the repository root:

```bash
scripts/subscriber_project_s0_baseline.sh --tenant-id tenant-local
```

The utility:

- performs `SELECT` queries only against the primary and temporal stores;
- reads existing `runtime/scheduled-jobs/*.json` status records;
- does not invoke a provider, scheduler, migration, gateway write, or background worker;
- reports the current universe, EOD and Options coverage, applied migrations, RBAC-grant count, authentication configuration booleans, and recorded job status; and
- emits Markdown to standard output so the operator can retain a timestamped evidence artifact in the approved operational-evidence location.

The script intentionally does not commit a live report to this repository: counts, dates, user-grant totals, and job state are environment-specific operational evidence.

## Current contracts being baselined

| Concern | Current contract | S0 action |
|---|---|---|
| Asset ownership | `marketops_asset_universe` is tenant-owned; `marketops_universal_assets` is the active projection. | Record group and ticker counts; do not create a global asset table. |
| Collection and algorithms | Existing post-close and scheduled workflows resolve the tenant-local active universe. | Record status and evidence coverage; do not alter a job or provider budget. |
| Market evidence | Primary-store Options captures and temporal normalized EOD records retain existing tenant scope and provenance. | Record dates and counts; do not copy, backfill, or reinterpret evidence. |
| Browser/API access | OIDC/JWT, tenant claim, registered-use-case grants, and access audit are the current foundation. | Record enabled-state and MarketOps grant count; S0-A hardens subscriber authorization later. |
| Retention | Existing retention jobs and policies remain authoritative. | Record migration/job context only; do not alter retention. |

## Reserved rollout controls

S0 reserves the following environment names. They all default to `false` and have no application behavior in S0:

- `SIGNALOPS_SUBSCRIBER_GLOBAL_CATALOG_ENABLED`
- `SIGNALOPS_SUBSCRIBER_LISTS_ENABLED`
- `SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_SHADOW_ENABLED`
- `SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_CANARY_ENABLED`
- `SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_ENABLED`

A later sprint may implement one flag only with all of the following: server-side enforcement, default-off configuration, a tenant-scoped allow-list where applicable, metric/alert coverage, a tested rollback, and an update to this document. A browser-only flag never authorizes access or provider collection.

## Baseline metrics

The baseline report is the source record for these measurements:

| Metric | Later comparison use |
|---|---|
| Latest applied migration in each store | Detect unreviewed schema drift before parity comparison. |
| Active rows by universe group and distinct active tickers | Prove no silent loss or duplication during catalog mapping. |
| Analyst-watchlist row count | Reconcile migration to tenant-default or user-private memberships. |
| Latest normalized EOD observation | Detect stale or changed processing behavior. |
| Options symbols, date span, and analytics-ready captures | Preserve truthful prospective-history and warm-up comparisons. |
| MarketOps grant count and auth enabled-state | Confirm the access-control starting condition before S0-A. |
| Recorded scheduled-job status | Identify existing operational degradation before a canary is evaluated. |

Counts alone do not establish correctness. Every future parity comparison must also sample stable asset IDs/tickers, session dates, quality states, provenance, and failure/deferred reasons.

## Evidence review and S0 exit

Before S0 is marked complete, an operator must:

1. Run `bash -n scripts/subscriber_project_s0_baseline.sh`.
2. Run the utility against the intended non-production or production-like tenant and retain its unedited output with execution timestamp and environment identifier.
3. Review existing failed/stale job status, sparse evidence, or missing coverage as baseline facts rather than silently normalizing them.
4. Record the approved evidence location and reviewer in the Sprint S0 work record.
5. Confirm that no migration, provider request, job schedule, API contract, access grant, or ownership change was made by S0.
6. Approve the rollback posture below before beginning S0-A or S1.

## Rollback posture

S0 is reversible by removal of its local script/documentation artifacts only. It adds no migration, storage record, runtime flag consumer, provider request, timer, API route, or UI path. If a later Sprint S0 change is proposed that would create any of those, it belongs in the owning later sprint and requires a separate rollback plan.

## Handoff

- **Next mandatory gate:** [S0-A access-control hardening](global_catalog_watchlists_roadmap.md#subscriber-access-control-readiness-gate).
- **Next data-plane sprint after S0-A:** S1 global-catalog shadow.
- **Authoritative architecture and sprint order:** [Global Catalog and Watchlist Roadmap](global_catalog_watchlists_roadmap.md).
