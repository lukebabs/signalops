# Subscriber User Activity Monitoring

Status: production activated for the controlled pilot environment.

Last updated: 2026-08-23.

Production activation evidence, 2026-08-23 UTC:

- migration `000157_subscriber_user_activity_ledger` applied to the dedicated MarketOps database at `2026-08-23 06:57:56 UTC`;
- migration `000158_subscriber_user_activity_retention_policy` is prepared to add a 180-day dry-run retention policy for `tenant-local` and `tenant-pilot-b`;
- `subscriber_user_activity_events` exists with RLS enabled and forced;
- `signalops_subscriber_gateway` has `SELECT,INSERT` on the activity ledger and execute access to `subscriber_subscription_admin_identity_labels(text)`;
- gateway and web were rebuilt/restarted through the constrained deployment agent;
- public deployment and subscriber browser smokes passed;
- Subscription Administration smoke passed after tightening the test to distinguish the base subscription snapshot endpoint from the new activity endpoint;
- the live activity ledger recorded login and feature-view events, latest observed at `2026-08-23 07:04:01 UTC`.

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

The intended detail-retention target is 180 days. Migration `000158_subscriber_user_activity_retention_policy` adds the retention-governance policy `subscriber.user_activity_180d` for the controlled tenant scopes.

The policy is deliberately seeded as `dry_run`. Daily retention governance can now count candidate detail rows older than 180 days through the existing `signalops-retention-governance` scheduled job and Admin Storage > Retention Governance surface. No activity row is deleted unless the policy mode is explicitly changed to `enforced` and the retention governor is run with `--execute`.

Enforcement remains a separate approval gate because product/legal retention, summarized activity metrics, and customer-support needs must be confirmed before pruning detailed activity rows.

## Validation

Source validation for this slice:

- `go test ./cmd/retention-governor ./internal/api ./internal/storage/postgres`
- `npm --prefix web run build`
- `python/tests/test_subscription_administration_ui_smoke.py` updated to require the Activity tab and selected-user drilldown.

Production validation after deployment should confirm:

1. migration `000157_subscriber_user_activity_ledger` is applied;
2. `/v1/session/activity` returns `202` for an authenticated user;
3. `/v1/administration/subscriptions/activity?tenant_id=tenant-local` returns summaries/events for a subscription admin;
4. Subscription Administration renders `User activity`;
5. a MarketOps watchlist mutation records an `api_mutation` event without exposing payload content.

## Retention policy implementation — 2026-08-23

The retention governor now recognizes `subscriber.user_activity_180d` and maps it to `subscriber_user_activity_events.occurred_at`. The target is tenant-scoped and does not use CyberOps evidence receipts.

Deployment steps for this slice:

1. Apply migration `000158_subscriber_user_activity_retention_policy` to the dedicated MarketOps database.
2. Rebuild/redeploy the retention-governor image or run through the next production deployment path that rebuilds the image.
3. Run `signalops-retention-governance` in its existing dry-run mode and verify Admin Storage > Retention Governance shows candidate/affected counts. Expected affected rows remain `0` while the policy is `dry_run`.
