# SignalOps SaaS Kubernetes Platform Architecture

Status: accepted target architecture for production SaaS planning.

## Decision

SignalOps will be operated as one SaaS platform with multiple independently scalable Kubernetes operational planes.

Signal-Connect is not a separate product boundary. Signal-Connect is the SignalOps ingestion subsystem. It should be deployed as a first-class SignalOps platform plane because ingestion has different scaling, security, rate-limit, and failure characteristics from browser/API traffic and MarketOps analytical jobs.

The design objective is not easiest deployment. The design objective is scalable, secure, observable production operation.

## Target namespace model

| Namespace | Primary responsibility | Scaling driver | Isolation reason |
|---|---|---|---|
| `signalops-app` | Web UI, public gateway/API, admin workbench, subscription/billing API surfaces | User/API traffic, browser fanout, Stripe/enrollment traffic | Protect user-facing services from ingestion and batch-job contention. |
| `signalops-connect` | Signal-Connect ingress, webhook receivers, connector workers, accepted-raw outbox | External source volume, webhook bursts, connector sync cadence | External-facing ingress requires separate rate limits, secrets, and failure handling. |
| `signalops-marketops` | MarketOps workers, provider polling, EOD/intraday jobs, SAF workers, Syncratic Intelligence workers | Market sessions, warm/hot asset count, analytical workload | Batch/market-data jobs must scale independently from user-facing APIs. |
| `signalops-cyberops` | CyberOps normalizers, detectors, CyberOps-specific workers | Cyber telemetry volume and detector workload | CyberOps should not contend with MarketOps jobs or subscriber UI traffic. |
| `signalops-identity` | Keycloak, realm/client reconciliation jobs, identity-adjacent controls | Login/enrollment/token-refresh load | Identity needs stricter security cadence, secrets, and availability posture. |
| `signalops-data` | PgBouncer, database access proxies, broker access helpers, backup/restore jobs if self-managed | Connection volume, backup/recovery operations, queue access | Data-plane access must be tightly governed and observable. |
| `signalops-observability` | Prometheus, Grafana, OTel collector, exporters, alerting | Metrics/log/trace volume and retention | Observability must survive app-plane issues and provide cross-plane diagnostics. |

This namespace split does not mean separate products. It means one platform with independent operational planes.

## Logical platform model

```text
SignalOps SaaS Platform
├── signalops-app
│   ├── web
│   ├── gateway
│   ├── admin workbench
│   └── subscription / billing APIs
├── signalops-connect
│   ├── ingestion gateway
│   ├── webhook receivers
│   ├── connector workers
│   ├── connector evidence persistence
│   └── accepted-raw outbox
├── signalops-marketops
│   ├── MarketOps scheduled jobs
│   ├── MarketOps provider workers
│   ├── Risk/Reward, EROC, EEOM, SRI workers
│   ├── SAF registrar / worker / outbox
│   └── Syncratic Intelligence workers
├── signalops-cyberops
│   ├── CyberOps normalizers
│   ├── CyberOps detectors
│   └── CyberOps alert/insight workers
├── signalops-identity
│   └── Keycloak
├── signalops-data
│   ├── PgBouncer
│   ├── broker access boundary
│   └── backup / restore controls
└── signalops-observability
    ├── Prometheus
    ├── Grafana
    ├── OTel collector
    ├── exporters
    └── alert manager
```

## Core principles

1. Signal-Connect is part of SignalOps, but it scales as its own ingestion plane.
2. User-facing API traffic must not contend directly with provider polling or batch analytics.
3. Scheduled market workflows must have explicit concurrency, retry, and completion contracts.
4. Tenant/user access control remains centralized at the gateway and data-access layer.
5. Provider polling remains centralized and deduplicated; browsers never trigger provider calls.
6. Keycloak is platform-critical infrastructure and should not be coupled to ordinary app deploy cadence.
7. Observability must be able to diagnose failures across all planes, including when the app plane is unhealthy.
8. Data access must be mediated through bounded pools, service identities, and explicit network policy.

## Expected scaling controls

