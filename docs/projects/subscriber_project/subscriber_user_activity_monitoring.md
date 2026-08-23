# Subscriber User Activity Monitoring

Status: source implemented; production activation requires migration `000157_subscriber_user_activity_ledger` and gateway/web deployment.

Last updated: 2026-08-23.

## Purpose

Subscription Administration needs operational visibility into how enrolled users access MarketOps without turning the platform into a broad request-log collector.

This slice adds an append-only user-activity ledger for:

- login;
- logout;
- MarketOps feature/page views;
- MarketOps `POST`, `PUT`, and `DELETE` API mutations.

It intentionally does not record blanket `GET` reads, request bodies, response bodies, cookies, bearer tokens, or provider payloads.

## Data boundary

The source of truth is the dedicated MarketOps database table `subscriber_user_activity_events`.

Each event records:

| Field | Purpose |
|---|---|
| `tenant_id` | Signed tenant scope. |
| `subject` | Immutable OIDC subject. |
| `app_id` | Currently constrained to `marketops`. |
| `event_type` | `login`, `logout`, `feature_view`, or `api_mutation`. |
| `feature_key` | Coarse MarketOps area such as `dashboard`, `assets`, `subscriber`, or `valuation`. |
| `http_method` | Present for mutation events. |
| `route_path` | Normalized route path; tenant IDs are replaced with `{tenant}`. |
| `status_code` | HTTP outcome for mutation events. |
| `correlation_id` | Caller/header correlation where present, otherwise generated for session events. |
| `metadata` | Small operational metadata object only. |
| `occurred_at` | Server-side event timestamp. |

RLS restricts gateway access to the active `signalops.tenant_id`. Subscription administrators can query tenant activity only through the controlled Administration API.

## Recording paths

### Session beacons

The browser sends authenticated best-effort beacons to:

`POST /v1/session/activity`

The gateway derives tenant and subject from the signed token. The browser may submit only `login`, `logout`, or `feature_view`.

### Mutation capture

The gateway automatically records authenticated MarketOps mutations matching:

`/v1/tenants/{tenant}/marketops/*`

for `POST`, `PUT`, and `DELETE`.

Capture runs after the route handler so the event includes the final HTTP status code. Activity persistence is best-effort and must not block the user’s primary request.

## Administration surface

Subscription Administration now includes:

- a `User activity` tab with search, user summaries, and recent event rows;
- a `Selected user activity` drilldown under `Users & seats`.

The activity view supports operator search across user identity labels, feature, route, event type, status, and correlation ID.

## Privacy and security rules

- Do not store payloads, tokens, cookies, or provider responses.
- Do not use email as an authority key. Email/display name are labels resolved from `tenant_user_access`; the immutable subject remains authoritative.
- Do not capture broad read activity until there is a concrete operational need and a retention policy.
- Keep activity writes append-only.
- Treat the activity endpoint as telemetry, not an authorization source.

## Retention posture

The intended detail-retention target is 180 days. The first implementation creates the ledger and UI; a follow-up operations-control slice should add a scheduled retention job or extend the existing retention governance policy to prune detail rows after the approved window.

## Validation

Source validation for this slice:

- `go test ./internal/api ./internal/storage/postgres`
- `npm --prefix web run build`
- `python/tests/test_subscription_administration_ui_smoke.py` updated to require the Activity tab and selected-user drilldown.

Production validation after deployment should confirm:

1. migration `000157_subscriber_user_activity_ledger` is applied;
2. `/v1/session/activity` returns `202` for an authenticated user;
3. `/v1/administration/subscriptions/activity?tenant_id=tenant-local` returns summaries/events for a subscription admin;
4. Subscription Administration renders `User activity`;
5. a MarketOps watchlist mutation records an `api_mutation` event without exposing payload content.
