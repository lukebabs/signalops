# SignalOps Frontend Auth Integration Specification

Gate: G053
Audience: frontend-agent
Status: ready for frontend implementation
Date: 2026-07-09

## Objective

Implement browser authentication for the SignalOps web app against Syncratic IdP so the frontend can operate after the backend gateway is switched to `SIGNALOPS_AUTH_ENABLED=true`.

This gate must integrate with the backend G052 contract without inventing backend behavior. The app is still an internal operational dashboard, not a marketing/login product. Keep the UI compact, operator-focused, and consistent with the existing SignalOps shell.

## Current Backend Contract

The gateway already implements optional JWT enforcement:

- Auth is controlled by `SIGNALOPS_AUTH_ENABLED`.
- `/healthz` and `/readyz` remain public.
- Protected `/v1/*` APIs require `Authorization: Bearer <jwt>` when auth is enabled.
- JWT validation checks issuer, JWKS RS256 signature, expiry/not-before, audience, and tenant claim.
- Expected issuer: `https://auth.syncratic.co/realms/syncratic`.
- Expected JWKS: `https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/certs`.
- Expected audience: `signalops-api`.
- Expected tenant claim: `tenant_id`, currently `tenant-local`.
- Actor precedence when auth is enabled: `preferred_username`, then `email`, then `sub`.
- Role checks:
  - Read/protected `/v1/*` APIs require one of `signalops:viewer`, `signalops:operator`, or `signalops:admin`.
  - Alert/insight lifecycle POST APIs require `signalops:operator` or `signalops:admin`.
- Explicit `tenant_id` query values and `/v1/tenants/{tenant_id}/...` paths must match the token `tenant_id` claim.

The current deployment intentionally has `SIGNALOPS_AUTH_ENABLED=false` until this frontend gate is complete.

## IdP Configuration Already Completed

Realm:

```text
syncratic
```

Browser client:

```text
signalops-web
```

API audience/resource:

```text
signalops-api
```

OIDC settings:

- Public OIDC client.
- Authorization Code Flow enabled.
- PKCE required with `S256`.
- Direct Access Grants disabled.
- Implicit Flow disabled.
- Service Accounts disabled.
- Register both `https://signalops.syncratic.io/auth/callback` and
  `https://signalops.syncratic.io/auth/silent-renew` as valid redirect URIs
  (use the corresponding exact public origin for each environment).

Token claims expected:

```text
iss = https://auth.syncratic.co/realms/syncratic
aud includes signalops-api
tenant_id = tenant-local
preferred_username
email
realm_access.roles includes one or more SignalOps roles
```

Roles:

```text
signalops:viewer
signalops:operator
signalops:admin
```

Groups:

```text
/signalops/viewers   -> signalops:viewer
/signalops/operators -> signalops:operator
/signalops/admins    -> signalops:admin
```

Initial admin user:

```text
lukeb / luke@strategiclabs.io -> /signalops/admins
```

## Runtime Environment

Use these Vite env variables. They already exist in `.env.example`:

```env
VITE_SIGNALOPS_AUTH_ENABLED=true
VITE_SIGNALOPS_AUTH_ISSUER=https://auth.syncratic.co/realms/syncratic
VITE_SIGNALOPS_AUTH_REALM=syncratic
VITE_SIGNALOPS_AUTH_CLIENT_ID=signalops-web
```

Add frontend support for an optional API audience env if useful for validation/debug display, but do not require it unless needed:

```env
VITE_SIGNALOPS_AUTH_AUDIENCE=signalops-api
```

Do not put secrets in frontend env. `signalops-web` is public and must not use a client secret.

## Recommended Library

Use a proven OIDC SPA client rather than hand-rolling Authorization Code + PKCE.

Recommended package:

```text
oidc-client-ts
```

Rationale:

- Supports Authorization Code + PKCE for SPAs.
- Maintains tokens in browser-side state/storage.
- Handles redirect callback parsing and silent renewal primitives.
- Keeps the implementation smaller and less error-prone than custom OAuth code.

Do not add a fake login form. Authentication must redirect to Syncratic IdP.

## Required User Experience

When `VITE_SIGNALOPS_AUTH_ENABLED=false`:

- Preserve current behavior.
- Do not force login.
- Continue same-origin API calls without Authorization headers.
- Lifecycle mutation fallback may continue working as today for local development.

When `VITE_SIGNALOPS_AUTH_ENABLED=true`:

- On first app load, unauthenticated users see a compact authentication screen with:
  - SignalOps title/brand mark using the existing shell visual language.
  - One primary action: sign in.
  - A concise error area if login/callback fails.
- Clicking sign in redirects to `auth.syncratic.co` using Authorization Code + PKCE.
- After IdP callback, the app stores the authenticated user/session and routes to the originally requested path, or `/` if no path is available.
- The app shell shows the authenticated operator identity, preferably `preferred_username`, else `email`, else `sub`.
- The app shell includes a logout button/icon that redirects through the IdP logout flow and returns to SignalOps.
- If the token is expired and cannot be refreshed, clear auth state and return to the sign-in screen.
- No protected `/v1/*` API request should be issued before an access token is available.

