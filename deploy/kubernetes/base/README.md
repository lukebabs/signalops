# SignalOps Kubernetes Base Scaffold

Status: initial infrastructure scaffold; not yet a deployment target.

This base establishes the accepted SaaS operational planes before individual workloads are converted from Compose. It intentionally creates only namespaces, service accounts, and default-deny NetworkPolicies. Workload Deployments, CronJobs, Services, Ingress, Secrets, and data-plane migrations are separate gates.

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

The production Kubernetes edge should use Traefik. The namespace running Traefik must carry this label before `allow-traefik-to-web-and-gateway` can admit traffic:

```yaml
signalops.syncratic.io/ingress-plane: traefik
```

The scaffold uses standard Kubernetes NetworkPolicy. Standard NetworkPolicy cannot restrict egress by DNS name, so MarketOps provider/API egress is modeled as outbound TCP/443 to public IP space while excluding private RFC1918 ranges. If the production cluster uses Cilium, Calico Enterprise, or another policy engine with FQDN controls, replace this with explicit FQDN egress for Massive, FMP, Stripe, Syncratic AI Gateway, and other approved providers.

## Secret-class strategy

`secrets/secret-class-policy.yaml` is intentionally a ConfigMap, not a Secret. It records the required secret boundaries until the production secret backend is selected. The preferred SaaS path is External Secrets Operator backed by AWS Secrets Manager or equivalent managed secret storage.

## Next steps

1. Select the production secret backend and replace the policy ConfigMap with ExternalSecret, SealedSecret, or SOPS-managed secret manifests.
2. Add Traefik IngressRoute or standard Ingress manifests for `signalops.syncratic.io` web, `/v1/*`, `/auth/*`, and Stripe webhook routing.
3. Convert `web` and `gateway` first as stateless app-plane Deployments with probes and DB pool caps.
4. Convert MarketOps scheduled jobs to CronJobs with `concurrencyPolicy: Forbid` and DB-backed completion evidence.

See `docs/projects/subscriber_project/kubernetes_workload_inventory.md` for the source-derived workload inventory.
