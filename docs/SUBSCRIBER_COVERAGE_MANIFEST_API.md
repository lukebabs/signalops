# Subscriber coverage manifest API

## Purpose

SignalOps owns the canonical asset catalog and tenant watchlist membership. A tenant administrator can select eligible assets from
the centrally governed warm cohort; the selection is recorded once in the tenant-default list and creates a governed hot-coverage
activation request. Existing legacy memberships are immutable from this surface and remain the tenant-local default.

The API is intentionally a resolved manifest rather than a second catalog. External consumers such as Syncratic receive the same
tenant-scoped selection that MarketOps uses, with no provider polling and no duplicated asset records.

## Read-only integration endpoint

`GET /v1/tenants/{tenant_id}/marketops/subscriber/coverage-manifest`

The caller must present an OIDC access token with the `signalops-api` audience, a matching `tenant_id` claim, and either the
`signalops:subscriber_catalog_reader`, `signalops:viewer`, `signalops:operator`, or `signalops:admin` role. The dedicated reader
role is intended for a service client. The tenant claim is always authoritative; a caller cannot read another tenant by changing the
path.

The response is `subscriber.coverage-manifest.v1` and includes the selected list, generated time, asset count, canonical global
asset IDs, and each asset's coverage tier. The `ETag` header is a SHA-256 representation of the response. Consumers should send
`If-None-Match` and treat `304 Not Modified` as a successful no-change poll. Responses are private and may be cached for 30 seconds.

## Tenant-admin selection endpoint

`GET /v1/tenants/{tenant_id}/marketops/subscriber/admin/warm-catalog?list_id=...&preset=snp500_top200&limit=1000`

`POST /v1/tenants/{tenant_id}/marketops/subscriber/admin/tenant-default-catalog-memberships`

These routes require the existing tenant administrator primitive (`super_admin`/`signalops:admin`). The bulk request accepts up to
400 global asset IDs and is idempotent. The UI's “ranked top 200” preset is based on the current governed market-cap ranking
snapshot. Until an independently sourced S&P 500 membership snapshot is loaded, it must be described as a ranked warm-catalog
proxy, not as an official index constituent list. The API does not silently claim index membership.

Adding a row writes the normal watchlist audit record and queues the existing global coverage activation request with reason
`subscriber_tenant_default_hot_asset`. It does not call Massive/FMP or bypass scheduler controls. Legacy rows are never removed or
replaced.

## Operational contract

- The database is the authority for catalog identity, list membership, entitlement, and coverage state.
- The tenant-default list remains the default view for tenant-local users; private lists remain subject-owned.
- A selected catalog asset is hot for intraday demand aggregation; the global warm cohort remains centrally governed.
- The manifest is read-only and safe for downstream cache refreshes.
- Production currently provisions the dedicated Keycloak client `signalops-subscriber-catalog-reader` with client-credentials
  enabled. Its realm role is `signalops:subscriber_catalog_reader`, its audience is `signalops-api`, and its tenant claim is
  hardcoded to `tenant-local`. Create one separately scoped client per additional tenant; do not broaden this client to accept a
  caller-supplied tenant claim.
