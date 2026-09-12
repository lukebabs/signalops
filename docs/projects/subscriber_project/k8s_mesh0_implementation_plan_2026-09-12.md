# Mesh-0 implementation plan — k3s service mesh baseline

Status: drafted from live cluster and current Syncratic-core Traefik evidence. No mesh install or production traffic cutover has been performed.

Recorded: 2026-09-12.

## Baseline observed

Current public edge and k3s routing are split across Docker Compose and Kubernetes:

- Docker Traefik is the current public front door for Syncratic-core and SignalOps.
- Traefik container image: `traefik:v3.6.6`.
- Traefik listens on host `0.0.0.0:80`, `0.0.0.0:443`, and dashboard `0.0.0.0:8081`.
- Traefik uses Docker provider with `exposedByDefault=false`.
- Traefik uses file provider at `/etc/traefik/dynamic`.
- TLS uses Let's Encrypt DNS challenge with GoDaddy provider.
- `websecure` trusts Imperva forwarded-header CIDR ranges.
- SignalOps Compose public route is attached through `compose.traefik.yaml` on external Docker network `syncratic-core_syncratic_net`.
- k3s has `ingress-nginx` exposed as LoadBalancer `192.168.2.233`, with NodePorts `30080` and `30443`.
- The current Syncratic-core dynamic Traefik bridge routes `portal.syncratic.co` to `https://host.docker.internal:30443` using `serversTransport` with `insecureSkipVerify=true`.
- k3s node services are visible on `192.168.2.5`, including Kubernetes API and Cilium-related listeners.
- k3s server version is `v1.33.5+k3s1`.
- Cilium is installed by Helm in `kube-system` at chart/app version `1.20.0`.
- Cilium Envoy is running.
- Gateway API CRDs are present.
- No `GatewayClass` is configured yet.
- No Istio CRDs are present.

## Interpretation

The platform already has a bridge pattern from Docker Traefik into k3s. That pattern should be preserved during migration because it gives us a rollback-friendly front door while Kubernetes matures behind it.

However, the target architecture is no longer a simple ingress controller. Since Syncratic-core, SignalOps, Signal-Connect, Keycloak, MarketOps, and observability are all moving into k3s, the target should be Gateway API plus an Envoy-based service mesh.

## Recommended implementation mode

Recommended first mode: **Cilium Gateway API / Envoy-first mesh entry**, followed by an Istio ambient evaluation.

Rationale:

- Cilium and Cilium Envoy are already present and healthy.
- Gateway API CRDs are already installed.
- No GatewayClass is currently configured, so the next step can be controlled and reversible.
- No Istio CRDs are present, so installing Istio now would add a second control plane before we have a measured need for full mesh features.
- Cilium can advance the Gateway API / Envoy ingress path and FQDN-aware/network-policy posture while keeping the operational blast radius smaller.
- Istio ambient remains attractive for mTLS, identity, telemetry, and service-to-service policy, but should be evaluated after the Gateway API ingress path is proven.

Fallback modes:

1. **Istio ambient first** if the platform needs mesh mTLS/telemetry before any production Kubernetes ingress movement.
2. **Istio sidecar** only if ambient cannot satisfy required policy/telemetry behavior.
3. **Envoy Gateway only** if Cilium Gateway API is not viable and full mesh is deferred.

## Mesh-0 acceptance criteria

Before installing or enabling any mesh/ingress replacement:

- choose the implementation mode: Cilium Gateway API first, Istio ambient first, or Envoy Gateway fallback;
- record whether Gateway API CRDs and GatewayClass exist;
- record current Traefik public edge and k3s ingress-nginx bridge behavior;
- preserve Docker Traefik as rollback/reference edge;
- define how `signalops.syncratic.io`, `auth.syncratic.co`, `portal.syncratic.co`, and Stripe webhook routes will be tested before DNS movement;
- define rollback to current Traefik -> nginx ingress / Compose route;
- no production traffic movement;
- no namespace auto-injection;
- no provider polling.

## Proposed staged execution

### Step 1 — Baseline verifier

Run:

```bash
scripts/verify_k8s_mesh0_baseline.sh
```

Expected current result:

```text
signalops_k8s_mesh0_baseline_verified
gateway_api_crds=present
gateway_classes=0
istio_crds_present=false
docker_traefik_edge=present_80_443
public_traffic_cutover_allowed=false
```

### Step 2 — Choose GatewayClass path

Preferred path:

- enable/configure the Cilium Gateway API controller if it is not already active;
- create a non-production GatewayClass/Gateway for staging only;
- do not bind production DNS;
- route only a staging hostname or local test path first.

### Step 3 — Create staging mesh/Gateway route

Create staging-only Gateway/HTTPRoute equivalents for:

- `signalops-web`;
- `signalops-gateway` `/healthz` and `/readyz`;
- eventually `/auth/*`, `/v1/*`, and Stripe webhook path after auth parity is planned.

### Step 4 — Browser and API parity

Run Playwright and API smokes through the mesh/Gateway route:

- public shell route;
- `/healthz` and `/readyz`;
- protected route login/callback through Keycloak-compatible staging hostname;
- Dashboard, Watchlists, Assets, Pricing, Profile/Settings, Syncratic Intelligence.

### Step 5 — Decide Istio ambient timing

After Gateway API ingress works, decide whether to add Istio ambient for service-to-service mTLS and telemetry before or after the first production cutover.

## Rollback posture

Rollback must remain simple during Mesh-0 and Mesh-1:

- Docker Traefik remains bound to public `80/443`.
- Existing Compose SignalOps route remains valid through `compose.traefik.yaml`.
- Existing k3s ingress-nginx bridge remains available at `host.docker.internal:30443` / `192.168.2.233:30443`.
- Mesh/Gateway staging routes must be removable without deleting SignalOps app/MarketOps data or OpenBao secret paths.
- Production DNS is not moved until a named cutover approval exists.

## Current conclusion

The next safe implementation step is not a full Istio install. It is to close Mesh-0 with a verified baseline and an implementation-mode decision. Based on the current cluster, the best first move is Cilium Gateway API / Envoy-first routing, with Istio ambient evaluated as the service-to-service mesh layer after Gateway API routing proves stable.
