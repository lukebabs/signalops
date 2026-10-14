# Mesh-3 authenticated Keycloak staging-route parity — 2026-09-13

Status: closed for staging. Production cutover remains not approved.

Recorded: 2026-09-13 UTC.

## Scope

Mesh-3 proved that an existing SignalOps QA identity can authenticate through the Istio-routed SignalOps staging host, complete the Keycloak redirect/callback journey, and reach the SignalOps enrollment resolver without moving production DNS.

The smoke did not create production users, did not run provider polling, did not call Stripe checkout, and did not cut over production traffic. Staging app workloads were scaled back to zero by the runner.

## Fixes required before closure

The gate exposed and closed four issues:

1. Public Keycloak OIDC discovery initially returned a challenge/503. After Keycloak moved to k3s, discovery and JWKS returned HTTP 200 JSON.
2. Browser PKCE failed over the original HTTP-only staging route. A staging-only HTTPS listener was added to `istio-system/public-ingress` with a self-signed `signalops-staging.syncratic.co` certificate and cross-namespace `ReferenceGrant`.
3. Keycloak rejected the staging callback URI. The `signalops-web` client was reconciled with staging redirect URIs, web origin, and post-logout URLs.
4. SignalOps callback initially returned 404 because `/auth/*` was routed to the gateway. The staging HTTPRoute now leaves `/auth/callback`, `/auth/silent-renew`, `/auth/signed-out`, and `/auth/login` on the SPA/web backend while keeping `/v1`, `/healthz`, and `/readyz` on the gateway.

The final remaining runtime issue was `access_resolution_failed` from `/v1/session/enrollment`. Root cause: OpenBao app staging runtime still pointed at the older `syncratic-runtime-smoke` database while the app was being tested against the dedicated MarketOps staging data boundary. The Mesh-3 runner now creates a non-production app runtime env from the dedicated staging DB secrets, writes it to OpenBao, and bootstraps only the minimal enrollment/access/subscription schema needed for the authenticated staging smoke.

## Passing evidence

```text
signalops_k8s_mesh3_keycloak_route_smoke_verified
namespace=signalops-app
staging_hostname=signalops-staging.syncratic.co
gateway=istio-system/public-ingress
gateway_ip=192.168.2.233
https_listener=signalops-staging-https
authenticated_keycloak_redirect=true
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
```

Playwright result:

```text
1 passed in 1.15s
```

## Source-controlled assets

- `deploy/kubernetes/staging/mesh-route/signalops-staging-tls.yaml`
- `deploy/kubernetes/staging/mesh-route/signalops-istio-staging-route.yaml`
- `scripts/provision_k8s_signalops_staging_https_listener.sh`
- `scripts/reconcile_keycloak_signalops_staging_client.sh`
- `scripts/create_k8s_signalops_app_runtime_staging_env.sh`
- `scripts/bootstrap_k8s_staging_enrollment_schema.sh`
- `scripts/run_k8s_mesh3_keycloak_staging_route_smoke.sh`
- `python/tests/test_k8s_mesh3_keycloak_staging_route_parity.py`

## Important operational note

A diagnostic command accidentally printed part of a non-production staging primary DB URL. The staging primary DB password was immediately rotated, the Kubernetes secret was updated, OpenBao app runtime was refreshed, and the gateway was restarted before the passing smoke. No production database credential was involved.

## Remaining Kubernetes production-readiness gates

Mesh-3 closes authenticated app parity through Istio. Production cutover remains blocked by the broader gates: full MarketOps scheduler parity, Signal-Connect ingestion shadow, ingress/DNS rollback plan, Stripe webhook parity through the K8S route, and capacity/load validation.
