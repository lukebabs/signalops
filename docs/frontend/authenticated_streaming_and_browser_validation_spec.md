# SignalOps Authenticated Streaming and Browser Validation Specification

Gate: G054
Audience: frontend-agent
Status: ready for frontend implementation
Date: 2026-07-09

## Objective

Close the frontend-owned auth enablement gaps that remain after G053:

1. Make dashboard streaming safe when frontend auth is enabled and backend auth is later switched to `SIGNALOPS_AUTH_ENABLED=true`.
2. Add a repeatable real-browser validation path for Syncratic IdP login/logout, token attachment, tenant/role resolution, and callback behavior.

This is a frontend gate. Do not change backend auth policy in this gate. Backend auth enablement remains a coordinated Codex/operator step after this gate is implemented and validated.

## Current State

Backend G052:

- Gateway auth is implemented and controlled by `SIGNALOPS_AUTH_ENABLED`.
- `/healthz` and `/readyz` are public.
- Protected `/v1/*` APIs require Bearer JWT when backend auth is enabled.
- `/v1/streams/dashboard` is under `/v1/*`, so it is protected when backend auth is enabled.

Frontend G053:

- `oidc-client-ts` login/logout support is implemented.
- App-level auth gate is implemented.
- API client attaches `Authorization: Bearer <token>` for normal fetch-based `/v1/*` calls when auth is enabled.
- Tenant and role helpers are implemented.
- Lifecycle controls are role-gated and stop sending `X-SignalOps-Actor: operator-local` when auth is enabled.
- Current deployed frontend remains auth-disabled by default until browser validation and backend auth enablement are coordinated.

Remaining frontend issue:

- `web/src/api/stream.ts` uses native browser `EventSource`.
- Native `EventSource` cannot set custom `Authorization` headers.
- Therefore the dashboard stream cannot authenticate against protected `/v1/streams/dashboard` using the G052 Bearer-token contract.

## Required Direction

For G054, implement an auth-aware dashboard stream mode with a safe fallback.

Recommended approach:

- When frontend auth is disabled: keep the existing native `EventSource` behavior.
- When frontend auth is enabled: do **not** open native `EventSource` to `/v1/streams/dashboard`.
- Instead, use REST polling fallback for dashboard freshness until the backend explicitly supports an authenticated stream transport.

Reasoning:

- The backend currently accepts Bearer tokens in the `Authorization` header.
- Adding access tokens to query strings would leak credentials into logs/history and is not acceptable unless a short-lived, stream-specific token is designed server-side.
- Cookie/session edge auth is not implemented.
- A fetch-based SSE parser could send Authorization headers, but browser and proxy behavior for long-lived fetch streams needs a dedicated backend/frontend validation gate. It is not necessary to safely enable backend auth for the dashboard because REST queries already support Bearer headers.

This gate should make the current frontend safe and explicit under auth, not invent new backend auth transport.

## Functional Requirements

### 1. Auth-Aware Stream Subscription

Update `web/src/api/stream.ts` and/or `DashboardStreamBridge` so streaming behavior depends on frontend auth state.

When `VITE_SIGNALOPS_AUTH_ENABLED=false`:

- Preserve current native `EventSource` behavior.
- Existing stream tests should continue to pass.

When `VITE_SIGNALOPS_AUTH_ENABLED=true`:

- Do not create native `EventSource` for `/v1/streams/dashboard`.
- Do not add `access_token`, `token`, `authorization`, or similar query parameters.
- Do not attempt to use `EventSource` with Bearer headers; it cannot work in native browser APIs.
- Mark stream state as intentionally disabled or auth-fallback mode.
- Ensure dashboard REST queries continue to run after login and continue to refresh using normal query invalidation/polling.

Implementation options:

- Add an auth-aware guard inside `DashboardStreamBridge`.
- Or add a `subscribeDashboardStream` option such as `enabled`/`authEnabled` and return a no-op subscription when auth is enabled.
- Or add `streamMode()` helper that returns `eventsource` when auth is disabled and `rest_fallback` when auth is enabled.

Prefer the least invasive implementation that keeps responsibilities clear and testable.

### 2. REST Fallback Refresh

When auth is enabled and SSE is disabled, dashboard widgets must still update reasonably.

Minimum behavior:

- Existing TanStack Query hooks should continue to fetch through the authenticated API client.
- Add `refetchInterval` to high-level dashboard queries that previously relied heavily on SSE invalidation, or add a dashboard-level interval that invalidates relevant query prefixes.
- Keep intervals modest to avoid noisy backend load.

Recommended interval:

```text
15 seconds for dashboard operational summaries
```

Candidate query prefixes to refresh under auth fallback:

```text
healthz
readyz
runs
raw-events
provider-usage
catalog-sources
catalog-pipelines
catalog-rules
normalized-events
signals
alerts
insights
```

Health/readiness already poll. Avoid duplicating those if current hooks already refresh them.

Acceptance for fallback:

- Dashboard remains useful without SSE.
- Stream status should not display as an alarming broken connection simply because auth fallback disabled SSE intentionally.
- If the UI shows stream state, it should distinguish `stream disabled under auth; REST refresh active` from a real disconnected stream.

### 3. UI State

The current UI uses `useUi` stream state and dashboard health/status panels.

When auth-enabled fallback is active:

