# Kubernetes/OpenBao staging cutover gate

Status: active production-readiness gate; no production workload cutover approved.

Recorded: 2026-09-12.

## Purpose

This gate turns the Kubernetes/OpenBao architecture from planning into a controlled staging path. The goal is to prove that SignalOps can run in Kubernetes with OpenBao-backed secrets, namespace isolation, scheduler parity, and browser/API parity before any production DNS or workload authority moves away from Docker Compose.

Docker Compose remains the live production authority until this gate passes and a separate cutover approval is recorded.

## Current evidence

OpenBao has been reported operational by the platform owner, but the repository read-only HA verifier did not pass on 2026-09-12:

```text
openbao_ha_readiness_failed: ready OpenBao server pods 1/3; HA gate requires at least 3 ready non-injector server pods
```

The verifier also surfaced k3s config permission warnings for `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. Those warnings did not prevent the cluster query, but they should be cleaned up before relying on automated K8s checks in CI or deployment agents.

## Entry requirements

Before deploying any SignalOps workload into Kubernetes staging, verify:

1. OpenBao HA surface passes:

   ```bash
   scripts/verify_openbao_ha_readiness.sh
   ```

2. OpenBao authenticated controls are proven by an operator without exposing secrets:
   - active/standby health is visible;
   - seal/unseal or auto-unseal recovery ownership is documented;
   - audit logging is enabled;
   - KV v2 SignalOps mount exists;
   - Kubernetes auth mount exists;
   - per-plane roles exist for `signalops-app`, `signalops-connect`, `signalops-marketops`, `signalops-cyberops`, `signalops-identity`, `signalops-data`, and `signalops-observability`;
   - cross-plane reads are denied.

3. The Kubernetes base render succeeds:

   ```bash
   kubectl kustomize deploy/kubernetes/base
   ```

4. No production secrets are committed to Git or rendered into plain Kubernetes `Secret` manifests by this repo.

## Staging sequence

### Stage K8S-1 — Secret and namespace foundation

Apply only the namespace, service-account, NetworkPolicy, and non-secret OpenBao annotation scaffolds to a staging cluster or staging namespace set.

Acceptance:

- seven SignalOps namespaces exist;
- default-deny NetworkPolicies are active;
- per-plane secret-reader service accounts exist;
- OpenBao injector annotations are present only on staging test pods;
- a staging test pod in each plane can read only its own non-production test secret path;
- cross-plane secret reads fail closed.

### Stage K8S-2 — Stateless app parity

Convert only `web` and `gateway` into staging Kubernetes Deployments behind non-production hostnames.

Acceptance:

- web and gateway pods use OpenBao-injected files or scoped environment projection;
- readiness/liveness probes pass;
- gateway DB pool caps are explicit;
- `/readyz` passes;
- Playwright smokes pass against the staging hostname for Dashboard, Watchlists, Assets, Pricing, Profile/Settings, and protected-route behavior;
- production Docker Compose remains unchanged.

### Stage K8S-3 — MarketOps worker shadow parity

Convert selected MarketOps jobs into Kubernetes CronJobs in shadow mode. Do not write production provider evidence until parity is approved.

Acceptance:

- CronJobs use `concurrencyPolicy: Forbid`, explicit deadlines, and bounded retry limits;
- job status writes to staging/dedicated operations tables;
- no browser path can trigger provider polling;
- warm/hot selector outputs match Docker authority for the same session;
- Admin Operations Health can show staging status without replacing production status.

### Stage K8S-4 — Signal-Connect ingestion shadow

Deploy Signal-Connect ingestion components as the ingestion plane, still part of SignalOps.

Acceptance:

- ingress rate limits and backpressure are configured;
- DLQ/outbox behavior is visible;
- Connect cannot read app/identity/MarketOps secrets;
- accepted raw events match contract expectations;
- no production detector or MarketOps consumer is switched until shadow evidence passes.

### Stage K8S-5 — Production cutover proposal

Only after K8S-1 through K8S-4 pass, prepare a separate cutover proposal.

Acceptance:

- rollback path is documented;
- backup/restore evidence is current;
- DNS/ingress change is isolated;
- Stripe webhook endpoint behavior is validated;
- Keycloak redirect URIs and web origins include the K8s hostnames;
- p95/p99 latency, DB connection saturation, queue lag, and error-rate thresholds are measured under load.

## Non-goals

- Do not move production traffic during this gate.
- Do not replace Docker Compose as production authority merely because manifests render.
- Do not migrate stateful databases into Kubernetes without a separate data-plane backup/restore gate.
- Do not broaden secret access through one global SignalOps role.
- Do not enforce SMS MFA as part of this gate; enrollment MFA remains a product/security decision outside K8s staging.

## Current blocker

The immediate blocker is OpenBao HA evidence. The current verifier requires at least three ready non-injector OpenBao server pods. If the intended production topology is a different fault-tolerant model, update the verifier and architecture documents with that explicit design before staging workload conversion.
