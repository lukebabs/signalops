# S6 Options Demand Snapshot Contract

Status: additive, shadow-only persistence contract. Migration `000111` remains unapplied; the dedicated command is wired but cannot run until it is applied and the workload login is provisioned.

The planner receives execute-only access to `subscriber_options_demand_aggregate()`. The function returns only one aggregate row per eligible global asset: the asset ID, highest policy-derived tier rank, eligible tenant count, eligible watcher count, and deferred-session age. It returns no tenant ID, list ID, subject, entitlement version, or membership row.

`subscriber_options_demand_snapshot_runs` and `subscriber_options_demand_snapshot_members` retain only that aggregate and the deterministic selection/deferred decision. Their execution mode is constrained to `shadow`, and no database permission is granted to the Options-capture identity.

The dedicated planner identity cannot select from subscriber watchlists, memberships, entitlements, or capabilities. It cannot call a provider through this schema, enqueue a capture, mutate an entitlement, or access existing MarketOps Options evidence.