| Plane | Required controls |
|---|---|
| App/API | Gateway replicas, HPA, readiness/liveness probes, request timeout, DB pool caps, route-level latency metrics. |
| Connect | Ingress rate limits, connector-specific queues, per-source credentials, idempotent outbox, DLQ, backpressure. |
| MarketOps | CronJobs with `concurrencyPolicy: Forbid` where applicable, worker resource limits, provider quota governance, job completion evidence. |
| CyberOps | Dedicated worker autoscaling, event-volume backpressure, separate detector resource limits. |
| Identity | Keycloak HA, realm config reconciliation, controlled admin credentials, token/session SLOs. |
| Data | PgBouncer, separate read/write pools, managed Postgres/Timescale preferred, broker partitions, PITR backup/rehearsal. |
| Observability | Cross-namespace scrape/trace/log permissions, alert rules, dashboards, retention policy. |

## Network policy posture

Default posture should be deny-by-default between namespaces, then explicitly allow required flows.

Required high-level flows:

```text
Browser -> Ingress -> signalops-app/web,gateway
signalops-app/gateway -> signalops-identity/keycloak
signalops-app/gateway -> signalops-data/pgbouncer
signalops-connect/* -> broker/data boundary
signalops-connect/outbox -> SignalOps raw-event topic
signalops-marketops/* -> provider APIs, broker, MarketOps DB/Timescale
signalops-cyberops/* -> broker, CyberOps DB projections
signalops-observability/* -> metrics/log/trace endpoints across namespaces
backup/restore jobs -> object storage and database endpoints
```

Signal-Connect should not receive broad database or Keycloak administrative permissions. It should write through its own ingress/evidence/outbox boundary and publish versioned raw-event contracts.

## Data-plane posture

For production SaaS, managed services are preferred where available:

- managed PostgreSQL for platform/identity/subscription/control data;
- managed Timescale-compatible time-series store or self-managed Timescale with tested backup/restore;
- managed Kafka/Redpanda-compatible broker or a properly operated Redpanda cluster;
- managed object storage for pgBackRest/PITR backup artifacts.

If databases/broker are self-hosted in Kubernetes, they should still be treated as the `signalops-data` plane with separate operational ownership, backups, and disaster-recovery tests.

## Source-derived workload inventory

The first infrastructure build artifact is [SignalOps Kubernetes Workload Inventory](kubernetes_workload_inventory.md). It classifies the current all-profile Compose topology into target planes, K8s workload kinds, statefulness, exposed ports, dependencies, and secret/config key classes. The initial Kubernetes base scaffold lives under `deploy/kubernetes/base` and currently contains namespaces, per-plane service accounts, default-deny NetworkPolicies, first-pass allow NetworkPolicies, and a planning-only secret-class policy.

## Initial migration path from Docker Compose

1. Produce a compose-to-K8s workload inventory for web, gateway, Signal-Connect, MarketOps workers, CyberOps workers, Keycloak, broker, databases, and observability.
2. Define Helm/Kustomize structure around the accepted namespace planes.
3. Add service accounts and secrets per workload class before moving traffic.
4. Add PgBouncer and explicit DB pool caps before scaling gateway replicas.
5. Move stateless workloads first: web, gateway, Connect workers, domain workers.
6. Convert host systemd scheduled jobs into Kubernetes CronJobs with equivalent guardrails.
7. Stand up K8s staging with production-like Keycloak, broker, database, and observability wiring.
8. Run parity checks between Docker production and K8s staging for Dashboard, Assets, Market State, Risk/Reward, EROC, EEOM, SRI, SAF, Syncratic Intelligence, subscriptions, and enrollment.
9. Run load tests and set supported concurrency from measured p95/p99 latency, DB saturation, queue lag, and error rate.
10. Cut over through ingress/DNS only after rollback and restore evidence exists.

## Production-readiness implications

This architecture adds the following production-readiness work:

- explicit gateway database pool limits;
- PgBouncer or equivalent connection pooling;
- namespace-aware RBAC and service accounts;
- namespace-aware NetworkPolicies;
- K8s CronJob conversion for MarketOps schedules;
- Keycloak HA and source-reconciled realm/client configuration;
- broker partition/retention/DLQ policy;
- observability dashboards and alerts across all planes;
- load testing before publishing concurrency limits;
- disaster-recovery rehearsal under the K8s topology.

## Non-goals for the first architecture pass

- Do not split Signal-Connect into a separate product or tenant platform.
- Do not move provider polling back into browser/API paths.
- Do not collapse every service into one namespace just for convenience.
- Do not claim large-scale B2C readiness until load testing and connection-pool controls exist.
- Do not migrate stateful data blindly; data-plane migration requires its own backup/restore and parity gate.
