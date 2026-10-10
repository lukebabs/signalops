# K8S-2 Staging App Parity Evidence — 2026-09-12

Status: closed for non-production SPA shell and web-to-gateway proxy parity.

## Scope

This gate validates that the Kubernetes staging app path can run the SignalOps web and gateway Deployments from private GHCR images, serve the SPA through the staging web service, and proxy gateway readiness endpoints through the Kubernetes-native nginx upstream.

This did not cut over production DNS, did not run provider polling, did not start MarketOps jobs, and did not migrate production secrets. Docker Compose remains the production authority.

## Controls

- Namespace: `signalops-app`
- Web service: `signalops-web`
- Gateway service: `signalops-gateway`
- Image source: private GHCR staging tags
- Runtime source: OpenBao staging app path `signalops/data/k8s/app/signalops-gateway-runtime-staging`
- Production label guard: `signalops.syncratic.io/production-cutover-allowed=false`
- Local validation path: bounded `kubectl port-forward` to `127.0.0.1` only
- Cleanup: staging web/gateway Deployments scaled back to zero

## Implementation changes

Added two staging-only NetworkPolicies because the first parity attempt proved static web serving worked but nginx could not reach the gateway service through the default-deny boundary:

- `allow-web-gateway-proxy-egress`: allows staging `signalops-web` egress to staging `signalops-gateway` on TCP 8080 and CoreDNS on TCP/UDP 53.
- `allow-web-gateway-proxy-ingress`: allows staging `signalops-gateway` ingress from staging `signalops-web` on TCP 8080.

Added repeatable parity automation:

- `scripts/run_k8s_staging_app_parity_smoke.sh`
- `python/tests/test_k8s_staging_app_parity.py`

## Validation evidence

Command:

```bash
scripts/run_k8s_staging_app_parity_smoke.sh
```

Result:

```text
signalops_k8s_staging_app_manifests_verified
services=2
deployments=2
ingresses=0
cronjobs=0
jobs=0
statefulsets=0
secrets=0
configmaps=1
networkpolicies=4
openbao_secret_path=signalops/data/k8s/app/signalops-gateway-runtime-staging
production_cutover_allowed=false
applied=false

deployment "signalops-gateway" successfully rolled out
deployment "signalops-web" successfully rolled out
1 passed in 0.37s

signalops_k8s_staging_app_parity_smoke_verified
namespace=signalops-app
base_url=http://127.0.0.1:18083
web_ready=true
gateway_proxy_healthz=true
gateway_proxy_readyz=true
authenticated_keycloak_redirect=false
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
```

Post-run cleanup check:

```text
signalops-web       0/0
signalops-gateway   0/0
```

## Explicit remaining boundary

Authenticated browser parity is not included in this gate. A localhost port-forward cannot complete the real Keycloak callback journey while the live client is configured for the production callback host. The next authenticated K8S UI gate needs one of the following:

1. a staging hostname with a matching Keycloak redirect URI, or
2. a dedicated staging Keycloak client/redirect policy.

Until that is approved, this gate proves staging app shell/proxy viability, not full tenant-authenticated UX parity.
