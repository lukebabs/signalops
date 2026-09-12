# SignalOps Kubernetes Base Scaffold

Status: initial infrastructure scaffold; not yet a deployment target.

This base establishes the accepted SaaS operational planes before individual workloads are converted from Compose. It intentionally creates only namespaces, service accounts, NetworkPolicies, and non-secret policy ConfigMaps. Workload Deployments, CronJobs, Services, Ingress, Secrets, and data-plane migrations are separate gates.

## Planes

- `signalops-app`: web, gateway/API, admin, subscription/billing surfaces.
- `signalops-connect`: Signal-Connect ingestion subsystem.
- `signalops-marketops`: MarketOps scheduled jobs, provider workers, analytics, SAF, Syncratic Intelligence.
- `signalops-cyberops`: CyberOps workers and detectors.
- `signalops-identity`: Keycloak and identity reconciliation.
- `signalops-data`: database/broker access boundaries, backup/restore controls.
- `signalops-observability`: metrics, logs, traces, dashboards, alerting.

## Guardrails

- Default network posture is deny-by-default for every namespace.
- Service accounts are named by workload class, not shared globally.
- No broad application `.env` equivalent is introduced here; Kubernetes secret classes must be added per plane/workload class.
- Stateful workloads remain managed-service candidates until a dedicated data-plane migration and restore rehearsal gate is approved.


## Ingress and network policy assumptions

The target production Kubernetes edge should use Gateway API with an Envoy/Istio mesh ingress. The current base scaffold still includes a Traefik-compatible allow policy because Traefik is the Compose-era production edge and remains a rollback/reference path until the mesh gate lands. The namespace running the active edge gateway must carry the matching ingress-plane label before app ingress can admit traffic:

```yaml
signalops.syncratic.io/ingress-plane: traefik  # current rollback/reference edge; mesh gate will add envoy/istio label support
```

The scaffold uses standard Kubernetes NetworkPolicy. Standard NetworkPolicy cannot restrict egress by DNS name, so MarketOps provider/API egress is modeled as outbound TCP/443 to public IP space while excluding private RFC1918 ranges. If the production cluster uses Cilium, Calico Enterprise, or another policy engine with FQDN controls, replace this with explicit FQDN egress for Massive, FMP, Stripe, Syncratic AI Gateway, and other approved providers.

## Secret-class strategy

`secrets/secret-class-policy.yaml` is intentionally a ConfigMap, not a Secret. It records the required secret boundaries for the selected OpenBao backend. The current cluster has OpenBao Agent Injector, so workload conversion should use injector annotations and per-plane OpenBao Kubernetes auth roles.

## Next steps

1. Validate OpenBao KV mount, Kubernetes auth mount, and per-plane roles/policies; then add workload-specific injector annotations during Deployment/CronJob conversion.
2. Add a Mesh-0 implementation plan for Gateway API with Envoy/Istio ingress, then add staging Gateway/VirtualService or equivalent route manifests for `signalops.syncratic.io` web, `/v1/*`, `/auth/*`, and Stripe webhook routing.
3. Convert `web` and `gateway` first as stateless app-plane Deployments with probes and DB pool caps.
4. Convert MarketOps scheduled jobs to CronJobs with `concurrencyPolicy: Forbid` and DB-backed completion evidence.

See `docs/projects/subscriber_project/kubernetes_workload_inventory.md` for the source-derived workload inventory.
