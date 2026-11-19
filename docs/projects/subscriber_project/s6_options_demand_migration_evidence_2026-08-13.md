# S6 Options-Demand Shadow Deployment Evidence — 2026-08-13

Status: internal shadow storage deployed; no planner login, snapshot, capture, scheduler, entitlement, or provider operation is enabled.

Migration `000111_subscriber_options_demand_shadow` is recorded in the production migration ledger. It was first validated in an explicitly rolled-back transaction, then applied atomically.

The `signalops_subscriber_options_demand` group-role rehearsal proved:

| Check | Result |
| --- | --- |
| Superuser, CREATEROLE, BYPASSRLS | All false |
| Membership in dedicated group | True |
| Direct `subscriber_watchlists` read | False |
| Direct entitlement/capability read | False |
| Direct MarketOps Options-chain read | False |
| Aggregate-projection execute | True |
| Snapshot `SELECT, INSERT` | True |
| Snapshot `UPDATE, DELETE` | False |

The aggregate projection returned zero rows because no tenant currently has the `options_demand` capability enabled. That is the expected default-deny state. No snapshot was written.

The final missing internal deployment input is a dedicated `LOGIN NOINHERIT` database principal, secret-managed DSN, and group membership in `signalops_subscriber_options_demand`. The preflight must pass through that real login before the shadow planner command is run. It must not be combined with a gateway, global-EOD, or Options-capture identity.
