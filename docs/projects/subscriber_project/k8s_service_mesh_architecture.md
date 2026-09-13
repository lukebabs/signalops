# K8S service mesh architecture decision

Status: accepted target direction; Mesh-1 Istio control-plane readiness verified without SignalOps production traffic cutover.

Recorded: 2026-09-12.

## Decision

Because Syncratic-core, SignalOps, Signal-Connect, MarketOps, Keycloak, and platform observability are moving into the k3s stack, the Kubernetes target architecture should include a service-mesh layer rather than relying only on a simple ingress controller.

The selected target is **Gateway API + Istio/Envoy service mesh**. Cilium remains the CNI and NetworkPolicy foundation, while Istio becomes the platform mesh and Gateway API controller for the staged production migration. Mesh-1 was verified on 2026-09-13: `istio-base` and `istiod` are deployed, Istio CRDs are present, `GatewayClass/istio` is accepted, and `istio-system/public-ingress` is programmed at `192.168.2.233`.

Implementation mode:

- **Istio Gateway API ingress** for staging-route parity and future controlled production routing;
- **Istio ambient mode** as the preferred service-to-service mesh enrollment model when app, Connect, MarketOps, identity, and observability paths are ready for mTLS/telemetry enforcement;
- **Istio sidecar mode** only for namespaces or workloads that cannot meet policy/telemetry needs through ambient mode;
- **Cilium NetworkPolicy** remains the network guardrail baseline.

This is a platform decision, not a SignalOps-only edge decision. Signal-Connect remains part of SignalOps, and the mesh must cover app, ingestion, MarketOps, identity, data-access, and observability paths.

## Why mesh is justified

The platform is moving from a single-host Compose model into a multi-plane SaaS architecture. A mesh gives us a consistent control layer for:

- service identity and service-to-service mTLS;
- consistent timeout, retry, and circuit-breaker policies;
- canary and weighted routing during production cutover;
- cross-plane telemetry for app, Connect, MarketOps, Keycloak, and Syncratic AI Gateway interactions;
- safer rollback during migration from Docker Compose/systemd to Kubernetes;
- policy separation between user traffic, ingestion traffic, provider jobs, and admin/control traffic.

The cost is operational complexity. We accept that cost because scalability and controlled production operation are now the design drivers.

## Target traffic model

```text
Internet / CDN / WAF
  -> Kubernetes Gateway API listener
  -> Envoy/Istio ingress gateway
  -> signalops-app/web and signalops-app/gateway

signalops-app/gateway
  -> signalops-identity/keycloak
  -> signalops-data/pgbouncer or database access boundary
  -> Syncratic AI Gateway / Stripe / approved external APIs

signalops-connect/*
  -> broker/data boundary
  -> accepted-raw outbox and DLQ

signalops-marketops/*
  -> provider APIs
  -> MarketOps primary/temporal data stores
  -> Syncratic Intelligence / AI Gateway where approved

signalops-observability/*
  -> mesh telemetry, metrics, traces, logs
```

## Mesh adoption gates

### Mesh-0 — design and install plan

Acceptance:

- choose Cilium Gateway API first, Istio ambient first, Istio sidecar, or Envoy Gateway fallback;
- document supported k3s version and CNI compatibility;
- document resource overhead and node sizing assumptions;
- document rollback procedure to remove mesh labels/policies without breaking Compose production;
- no production traffic moved.

### Mesh-1 — non-production control plane install

Acceptance:

- mesh control plane is installed in a dedicated namespace;
- Gateway API CRDs and Istio CRDs are present;
- `GatewayClass/istio` is accepted;
- an Istio Gateway is programmed;
- no SignalOps production DNS or traffic changes;
- no namespace auto-injection or ambient enrollment on SignalOps namespaces until explicitly approved;
- baseline health is visible.

Status: closed on 2026-09-13. See [Mesh-1 Istio readiness evidence](k8s_mesh1_istio_readiness_2026-09-13.md).

### Mesh-2 — staging app mesh parity

Acceptance:

- `signalops-app` staging web/gateway traffic works through Gateway API / mesh ingress;
- `/healthz`, `/readyz`, and Playwright staging smokes pass;
- app-to-gateway and gateway-to-data/identity paths work under mesh policy;
- mTLS mode is recorded;
- production cutover remains false.

### Mesh-3 — MarketOps and Signal-Connect shadow parity

Acceptance:

- MarketOps CronJobs retain scheduler evidence under mesh policy;
- one no-provider and one approved provider smoke remain bounded;
- Signal-Connect staging ingress has rate limits/backpressure and accepted-raw evidence;
- telemetry identifies app, Connect, MarketOps, identity, and data-plane calls.

### Mesh-4 — production cutover proposal

Acceptance:

- Gateway hostnames, TLS, Keycloak redirects, Stripe webhook route, and rollback path are documented;
- traffic shifting/canary policy is defined;
- p95/p99 latency, error-rate, and saturation thresholds are measured;
- rollback returns traffic to Compose/systemd authority without data loss;
- named approval is required before any production traffic move.

## Guardrails

- Do not enable namespace-wide automatic injection for production workloads until that namespace has passed staging parity.
- Do not allow mesh retries on non-idempotent POST/PUT/DELETE routes unless the application has explicit idempotency protection.
- Do not let mesh-level retries create duplicate provider calls. MarketOps provider jobs must remain governed by job-level idempotency and no-browser-provider-call rules.
- Do not replace OpenBao recovery, database backup/restore, or application authorization with mesh policy. Mesh policy is an additional control layer.
- Do not move Keycloak traffic until redirect URIs, web origins, session cookies, and token refresh behavior are proven through browser tests.

## Current conclusion

The target production Kubernetes platform should use Istio as the service mesh and Gateway API controller. Mesh-1 proves the control plane exists and is healthy enough for staging-route parity. Until Mesh-2 and the broader K8S parity gates pass, Docker Compose/systemd remains the production authority and `production_cutover_allowed=false` remains correct.
