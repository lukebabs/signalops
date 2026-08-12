# Subscriber Entitlement and Quota Policy Contract

Status: storage foundation implemented. RLS-scoped provisioning, atomic reservation, and durable audit are available only to future subscriber server-side code; no gateway route, worker integration, workload credential, or subscriber feature is enabled.

## Purpose

Subscriber product policy is separate from the existing SignalOps and MarketOps grants. Existing grants answer whether an authenticated caller may use a registered application endpoint. Subscriber policy answers whether a tenant is entitled to a metered platform capability.

The pure evaluator is at `internal/subscriber/policy`. It creates deterministic, audit-ready decisions but does not persist a tenant, reserve quota, expose an API, or enable a feature flag.

## Initial capabilities

| Capability | Intended use |
|---|---|
| `catalog_search` | Search the centrally governed global asset catalog. |
| `eod_activation` | Request or maintain centrally planned shared EOD coverage. |
| `options_demand` | Contribute an eligible asset to the centrally deduplicated Options-demand plan. |

Future enrichments require a new explicit capability. A capability must never be inferred from a MarketOps grant, product name, browser control, or a client-supplied tier.

## Decision contract

An authoritative provisioning adapter will provide one tenant-level entitlement record containing the tenant ID, provisioning version, enabled capabilities, and a quota limit per capability. A caller supplies the authenticated tenant, immutable subject, capability, request units, correlation ID, time, and a current usage snapshot.

`Evaluate` returns one of these stable outcomes:

| Outcome | Meaning |
|---|---|
| `allowed` | Capability is explicitly enabled and requested units fit within the explicit quota. |
| `blocked_entitlement` | Tenant does not match the entitlement record or the capability is not explicitly enabled. |
| `deferred_quota` | Capability is enabled but its quota is absent, zero, or insufficient. |
| `invalid_request` | Required scope or units are invalid, or the usage snapshot is invalid. |

The policy is default-deny. There is no implicit free tier or quota. Exact commercial tiers, prices, periods, and limits remain a product-provisioning decision and are deliberately not encoded in this evaluator.

The returned decision carries tenant, subject, capability, requested and consumed units, quota limit and remaining units, entitlement version, policy version, correlation ID, and decision time. A future durable audit adapter must retain that result with the provisioning and usage-reservation provenance that produced it.

## Activation boundary

The RLS-scoped storage foundation now retains explicit tenant provisioning, per-provisioning-version idempotent quota reservations, provisioning audit, and policy-decision audit. It remains unreachable from a route or worker and has no feature flag consumer. Before any subscriber capability is enabled, the project must add workload-identity enforcement, server-side authorization at the relevant route or worker, release/consume lifecycle handling, bounded concurrency behavior, and gateway-level cross-tenant negative tests.
