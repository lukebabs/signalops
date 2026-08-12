# S0-A Deployment Evidence — 2026-08-12

Status: evidence package prepared; awaiting the final replay assertion and formal security/platform approval.

## Scope

This record covers the deployed tenant-isolation and bootstrap controls for the `tenant-pilot-b` MarketOps pilot. It records outcomes only; it contains no bearer, refresh, or IdP administrative token.

## Verified evidence

| Control | Result | Evidence retained |
|---|---|---|
| OIDC tenant scope | Pass | Keycloak has one authoritative `tenant_id` user-attribute mapper per user. Existing users are mapped to `tenant-local`; the pilot is mapped to `tenant-pilot-b`. |
| Initial tenant bootstrap | Pass | A controlled identity with the temporary `signalops:tenant_provisioner` role created the first `marketops: read` grant for `tenant-pilot-b`. The persisted grant and immutable access audit were reviewed. The role was then removed and its issued token invalidated. |
| Pilot authenticated experience | Pass | The pilot session resolved `tenant-pilot-b`, `GET /v1/session/experience` returned 200, and MarketOps read endpoints returned 200 only for `tenant-pilot-b`. Empty results are expected before the S1 global-catalog projection exists. |
| Read-only mutation denial | Pass | `POST /v1/tenants/tenant-pilot-b/marketops/assets/onboard` returned `403 insufficient_role` to the pilot read user. |
| Cross-tenant denial | Pass | The pilot token sent to `POST /v1/tenants/tenant-local/marketops/assets/onboard` returned `403 tenant_mismatch` before a mutation could occur. |
| Browser renewal | Pass | The frontend renewal window was reduced from 300 seconds to 60 seconds and deployed in commit `9638fdf`, preventing immediate refresh-token churn for five-minute access tokens. |

The HAR captured during validation is sensitive because it contains OIDC refresh-token exchanges. Retain it only in the approved evidence store, restrict access, and delete/redact working copies after review.

## Final assertion to capture

1. Temporarily assign `signalops:tenant_provisioner` to the controlled bootstrap identity and obtain a fresh token.
2. Repeat `POST /v1/administration/tenant-provisioning/access` for `tenant-pilot-b`.
3. Confirm `409 tenant_already_provisioned`; retain the status, correlation/request identifier, and response body in the evidence store.
4. Remove the provisioner role again and invalidate the temporary session/token.
5. Record a request made without the provisioner role returning `403`. The exact error may be `insufficient_role` or `tenant_mismatch`, depending on whether the request also conflicts with the caller tenant.

## Approval decision

After the final assertion is retained, the security and platform owners may mark S0-A complete. This approval enables S1 global-catalog shadow work only. It does not enable subscriber list features, global data projections, provider-budget expansion, or any tenant rollout flag.