- Do not show a persistent error like `dashboard stream disconnected` solely because SSE was intentionally not opened.
- Prefer neutral wording if visible, such as `REST refresh` or `Stream disabled under auth`.
- Keep the UI compact; do not add explanatory panels or tutorials.

If no visible stream indicator exists, it is acceptable to only avoid setting the stream error.

### 4. Tests

Add tests for auth-aware stream behavior.

Minimum expected coverage:

1. Existing auth-disabled stream behavior:
   - Native `EventSource` is created with the same URL/channels as today.
   - Events still parse and close as before.

2. Auth-enabled stream behavior:
   - Native `EventSource` is not constructed.
   - No token appears in the stream URL.
   - A no-op subscription can close safely.
   - UI/store state is not set to a disconnected error just because auth fallback is active.

3. REST fallback refresh behavior:
   - If implemented in `DashboardStreamBridge`, test that auth-enabled mode schedules/executes query invalidation or uses query refetch intervals.
   - If implemented via query hook intervals, test the config/helper that chooses intervals.

4. API auth behavior should remain covered:
   - Existing G053 tests for Bearer attachment and `X-SignalOps-Actor` suppression must continue passing.

### 5. Browser Validation Script/Checklist

Add a repeatable validation checklist for the real-browser auth flow. This may be a markdown doc or a script-assisted checklist under `docs/frontend`.

Recommended file:

```text
docs/frontend/auth_browser_validation_checklist.md
```

It should cover:

- Build auth-enabled frontend image with:

```bash
VITE_SIGNALOPS_AUTH_ENABLED=true \
VITE_SIGNALOPS_AUTH_ISSUER=https://auth.syncratic.co/realms/syncratic \
VITE_SIGNALOPS_AUTH_REALM=syncratic \
VITE_SIGNALOPS_AUTH_CLIENT_ID=signalops-web \
VITE_SIGNALOPS_AUTH_AUDIENCE=signalops-api \
docker compose -f compose.yaml -f compose.traefik.yaml build web
```

- Redeploy frontend only for validation, while keeping backend `SIGNALOPS_AUTH_ENABLED=false` initially.
- Open `https://signalops.syncratic.io` in a real browser.
- Confirm unauthenticated user sees compact Sign in screen.
- Click Sign in and confirm redirect to Syncratic IdP.
- Sign in as `lukeb`.
- Confirm callback returns to SignalOps.
- Confirm operator identity is displayed in shell.
- Confirm tenant resolves to `tenant-local`.
- Confirm role-derived lifecycle permissions are admin/operator capable.
- Confirm `/v1/*` fetch requests include `Authorization: Bearer ...` in browser devtools.
- Confirm dashboard data loads.
- Confirm dashboard stream does not produce a persistent auth-related error under auth fallback.
- Confirm logout redirects/clears session and returns to sign-in state.
- Restore auth-disabled frontend image after validation unless backend auth enablement is being coordinated immediately.

The checklist should explicitly state that headless automation may be blocked by Imperva/Incapsula and that this validation requires a real browser session.

### 6. Documentation and Audit

Update:

```text
docs/build_journal.md
docs/gate_audit.md
```

Record:

- Files changed.
- Auth-disabled and auth-enabled behavior decisions.
- Tests run.
- Build results.
- Browser validation status.
- Whether auth-enabled frontend was redeployed temporarily or only build-tested.
- Whether the default deployed image was restored to auth-disabled.

## Out of Scope

Do not implement these in G054:

- Backend auth changes.
- Permanent `SIGNALOPS_AUTH_ENABLED=true` backend deployment.
- Query-string access tokens for SSE.
- Cookie/session auth at Traefik or nginx.
- Fetch-stream SSE parser unless explicitly chosen and thoroughly validated as part of a separate backend-compatible gate.
- IdP role/client changes.
- Durable lifecycle audit history.
- TimescaleDB migration.
- Dashboard redesign.

## Acceptance Criteria

G054 is complete when:

- Auth-disabled native `EventSource` behavior remains intact.
- Auth-enabled frontend no longer attempts native `EventSource` to protected `/v1/streams/dashboard`.
- No access token is placed in a URL for streaming.
- Dashboard uses REST fallback refresh when auth is enabled.
- Stream UI state does not show an intentional auth fallback as a broken stream error.
- Tests cover auth-disabled streaming and auth-enabled fallback behavior.
- `npm test`, `npm run build`, and `npm audit --json` pass.
- Auth-enabled image build path passes.
- Real-browser validation checklist exists and is filled out as far as the frontend-agent can execute.
- Journal and gate audit are updated with UTC timestamped evidence.

## Suggested Implementation Order

1. Add stream mode helper based on `authConfig.authEnabled`.
2. Update `DashboardStreamBridge` to skip native SSE and use REST refresh when auth is enabled.
3. Add/adjust query invalidation or refetch intervals for dashboard fallback.
4. Update stream tests for both modes.
5. Add the browser validation checklist document.
6. Run `npm test`, `npm run build`, `npm audit --json`.
7. Build auth-disabled web image and validate current app still works.
8. Build auth-enabled web image as a production enablement check.
9. If a real browser session is available, execute the checklist; otherwise document it as pending operator validation.
10. Restore auth-disabled deployed image unless backend auth enablement is being coordinated immediately.
11. Update journal and audit.
