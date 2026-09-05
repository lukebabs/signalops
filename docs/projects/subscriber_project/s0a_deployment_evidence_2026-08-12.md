# S0-A Deployment Evidence — 2026-08-12

Status: all technical assertions passed; awaiting temporary-role teardown and formal security/platform approval.

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
| Bootstrap replay guard | Pass | On 2026-08-12, controlled subject `df0edb14-66ae-4594-bab7-97a803312e89` with `tenant_id=tenant-local` and the temporary provisioner role retried the pilot bootstrap. The endpoint returned `409 tenant_already_provisioned` with `target tenant already has access history`. No grant was created or modified. |

The HAR captured during validation is sensitive because it contains OIDC refresh-token exchanges. Retain it only in the approved evidence store, restrict access, and delete/redact working copies after review.

## Required teardown and approval

1. Remove `signalops:tenant_provisioner` from the controlled Chigos identity and invalidate the temporary session/token used for the replay.
2. Record the role-removal and session-invalidation confirmation in the approved evidence store.
3. Security and platform owners review this record plus the retained preflight, Keycloak, audit, and browser evidence, then approve S0-A closure.

## Approval decision

After the required temporary-role teardown is retained and the security and platform owners approve this record, S0-A may be marked complete. This approval enables S1 global-catalog shadow work only. It does not enable subscriber list features, global data projections, provider-budget expansion, or any tenant rollout flag.
