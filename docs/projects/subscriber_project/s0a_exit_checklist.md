# Sprint S0-A Exit Checklist

Status: implementation complete locally; formal deployment exit remains pending.

## Completed repository and local proof

The Subscriber Project now has:

- canonical principal-bound tenant and subject guards in the gateway;
- default-deny entitlement policy, forced-RLS entitlement/quota storage, atomic idempotent reservation, consume/release lifecycle, and immutable audits;
- NOLOGIN database group roles, transaction-local tenant scope, and schema policy verification;
- a dedicated local non-superuser gateway login that passed the workload preflight; and
- a passing OIDC discovery/JWKS preflight and full Go test suite.

No subscriber feature is enabled by this work. The current MarketOps Assets APIs and scheduled workers remain the production path.

## Required deployment evidence

Run these against the intended deployment using its privileged migration connection and its separately provisioned subscriber gateway login:

    bash ./scripts/subscriber_project_s0_baseline.sh --tenant-id <pilot-tenant>
    bash ./scripts/signalops_oidc_preflight.sh
    bash ./scripts/subscriber_project_rls_preflight.sh
    SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL='<dedicated-gateway-login-dsn>' bash ./scripts/subscriber_project_gateway_workload_preflight.sh

The gateway workload preflight requires a login that is non-superuser, non-CREATEROLE, NOBYPASSRLS, and a member of only signalops_subscriber_gateway. It proves:

- no tenant-private row is visible without signalops.tenant_id;
- a transaction sees its own seeded probe row only;
- switching the tenant context suppresses that row; and
- the transaction rolls back, leaving no probe data.

## Browser and cross-tenant test

Use two browser users in distinct tenants plus a tenant administrator:

1. Authenticate each user with the intended OIDC configuration and verify only its granted MarketOps APIs are available.
2. Attempt conflicting path, query, and JSON-body tenant_id values; each must be rejected before a storage mutation.
3. Attempt foreign record detail and mutation requests; each must return the ordinary not-found response and leave the foreign record unchanged.
4. Verify a user cannot impersonate another subject and a non-administrator cannot manage a tenant-default resource.
5. Retain correlation IDs, screenshots/request logs, and the unedited preflight output in the approved deployment evidence store.

## Initial tenant bootstrap test

The initial provisioner endpoint is a controlled setup operation, not a general cross-tenant administration feature:

1. Create the Keycloak realm role `signalops:tenant_provisioner` and assign it only to the approved provisioning identity. Ensure its access token contains the role.
2. With that identity's token, create the first `marketops` `read` or `write` grant for the pilot identity in `tenant-pilot-b` through `POST /v1/administration/tenant-provisioning/access`.
3. Confirm the response is `201`, the immutable access audit identifies the provisioning subject, and the target tenant's first grant has the intended Keycloak `sub`.
4. Repeat the request: it must return `409 tenant_already_provisioned`. Repeat with an identity without `signalops:tenant_provisioner`: it must return `403 tenant_mismatch`.
5. Configure the pilot identity's one authoritative `tenant_id` claim and its SignalOps viewer/operator role, then sign in and confirm its tenant-scoped MarketOps access works. Do not leave multiple Keycloak mappers emitting `tenant_id`.

## Formal exit decision

S0-A may be marked complete only after the deployment evidence and browser/cross-tenant test are reviewed by the security and platform owners. Until then, every subscriber rollout flag remains false and S1 global-catalog work must not be enabled.

## Rollback

Disable or remove the dedicated workload login grants and leave all subscriber rollout flags false. This preserves existing MarketOps workflows and does not delete the RLS audit evidence.
