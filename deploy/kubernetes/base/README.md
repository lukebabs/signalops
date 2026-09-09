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

## Next steps

1. Add explicit allow NetworkPolicies for browser ingress, gateway-to-identity, gateway-to-data, MarketOps-to-data/broker/provider egress, Connect-to-broker/data, and observability scrapes.
2. Add secret-provider strategy and placeholder ExternalSecret/SecretClass manifests.
3. Convert `web` and `gateway` first as stateless app-plane Deployments with probes and DB pool caps.
4. Convert MarketOps scheduled jobs to CronJobs with `concurrencyPolicy: Forbid` and DB-backed completion evidence.

See `docs/projects/subscriber_project/kubernetes_workload_inventory.md` for the source-derived workload inventory.