## Route Guarding

Implement auth gating at the app/shell level so all existing route pages remain protected together.

Public without login:

- `/healthz` and `/readyz` are backend endpoints, not SPA pages.
- The SPA itself can show the login screen for all app routes when auth is enabled.

Protected app routes:

```text
/
/runs
/raw-events
/normalized-events
/idempotency
/sources
/pipelines
/rules
/signals
/alerts
/insights
/system
```

Do not create multiple fake unauthenticated route variants.

## Token Attachment

Update the API client in `web/src/api/client.ts` so every `/v1/*` request includes:

```http
Authorization: Bearer <access_token>
```

when frontend auth is enabled and a token exists.

Health/readiness calls may omit the token because the backend keeps them public. It is acceptable if the shared client attaches the token to health calls after login, but unauthenticated health checks must still work.

Required changes:

- Introduce a central token getter used by `get<T>` and `post<T>`.
- Avoid passing tokens manually from each route component.
- Keep all API calls same-origin unless `VITE_SIGNALOPS_API_BASE_URL` is explicitly configured.
- Preserve existing `ApiError` behavior for JSON gateway errors.
- Surface `401 unauthorized` and `403 insufficient_role`/`tenant_mismatch` distinctly in UI states.

## Tenant Handling

When auth is enabled, derive the frontend tenant from token claim:

```text
tenant_id
```

Replace hardcoded route constants like:

```ts
const TENANT_ID = 'tenant-local';
```

with the authenticated tenant from a shared auth/session hook or context.

Affected areas include, but may not be limited to:

- `DashboardRoute.tsx`
- `SourcesRoute.tsx`
- `PipelinesRoute.tsx`
- `RulesRoute.tsx`
- `NormalizedEventsRoute.tsx`
- `SignalsRoute.tsx`
- `AlertsRoute.tsx`
- `InsightsRoute.tsx`
- `IdempotencyLookup.tsx`
- `api/queries.ts` default tenant helpers
- `api/client.ts` catalog/list calls

When auth is disabled, preserve the current `tenant-local` default.

Do not add a tenant selector in G053. Multi-tenant selection is out of scope until backend policy supports it.

## Role Handling

Parse roles from:

```text
realm_access.roles
```

Also support this shape for future compatibility:

```text
resource_access.signalops-api.roles
```

Expose helper checks:

```ts
hasRole('signalops:viewer')
hasRole('signalops:operator')
hasRole('signalops:admin')
canReadSignalOps()
canMutateLifecycle()
```

Role behavior:

- Users with viewer/operator/admin can view protected app pages.
- Alert/insight lifecycle controls require operator or admin.
- If user lacks operator/admin, render lifecycle buttons disabled with a short tooltip/title and do not send the mutation request.
- If a mutation still returns `403`, show the existing inline mutation error style.

Do not hide the Alerts/Insights pages from viewers; viewers should still inspect data.

## Lifecycle Actor Behavior

The backend now derives actor from the token when auth is enabled. Update frontend lifecycle mutations:

- Stop sending `X-SignalOps-Actor: operator-local` when auth is enabled.
- The mutation body should keep only useful user-entered fields such as `note` and `reason`.
- When auth is disabled, it is acceptable to preserve the current local-development placeholder header or body actor to keep existing tests/dev flows working.

## Suggested File/Module Structure

Add a small auth module. Suggested paths:

```text
web/src/auth/config.ts
web/src/auth/oidc.ts
web/src/auth/session.tsx
web/src/auth/claims.ts
web/src/auth/session.test.tsx or claims.test.ts
```

Suggested responsibilities:

- `config.ts`: read and normalize `VITE_SIGNALOPS_AUTH_*` env values.
- `oidc.ts`: configure `UserManager` from `oidc-client-ts`.
- `claims.ts`: decode/access claims, roles, tenant, display identity.
- `session.tsx`: React provider/hook for auth state, sign in, sign out, token getter, role helpers.

A simpler structure is acceptable if tests and responsibilities remain clear.

## OIDC Redirect Handling

Use the current origin for redirect/logout return URIs:

```ts
const origin = window.location.origin;
redirect_uri = `${origin}/auth/callback`;
post_logout_redirect_uri = origin;
```

Add an SPA route for callback handling:

```text
/auth/callback
```

Also add a silent-renew callback route:

```text
/auth/silent-renew
```

It is loaded in a hidden iframe only when the IdP does not issue a refresh token.
It must complete `signinSilentCallback()` and must remain an allowed redirect URI.

### Session inactivity and renewal

A signed-in session remains valid while the operator is active. Pointer, keyboard,
scroll, touch, and window-focus activity reset the inactivity clock. The frontend
renews a near-expiry access token while that clock is active, but renewal does not
reset the clock. At the configured idle threshold it removes the local user and
requires a new sign-in.

Build-time settings:

```text
VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES=30
VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS=300
```

