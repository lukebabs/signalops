# Mesh-3 authenticated Keycloak staging-route parity blocker — 2026-09-13

Status: partially remediated after Keycloak migration to k3s. Public OIDC discovery and JWKS are verified; authenticated SignalOps staging parity is now blocked by the HTTP-only staging route because browser PKCE requires a secure context. No SignalOps production traffic cutover.

Recorded: 2026-09-13 UTC.

## Scope

Mesh-3 is the next gate after Mesh-2. It attempts to prove that an existing SignalOps QA identity can authenticate through the Istio-routed staging host and return to the SignalOps `/auth/callback` route without moving production DNS.

The attempted smoke did not create users, did not register accounts, did not call Stripe, did not run provider polling, and did not change Keycloak configuration.

## What passed before the blocker

The smoke confirmed the previously closed Mesh-2 infrastructure still works:

- staging app manifests validate;
- Mesh-2 route manifests validate;
- `signalops-web` and `signalops-gateway` roll out in `signalops-app`;
- `signalops-staging-route` remains accepted by `istio-system/public-ingress`;
- backend references resolve;
- cleanup scales staging app workloads back to zero.

Post-failure cleanup state:

```text
signalops-web       0/0
signalops-gateway   0/0
```

## Blocker observed

The browser loaded the SignalOps SPA through the Istio staging route. When the user clicked Sign in, the OIDC client attempted to fetch Keycloak discovery metadata and failed before reaching the Keycloak login form.

Minimal HAR inspection, with query strings/cookies/tokens omitted, showed:

```text
GET 200 http://signalops-staging.syncratic.co:<local-port>/marketops/dashboard
GET 200 http://signalops-staging.syncratic.co:<local-port>/assets/index-*.js
GET 200 http://signalops-staging.syncratic.co:<local-port>/assets/router-*.js
GET 200 http://signalops-staging.syncratic.co:<local-port>/assets/index-*.css
GET -1 https://auth.syncratic.co/realms/syncratic/.well-known/openid-configuration
```

A direct retry from the host returned an Imperva/Incapsula challenge response instead of OIDC JSON:

```text
HTTP/2 503
retry-after: 5
content-type: text/html
...
Request unsuccessful. Incapsula incident ID: <captured in runtime output>
```

This means the authenticated parity gate is blocked before callback URI validation. The immediate issue is not a bad SignalOps route and not a QA password. It is that the browser/OIDC client cannot reliably read Keycloak discovery metadata from `https://auth.syncratic.co/realms/syncratic/.well-known/openid-configuration`.

## Source-controlled guard added

Added:

```bash
scripts/verify_keycloak_oidc_discovery_reachability.sh
scripts/run_k8s_mesh3_keycloak_staging_route_smoke.sh
python/tests/test_k8s_mesh3_keycloak_staging_route_parity.py
```

The Mesh-3 runner now checks OIDC discovery/JWKS reachability before scaling Kubernetes app workloads. If Keycloak discovery returns a WAF/challenge page or non-200 status, or if browser PKCE cannot run in a secure staging context, it fails closed and avoids a misleading auth test.

## Required remediation

Before Mesh-3 can pass, the SignalOps staging route must provide a secure browser context for PKCE. Keycloak/OIDC endpoints are now reachable:

1. Keep unauthenticated GET access to Keycloak OIDC discovery and JWKS endpoints through the CDN/WAF:
   - `/realms/syncratic/.well-known/openid-configuration`
   - `/realms/syncratic/protocol/openid-connect/certs`
2. Confirmed 2026-09-13: those endpoints return JSON metadata/keys, not an HTML WAF challenge.
3. Decide the staging callback strategy:
   - add `http://signalops-staging.syncratic.co:<smoke-port>/auth/callback` only for local-port smoke is not practical long term; or
   - create a stable staging DNS/TLS hostname and Keycloak redirect/web-origin entries; or
   - create a dedicated staging Keycloak client with equivalent claims/audience/roles.
4. Rerun:

```bash
scripts/run_k8s_mesh3_keycloak_staging_route_smoke.sh
```

Until the secure staging callback path closes, `pending_authenticated_keycloak_mesh_route_parity=true` remains correct.
