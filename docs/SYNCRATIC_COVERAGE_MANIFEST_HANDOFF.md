# Syncratic coverage manifest handoff

SignalOps exposes the tenant-local resolved coverage list for approved downstream consumers.

## Endpoint

```text
GET https://signalops.syncratic.io/v1/tenants/tenant-local/marketops/subscriber/coverage-manifest
```

The response is the tenant's current default coverage selection. Tenant-local now contains the additive union of the preserved
legacy 132 assets and the eligible governed S&P-200 selection (197 assets at migration `000189`). The preserved legacy SAF
cohort remains immutable evidence even though the operational default has expanded. When the
tenant administrator adds assets from the governed warm catalog, the same manifest expands without creating duplicate asset records.

## Authentication

Use OIDC client credentials against Keycloak:

```text
Token URL: https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/token
Client ID: signalops-subscriber-catalog-reader
Grant: client_credentials
Audience: signalops-api
Role: signalops:subscriber_catalog_reader
Tenant claim: tenant_id=tenant-local
```

The client secret must be exchanged through the approved secret channel; do not place it in source code, browser code, logs, or this
document. The client is intentionally hard-scoped to `tenant-local`. A different tenant requires a separately scoped service client.

Example token request:

```bash
TOKEN=$(curl -sS -X POST \
  'https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode grant_type=client_credentials \
  --data-urlencode client_id=signalops-subscriber-catalog-reader \
  --data-urlencode client_secret="$SIGNALOPS_COVERAGE_READER_CLIENT_SECRET" \
  | jq -r .access_token)
```

Fetch the manifest:

```bash
curl -fsS \
  -H "Authorization: Bearer $TOKEN" \
  'https://signalops.syncratic.io/v1/tenants/tenant-local/marketops/subscriber/coverage-manifest'
```

## Caching and refresh

The response includes an `ETag`. Store it and send it back on the next poll:

```text
If-None-Match: "<previous-etag>"
```

`304 Not Modified` means the manifest has not changed. Responses are private and have a short cache recommendation. The manifest is a
read-only projection; it does not trigger Massive/FMP polling or alter SignalOps schedules.

## Payload contract

The response schema is `subscriber.coverage-manifest.v1` and includes:

- `tenant_id`, `list_id`, `list_name`, `generated_at`
- `asset_count`
- `assets[]` with `global_asset_id`, `ticker`, `company_name`, `asset_type`, `exchange`, `sector`
- `eligibility_status`, `coverage_state`, `coverage_mode`, `coverage_tier`, and `added_at`

Use `global_asset_id` as the stable identity. Do not key integrations solely by ticker. SignalOps remains authoritative for identity,
eligibility, tenant membership, and coverage state.

## Current scope note

The Settings selector's ranked top-200 preset currently uses the governed market-cap ranking snapshot as a proxy. It is not an
official S&P constituent feed until an authoritative S&P 500 membership snapshot is loaded.