The IdP session idle/max settings remain authoritative and must allow at least the
chosen inactivity duration.

Callback route requirements:

- Complete signin redirect processing.
- Restore requested path if stored before redirect.
- Show a compact loading/error state while processing.
- On success, navigate back into the app.

Because this is a Vite SPA behind nginx fallback, no backend route should be needed for `/auth/callback`; browser refresh should still serve `index.html`.

Verify nginx fallback still supports this route. If needed, update `web/deploy/nginx.conf` only for SPA fallback correctness, not for auth proxying.

## Query/Cache Behavior

When auth state changes:

- On successful login, allow normal queries to run.
- On logout, clear TanStack Query cache to avoid showing prior tenant/operator data.
- On token refresh, no full cache clear is required.
- On `401`, clear session and route to login if auth is enabled.
- On `403`, keep session and show forbidden state/error.

Health indicator can continue polling public endpoints before login, but protected dashboard data must not fetch until authenticated.

## Error States

Use existing UI patterns from `States.tsx` and route-level inline errors. Add compact auth-specific states only where needed.

Required states:

- Signing in/loading session.
- Sign-in failed/callback failed.
- Unauthorized/session expired.
- Forbidden/insufficient role.
- Missing tenant claim.

Do not add verbose tutorial text or marketing copy.

## Tests Required

Add frontend tests covering auth behavior. Minimum expected coverage:

1. Env parsing:
   - Auth disabled by default/false.
   - Auth enabled when `VITE_SIGNALOPS_AUTH_ENABLED=true`.
2. Claims parsing:
   - `tenant_id` extraction.
   - display identity precedence: `preferred_username`, `email`, `sub`.
   - roles from `realm_access.roles`.
   - roles from `resource_access.signalops-api.roles`.
3. API client token behavior:
   - `/v1/*` GET includes `Authorization: Bearer ...` when auth enabled and token exists.
   - `/healthz` still works without token.
   - `401` and `403` gateway error envelopes become `ApiError` with correct status/code/message.
4. Lifecycle mutation behavior:
   - Auth enabled: no `X-SignalOps-Actor: operator-local` header is sent; Authorization header is sent.
   - Auth disabled: existing local placeholder behavior remains acceptable.
5. Role helpers:
   - viewer can read but cannot mutate lifecycle.
   - operator/admin can mutate lifecycle.

If full React provider tests are too expensive, prioritize pure unit tests for config/claims/client behavior plus one shell rendering test.

## Browser Validation Required

Use the running app after implementation.

With `VITE_SIGNALOPS_AUTH_ENABLED=false`:

- `npm test` passes.
- `npm run build` passes.
- Existing unauthenticated dashboard flow still works through `http://localhost:15173`.

With auth enabled in frontend dev/build configuration and backend still `SIGNALOPS_AUTH_ENABLED=false`:

- Sign in redirects to IdP.
- Callback completes and displays operator identity.
- API calls include Authorization header, even though backend does not require it yet.
- Logout clears session and query cache.

After backend auth is enabled in a later deployment validation step:

- `/healthz` and `/readyz` remain reachable without login.
- Unauthenticated `/v1/*` returns `401`.
- Authenticated viewer/operator/admin reads work.
- Operator/admin lifecycle mutations work and record token-derived actor.
- Viewer lifecycle mutation controls are disabled or backend returns `403` if attempted.

The last backend-enabled validation may be coordinated with Codex after G053 implementation; do not enable backend auth permanently until the frontend behavior is verified.

## Out of Scope

Do not implement these in G053:

- Backend auth changes; G052 already implemented the backend contract.
- New IdP clients, roles, or groups.
- Tenant selector or tenant administration.
- User management screens.
- Refresh-token storage redesign beyond what `oidc-client-ts` provides.
- Durable lifecycle audit history.
- TimescaleDB migration.
- WebSocket migration.
- Broad UI redesign.

## Acceptance Criteria

G053 is complete when:

- The frontend supports Syncratic IdP Authorization Code + PKCE login/logout.
- Auth-enabled API calls attach Bearer tokens centrally.
- Tenant values used by route queries come from token `tenant_id` when auth is enabled.
- Role helpers gate lifecycle action controls.
- `operator-local` lifecycle actor headers are not sent when auth is enabled.
- Auth-disabled mode preserves current development behavior.
- Tests and production build pass.
- Browser validation is recorded.
- `docs/build_journal.md` and `docs/gate_audit.md` receive UTC timestamped G053 implementation evidence.

## Suggested Implementation Order

1. Add `oidc-client-ts`.
2. Add auth config and claims helpers with tests.
3. Add session provider/hook and wrap the app.
4. Add `/auth/callback` route.
5. Add token getter support to `api/client.ts`.
6. Replace hardcoded tenant usage with session tenant fallback helpers.
7. Gate lifecycle controls by role and remove placeholder actor when auth is enabled.
8. Add tests for client/token/claims/role behavior.
9. Run `npm test`, `npm run build`, and browser validation.
10. Update journal/audit with exact evidence.
