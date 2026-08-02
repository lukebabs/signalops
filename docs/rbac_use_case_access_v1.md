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
