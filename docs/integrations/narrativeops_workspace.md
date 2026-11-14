# SignalOps Integration — NarrativeOps Workspace

## Public workspace

NarrativeOps is mounted as an independently deployed SignalOps workspace at:

```text
https://signalops.syncratic.io/narrativeops
```

The SignalOps shell owns discovery and navigation. NarrativeOps owns narrative formation, evidence admission, replay, market validation, and operations APIs.

## Request paths

| Public path | Internal target |
|---|---|
| `/narrativeops/` | NarrativeOps frontend SPA |
| `/narrativeops/healthz` | NarrativeOps `/healthz` |
| `/narrativeops/readyz` | NarrativeOps `/readyz` |
| `/narrativeops/api/v1/...` | NarrativeOps `/v1/...` |

The frontend uses `/narrativeops/api` as its production API base path and uses `/v1` directly through the Vite proxy in local development.

## Authentication

Production uses the shared SignalOps OIDC session and `signalops-web` client. NarrativeOps validates the bearer token against the Syncratic issuer, `signalops-api` audience, JWKS, tenant claim, and administrator role. The frontend redirects unauthenticated users to SignalOps login and performs one silent-renew retry after `401`.

Local development remains mock/header based. `X-Tenant-ID` is accepted only in development/test mode; it is not a production identity mechanism.

### Syncratic narrative reader (2026-11-13)

The dedicated Keycloak client `syncratic-narrativeops-reader` is enabled for
`client_credentials` with `aud=signalops-api`, `tenant_id=tenant-local` and only
the `narrativeops:reader` realm role. It has no browser, password-grant or
implicit flow and no SignalOps admin role. Its secret was not printed or
persisted in the integration repository. SignalOps' HTTPRoute forwards the
mounted API path directly to NarrativeOps; NarrativeOps verifies the signed
token and applies the exact-client, GET-only route guard.

The reader may use only the narrative list, narrative detail, evidence,
timeline and history GET routes under `/narrativeops/api/v1/narratives`.
Operations, processing and context-query routes retain the existing admin
requirement. NarrativeOps derives the tenant from the validated token and
replaces any caller-supplied `X-Tenant-ID` header. The local content-silent
probe reads a bounded page and outputs only aggregate state; it does not
access NarrativeOps Redis or PostgreSQL.

Live acceptance: API 2/2 Ready; missing bearer returned 401; the reader was
denied on operations and processing routes with 403; a forged tenant header
did not widen the returned tenant. One 25-item page was visible, with 20
generated and five deterministic-fallback synthesis statuses. This is a
sample, not a full-stream total or a claim about narrative quality.

The SignalOps **Administration → Access** source change adds a dedicated
`Syncratic NarrativeOps reader` control for tenant-local super-admins. It
reads the exact service client's Keycloak role status and can idempotently
grant only `narrativeops:reader` after checking its service-only settings,
audience and tenant mappers. The operator's bearer token must itself have the
required Keycloak realm-management permission; the UI never reads or shows a
client secret. This source change is not a deployment or a Syncratic credential
installation. Syncratic still needs its own approved Secret reference before
the consumer can authenticate.

## Reading the SignalOps asset list

NarrativeOps should use SignalOps as the authority for the tenant's resolved asset selection. Do not maintain a second asset catalog
or infer membership from ticker symbols. The read-only manifest endpoint is:

```text
GET https://signalops.syncratic.io/v1/tenants/{tenant_id}/marketops/subscriber/coverage-manifest
```

For the initial tenant-local integration, use the dedicated OIDC service client
`signalops-subscriber-catalog-reader`. Obtain a short-lived access token from
`https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/token` using `client_credentials`, request the
`signalops-api` audience, and send the bearer token to the manifest endpoint. The token must carry
`tenant_id=tenant-local` and `signalops:subscriber_catalog_reader`; the URL tenant and token tenant must match. Client secrets
must remain in NarrativeOps/Kubernetes secret storage and must never be placed in browser code or this document.

Example read:

```bash
curl -fsS \
  -H "Authorization: Bearer $SIGNALOPS_COVERAGE_READER_TOKEN" \
  'https://signalops.syncratic.io/v1/tenants/tenant-local/marketops/subscriber/coverage-manifest'
```

The `subscriber.coverage-manifest.v1` response contains `list_id`, `generated_at`, `asset_count`, and `assets[]`. Each asset
includes the stable `global_asset_id`, canonical `ticker`, company name, asset type, exchange, sector, eligibility status,
coverage state, coverage mode, coverage tier, and `added_at`. Use `global_asset_id` for joins and narrative evidence; ticker is
display metadata only.

The response is a read-only projection. It does not trigger provider polling, create watchlist memberships, or alter SignalOps
schedules. Cache the response briefly and send its `ETag` as `If-None-Match` on the next poll; `304 Not Modified` is a successful
no-change result. A tenant-specific NarrativeOps deployment must use a separately scoped reader client rather than changing the
tenant path or reusing the tenant-local credential.

## Deployment

The Kubernetes package in `deploy/k8s/` provides the API, frontend, configuration, services, namespace, and HTTPRoute. Images are intended to be published as:

- `ghcr.io/syncratic-inc/narrativeops-api:<tag>`
- `ghcr.io/syncratic-inc/narrativeops-frontend:<tag>`

The current backend remains an in-memory/local qualification implementation. Durable persistence, production backup/restore, and external artifact publication remain separate readiness gates.

## StreamRecorder boundary

StreamRecorder remains upstream. Its immutable `TranscriptionArtifact` output is admitted by NarrativeOps as an `EvidenceArtifact` through the producer/artifact boundary. NarrativeOps must not import StreamRecorder storage or depend on its database schema.
