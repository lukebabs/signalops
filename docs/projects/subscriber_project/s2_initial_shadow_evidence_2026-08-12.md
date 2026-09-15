# S2 Catalog Breadth and EOD Planner Shadow+ßuÁ‚ùÁT Initial Evidence

Date: 2026-08-12 UTC  
Environment: live SignalOps PostgreSQL  
Migration: 000092_subscriber_global_eod_planner_shadow  
Plan run: subeodplan_9c365ced12694471d8c76e68  
Correlation: s2-initial-shadow-plan-2026-08-12

## Migration and workload boundary

Migration 000092 applied at 2026-08-12 16:05:46 UTC. Its eligibility-decision, EOD plan, plan-member, and activation-request tables are owned by signalops_subscriber_migrator.

Two dedicated workload logins are now present:

| Workload login | Group role | Superuser / bypass RLS |
|---|---|---|
| signalops_subscriber_catalog_sync_runtime | signalops_subscriber_catalog_sync | no / no |
| signalops_subscriber_global_eod_runtime | signalops_subscriber_global_eod | no / no |

The global-EOD login can read the shared catalog and write plans but cannot read catalog eligibility-decision history. It cannot delete plan data or read the legacy tenant universe. The browser gateway has no SELECT privilege on any S2 table.

## Initial planner result

| Measure | Result |
|---|---:|
| Planner mode | shadow |
| Approved capacity | 1,000 |
| Candidates | 178 |
| Eligible | 0 |
| Selected | 0 |
| Excluded | 178 |
| Exclusion reason | not_eligible |
| Hot-set members written | 0 |
| Global eligibility decisions recorded | 0 |
| Cold activation requests recorded | 0 |
| Coverage rows outside shadow mode | 0 |

This is the expected safe result. The S1 compatibility seed preserved source identity but did not contain governed provider evidence sufficient to confirm US exchange-listed common-stock eligibility. The S2 planner therefore selected no asset and changed no coverage state, provider activity, scheduler, or tenant-facing path.

## Outstanding S2 evidence

S2 cannot close until a governed Massive-reference import provides valid admission evidence, a replayable hot-set plan selects the eligible top cohort, and an authorized later list path proves duplicate cold-demand requests coalesce. These are deliberately not inferred from the legacy universe and do not authorize a provider or browser change.
