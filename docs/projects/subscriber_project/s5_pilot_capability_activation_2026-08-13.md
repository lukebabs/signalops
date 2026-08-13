# S5 Pilot Capability Activation — 2026-08-13

Status: approved limited pilot policy, pending deployment of the quota-enforcement revision.

## Policy

The named pilot remains `tenant-pilot-b` under `subscriber-list-pilot-v1`.

| Capability | Enabled | Quota | Scope |
| --- | ---: | ---: | --- |
| `catalog_search` | Yes | 50 | Governed-catalog results only; no provider call or catalog admission. |
| `eod_activation` | Yes | 10 | At most ten queued cold-asset activation requests for this tenant and entitlement version. |
| `options_demand` | No | 0 | No Options demand planning or capture. |

## Enforcement

The gateway reserves one server-derived, idempotent `eod_activation` quota unit before it writes a private-list catalog membership. It consumes that unit only when the stored coverage state results in a `queued` central activation request. It releases the unit for an already enabled asset or an unsuccessful mutation. When the ten-unit quota is exhausted, the gateway returns `429 subscriber_activation_quota_exhausted` before the list mutation.

An activation request remains only a centrally deduplicated record of demand. It does not enable EOD collection, call Massive, alter a tenant-default list, or change the existing shadow coverage plan. A separate approved canary/reconciler rollout is required before any provider-backed coverage changes.

## Controlled operation

`subscriber-s3-pilot-provisioner` accepts explicit default-deny flags:

```text
--catalog-search-quota=50 --eod-activation-quota=10
```

A zero value disables the respective capability. The command uses the dedicated subscriber gateway database identity and ordinary RLS-scoped storage methods, preserving entitlement provisioning and quota audits.
