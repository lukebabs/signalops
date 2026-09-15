# Mesh-2 SignalOps staging route parity — 2026-09-13

Status: closed for unauthenticated SignalOps route parity through Istio; authenticated Keycloak callback parity remains pending.

Recorded: 2026-09-13 UTC.

## Scope

Mesh-2 creates a SignalOps staging-only HTTP route through the existing Istio Gateway API public ingress. This gate proves that the staged Kubernetes web and gateway services can be reached through Istio without moving production DNS or traffic.

This gate did not:

- move `signalops.syncratic.io` DNS;
- add a production TLS listener for SignalOps;
- change Keycloak redirect URIs or web origins;
- run provider polling;
- enable Kubernetes MarketOps schedules;
- enroll SignalOps namespaces into Istio sidecar or ambient dataplane mode;
- replace Docker Compose/systemd as production authority.

## Source-controlled migration assets

Added staging-only route overlay:

```text
deploy/kubernetes/staging/mesh-route/
├── allow-istio-public-ingress-to-signalops-app.yaml
├── kustomization.yaml
├── signalops-app-public-ingress-namespace-label.yaml
└── signalops-istio-staging-route.yaml
```

Added verifiers and smoke test:

```text
scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh
scripts/run_k8s_mesh2_signalops_staging_route_smoke.sh
python/tests/test_k8s_mesh2_signalops_staging_route.py
```

## Route design

The staging route uses hostname `signalops-staging.syncratic.co` and attaches only to the existing `istio-system/public-ingress` HTTP listener.

Routing rules:

- `/v1`, `/auth`, `/healthz`, and `/readyz` route to `signalops-gateway.signalops-app.svc:8080`;
- `/` routes to `signalops-web.signalops-app.svc:8080`.

The `signalops-app` namespace receives `syncratic.io/public-ingress=true` so the existing Gateway listener may accept routes from it. This is not an Istio injection label. SignalOps namespaces still have no `istio-injection` or `istio.io/dataplane-mode` enrollment.

The added NetworkPolicy allows ingress only from the Istio public gateway pod selector `gateway.networking.k8s.io/gateway-name=public-ingress` in `istio-system` to staging `signalops-web` and `signalops-gateway` pods on port `8080`.

## Manifest verifier evidence

```text
signalops_k8s_mesh2_route_manifests_verified
namespaces=1
httproutes=1
networkpolicies=1
ingresses=0
gateways=0
secrets=0
services=0
deployments=0
cronjobs=0
jobs=0
staging_hostname=signalops-staging.syncratic.co
parent_gateway=istio-system/public-ingress
listener=http
production_cutover_allowed=false
applied=false
```

## Live Playwright smoke evidence

The smoke applied the staging app overlay and Mesh-2 route overlay, scaled `signalops-web` and `signalops-gateway` to one replica, waited for rollout, verified the HTTPRoute was accepted and references resolved, port-forwarded `istio-system/public-ingress-istio:80`, mapped `signalops-staging.syncratic.co` to localhost in Chromium, and ran browser navigation through the Istio route.

Passing output:

```text
1 passed in 0.41s
signalops_k8s_mesh2_staging_route_smoke_verified
namespace=signalops-app
staging_hostname=signalops-staging.syncratic.co
port_forward=istio-system/public-ingress-istio:18180->80
route_accepted=true
route_refs_resolved=true
web_ready=true
gateway_proxy_healthz=true
gateway_proxy_readyz=true
authenticated_keycloak_redirect=false
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
```

Post-smoke cleanup verified:

```text
signalops-web       0/0
signalops-gateway   0/0
```

The HTTPRoute and NetworkPolicy remain applied as staging infrastructure, but with zero app replicas and no production DNS, they do not receive public SignalOps traffic.

## Remaining work

Mesh-2 closes unauthenticated route parity only. The next gate is authenticated Keycloak parity through a staging hostname/client configuration. That requires either:

1. a staging hostname with Keycloak redirect URI/web-origin allowance; or
2. a dedicated staging Keycloak client with the same claims, audience, roles, tenant behavior, token refresh behavior, and logout behavior.

Only after authenticated parity, broader scheduler parity, Signal-Connect shadow parity, rollback, and capacity evidence can a production cutover proposal be prepared.
