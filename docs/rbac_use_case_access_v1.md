# Tenant Use-Case RBAC v1

SignalOps authorizes authenticated people within the `tenant_id` carried by their verified Keycloak token. Access is assigned per registered use case. The initial registered use cases are
`marketops` and `cyberops`.

## Roles

- **Super-admin:** the Keycloak realm role `super_admin`. The legacy `signalops:admin` role remains accepted during migration. This role is tenant-scoped by the JWT claim and can manage grants, use every application, and use the platform Administration Workbench.
- **Read:** may issue read and streaming requests for the assigned use case only.
- **Write:** includes read plus lifecycle actions, settings/configuration, replays, and other mutations within the assigned use case.

Platform Console and grant management are super-admin only. `signalops:viewer` and `signalops:operator` no longer grant product access once the RBAC repository is configured; explicit grants are authoritative.

## Grant management

### IdP directory search

The Administration Workbench searches existing Keycloak identities by name, username, or email before a grant is saved. SignalOps forwards the signed-in super-admin bearer token to Keycloak; it does not store or expose an IdP administration credential. The Keycloak `super_admin` composite must include the realm-management client roles `query-users` and `view-users`.

Super-admins call the Administration access endpoints with the target identity's immutable Keycloak `sub`, optional display name/email, application, and `read` or `write` permission. SignalOps does not create IdP users, issue invitations, or manage passwords. Each grant, permission change, and revocation creates an immutable tenant audit record.

### Initial cross-tenant provisioning

The dedicated Keycloak realm role `signalops:tenant_provisioner` supports a narrow, one-time tenant bootstrap only. It may call `POST /v1/administration/tenant-provisioning/access` to create the first access grant in a tenant other than the provisioner's own token tenant.

The endpoint is deliberately restricted:

- it accepts only one initial `marketops` grant with `read` or `write` permission;
- it rejects a target tenant with current access or immutable access history;
- it writes through the normal grant and immutable audit path, with the provisioner's subject as the grant actor;
- it cannot list, edit, or revoke foreign-tenant grants; and
- it does not grant cross-tenant MarketOps, subscriber, or platform-data access.

All ordinary endpoints, including the Administration Workbench, remain bound to the authenticated token tenant. Assign `signalops:tenant_provisioner` only to a controlled provisioning identity (or as a short-lived additional role to a platform operator), then remove it when the bootstrap is complete. The target tenant must subsequently manage its own grants through a tenant-scoped administrator.

Example request, using the provisioner's own access token:

```sh
curl --request POST 'https://signalops.syncratic.io/v1/administration/tenant-provisioning/access' \
  --header 'Authorization: Bearer <provisioner-access-token>' \
  --header 'Content-Type: application/json' \
  --data '{
    "tenant_id": "tenant-pilot-b",
    "subject": "<keycloak-subject>",
    "display_name": "Pilot user",
    "email": "pilot@example.com",
    "app_id": "marketops",
    "permission": "read"
  }'
```

## Landing experience

Authenticated entry uses the self-experience response to show only the use-case profiles available to the caller. A user with one permitted domain opens it directly; a user with multiple domains selects from the concise landing page. Super-admin Administration remains a header utility rather than a use-case tile. Profile metadata supplies the landing summary and route prefix, so a newly registered backend profile appears without a frontend domain-specific widget.

## Registered use-case boundary

Migration `000073_registered_use_case_profiles` makes the registered-profile
table the database authorization boundary for grants. `tenant_user_access.app_id`
is a foreign key to that table, which is seeded with MarketOps and CyberOps.
Adding a future domain therefore requires both its canonical backend profile and
an explicit registration migration; arbitrary app IDs cannot be granted. This
keeps the landing experience profile-driven without weakening RBAC.

## Enforcement

The gateway verifies issuer, audience, signature, expiry, and tenant match before resolving grants. The UI derives its role visibility from the same access-token claims; OIDC UserInfo alone is not authoritative for roles. It classifies each request as MarketOps, CyberOps, or platform. UI visibility is a convenience only; direct API requests are denied when the caller lacks the matching grant.

## Subscriber Project extension

The current grant model is the authorization foundation for the future Subscriber Project, not its complete production policy. It already provides verified identity, tenant scoping, immutable subject identity, registered application grants, super-admin management, and grant audit.

Before subscriber capabilities are enabled, the gateway and data model must additionally enforce subject-owned lists, tenant-managed default lists, shared-global-record projections, product entitlements, provider budgets, and scoped worker identities. Every tenant-bearing value from a path, query, or request body must be derived from or checked against the authenticated principal. Cross-tenant and privilege-escalation attempts must be covered by direct API integration tests.

The Subscriber Project roadmap defines this as S0-A, Access-control hardening. No subscriber global-catalog, membership, entitlement, or coverage projection is enabled until that gate has passed.
