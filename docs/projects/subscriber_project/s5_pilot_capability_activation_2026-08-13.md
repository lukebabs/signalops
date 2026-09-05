# S5 Pilot Capability Activation — 2026-08-13

Status: active limited pilot policy, deployed with gateway revision `7759446`.

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

## Activation evidence

- Gateway revision `7759446` is healthy in the public stack.
- The controlled provisioner successfully recorded `catalog_search=50`, `eod_activation=10`, and `options_demand=0` for `tenant-pilot-b`.
- The entitlement provisioning audit contains the resulting record. At activation time there were zero quota reservations and zero tenant-pilot-b activation requests.
- Migration `000100_subscriber_catalog_canonical_projection` is applied. The RLS-scoped search projection returns one canonical AAPL record, while the underlying source provenance remains retained.

## Controlled operation

`subscriber-s3-pilot-provisioner` accepts explicit default-deny flags:

```text
--catalog-search-quota=50 --eod-activation-quota=10
```

A zero value disables the respective capability. The command uses the dedicated subscriber gateway database identity and ordinary RLS-scoped storage methods, preserving entitlement provisioning and quota audits.
