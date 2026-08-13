# S4 Canary Execution Gate Evidence — 2026-08-13

Status: deployed control plane; provider execution remains disabled.

## Deployment record

- Source commit: `d9f6eef` (`feat(subscribers): add disabled S4 canary execution gate`)
- Migration: `000102_subscriber_global_eod_canary_execution_gate`
- Migration result: applied successfully from a clean `git archive` of the source commit. The untracked SRI migration and other local work were excluded.
- No gateway/web rebuild, scheduler mutation, coverage update, or market-data/provider request occurred.

## Dedicated workload preflight

The global-EOD workload preflight passed using the dedicated subscriber global-EOD database connection and declared machine identity `subscriber-global-eod-reconciler`.

The host does not have a PostgreSQL client installed, so the unchanged preflight script was run in a temporary PostgreSQL client container using host networking solely to reach the existing localhost-bound database connection. The script verified:

- non-superuser, non-CREATEROLE, NOBYPASSRLS membership in `signalops_subscriber_global_eod`;
- required read/append privileges on the global canary records;
- no `UPDATE`/`DELETE` privilege on the frozen execution plan, member, or evidence tables; and
- no access to subscriber watchlists, subscriber entitlements, or the legacy MarketOps asset-ownership table.

## Persisted gate

| Field | Observed value |
|---|---|
| Execution plan | `subeodgate_acc469bbbae92ff3121acfcb` |
| Frozen canary | `subeodcanary_c44b065fb587d29470db5119` |
| Session | 2026-08-12 |
| Execution state | `disabled` |
| Provider execution enabled | `false` |
| Scheduled execution enabled | `false` |
| Kill switch engaged | `true` |
| Maximum provider requests | 2 |
| Frozen request slots | `NVDA:1`, `AAPL:2` |
| Planned evidence rows | 2 |

The only ledger event kind is `execution_planned` (two rows). A valid-shape attempt to append `provider_request_started` was rejected by `subscriber_global_eod_canary_evidence_disabled_guard` with: `global EOD canary execution is disabled; only execution_planned evidence is permitted`. A subsequent ledger query confirmed zero provider-request events persisted.

## Remaining gate

This evidence closes the control-plane deployment, not collection execution. A later explicit provider-execution authorization still requires the separate execution worker release, short-lived workload credential, named-session approval, parity-report implementation and review, and backup/restore readiness. The current database schema makes these actions impossible through this gate.
