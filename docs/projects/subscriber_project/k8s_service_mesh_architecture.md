# K8S service mesh architecture decision

Status: accepted target direction; implementation not yet applied to production.

Recorded: 2026-09-12.

## Decision

Because Syncratic-core, SignalOps, Signal-Connect, MarketOps, Keycloak, and platform observability are moving into the k3s stack, the Kubernetes target architecture should include a service-mesh layer rather than relying only on a simple ingress controller.

The preferred target is **Gateway API + Envoy-based mesh**. After inspecting the live k3s stack, the first implementation candidate is Cilium Gateway API / Envoy-first routing because Cilium and Cilium Envoy are already installed and Gateway API CRDs are present. Istio remains the leading service-to-service mesh candidate once the ingress/Gateway path is proven. The exact mode should be confirmed during Mesh-0:

- **Cilium Gateway API / Envoy-first** for the initial mesh ingress path, based on the current cluster baseline;
- **Istio ambient mode** if service-to-service mTLS and richer mesh telemetry should be introduced before production cutover;
- **Istio sidecar mode** only if ambient mode cannot satisfy traffic policy, telemetry, or mTLS requirements;
- **Envoy Gateway without full mesh** only as a fallback if Cilium Gateway API is not viable and full service-to-service mesh is deferred.

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
- Gateway API CRDs and/or Istio CRDs are present;
- no SignalOps production DNS or traffic changes;
- no namespace auto-injection on production namespaces until explicitly approved;
- baseline health and telemetry are visible.

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

The target production Kubernetes platform should use a service mesh. Mesh-0 now starts from the current Traefik-to-k3s bridge baseline and recommends Cilium Gateway API / Envoy-first routing before a broader Istio ambient decision. Until Mesh-2 and the broader K8S parity gates pass, Docker Compose/systemd remains the production authority and `production_cutover_allowed=false` remains correct.
