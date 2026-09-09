# SignalOps Kubernetes Workload Inventory

Status: source-derived infrastructure build artifact for SaaS/Kubernetes migration planning.

Generated from the current Compose topology with all profiles enabled. Secret values are intentionally omitted; only key names and workload characteristics are documented.

## Summary

- `signalops-app`: 3 workloads — administration-notification-recorder, gateway, web
- `signalops-connect`: 3 workloads — cyberops-connect-outbox, cyberops-connect-persister, raw-worker
- `signalops-marketops`: 38 workloads — algorithm-runner, marketops-algorithm-adjudicator, marketops-asset-backfill-worker, marketops-eeom-runner, marketops-eroc-runner, marketops-intelligence-cohort-runner, marketops-intraday-monitor, marketops-options-coverage-runner, marketops-options-feature-materializer, marketops-postgres, marketops-postgres-migrate, marketops-retention-governor, marketops-signal-assurance-outbox, marketops-signal-assurance-registrar, marketops-signal-assurance-worker, marketops-sri-holdings-runner, marketops-sri-runner, marketops-syncratic-intelligence-runner, marketops-tactical-valuation-runner, marketops-timescaledb, marketops-timescaledb-migrate, marketops-valuation-runner, massive-puller, massive-scheduler, normalizer, replay-worker, signal-persister, subscriber-global-annual-financial-refresh, subscriber-global-annual-financial-task-worker, subscriber-global-annual-valuation-materializer, subscriber-global-catalog-admission, subscriber-global-eod-history-materializer, subscriber-global-eod-shadow-planner, subscriber-global-intraday-shadow-capture, subscriber-global-marketops-evidence-materializer, subscriber-global-marketops-parity-manifest, subscriber-global-ranking-import, subscriber-global-saf-benchmark-materializer
- `signalops-cyberops`: 5 workloads — cyberops-daily-feature-materializer, cyberops-detector, cyberops-hourly-feature-materializer, cyberops-iot-anomaly, cyberops-normalizer
- `signalops-identity`: 0 workloads — none in this Compose package yet
- `signalops-data`: 11 workloads — postgres, postgres-migrate, redpanda, redpanda-console, retention-governor, retry-replayer, storage-monitor, temporal-backfill, timescaledb, timescaledb-migrate, topic-bootstrap
- `signalops-observability`: 0 workloads — none in this Compose package yet

## Migration rules

- Treat this as the authoritative starting inventory for Helm/Kustomize planning; update it when Compose services are added, removed, or reclassified.
- Stateful Compose services are managed-service candidates first. If self-hosted in Kubernetes, they require storage classes, backup/restore, PodDisruptionBudgets, and separate restore rehearsals.
- Profile-based one-shot workers should become Kubernetes Jobs or CronJobs. Market-session schedules must use `concurrencyPolicy: Forbid`, explicit deadlines, retry limits, and DB-backed completion evidence.
- Secrets must be per-plane and per-workload-class. Do not reuse broad `.env` injection as the Kubernetes model.
- User-facing services must scale independently from ingestion, provider polling, and analytical workers.

## Workload inventory

| Workload | Target plane | K8s kind | State | Exposure | Profiles | Depends on | Secret/config keys |
|---|---|---|---|---|---|---|---|
| `administration-notification-recorder` | `signalops-app` | CronJob / Job worker | stateless | cluster-internal | administration-notifications | postgres | SIGNALOPS_DATABASE_URL |
| `algorithm-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `cyberops-connect-outbox` | `signalops-connect` | CronJob / Job worker | stateless | cluster-internal | cyberops | postgres, redpanda, topic-bootstrap | SIGNALOPS_DATABASE_URL |
| `cyberops-connect-persister` | `signalops-connect` | CronJob / Job worker | stateless | cluster-internal | cyberops | postgres, redpanda, topic-bootstrap | SIGNALOPS_DATABASE_URL |
| `cyberops-daily-feature-materializer` | `signalops-cyberops` | CronJob / Job worker | stateless | cluster-internal | retention-governance | postgres | SIGNALOPS_DATABASE_URL |
| `cyberops-detector` | `signalops-cyberops` | CronJob / Job worker | stateless | cluster-internal | cyberops | redpanda, topic-bootstrap | SIGNALOPS_DATABASE_URL |
| `cyberops-hourly-feature-materializer` | `signalops-cyberops` | CronJob / Job worker | stateless | cluster-internal | cyberops-materialization | postgres | SIGNALOPS_DATABASE_URL |
| `cyberops-iot-anomaly` | `signalops-cyberops` | CronJob / Job worker | stateless | cluster-internal | cyberops | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `cyberops-normalizer` | `signalops-cyberops` | CronJob / Job worker | stateless | cluster-internal | cyberops | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `gateway` | `signalops-app` | Deployment | mounts config/artifacts | ingress-facing; 18000->8080 | default | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY, SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL, STRIPE_API_KEY, STRIPE_RESTRICTED_API_KEY, STRIPE_WEBHOOK_SECRET, SYNCRATIC_API_BASE_URL, SYNCRATIC_CLIENT_SECRET, SYNCRATIC_PASSWORD, SYNCRATIC_TOKEN_AUDIENCE, SYNCRATIC_TOKEN_GRANT, SYNCRATIC_TOKEN_URL |
| `marketops-algorithm-adjudicator` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-asset-backfill-worker` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-backfill | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-eeom-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_FMP_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-eroc-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-intelligence-cohort-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-intraday-monitor` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-intraday | postgres | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-options-coverage-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-options-feature-materializer` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-postgres` | `signalops-marketops` | StatefulSet / managed service candidate | persistent data volume | operator/internal; 15434->5432 | default | — | POSTGRES_PASSWORD |
| `marketops-postgres-migrate` | `signalops-marketops` | Job | mounts config/artifacts | cluster-internal | marketops-boundary | marketops-postgres | PGPASSWORD, SIGNALOPS_DATABASE_URL |
| `marketops-retention-governor` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-retention-governance | marketops-postgres | SIGNALOPS_DATABASE_URL |
| `marketops-signal-assurance-outbox` | `signalops-marketops` | Deployment / worker | stateless | cluster-internal | default | postgres, redpanda, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL |
| `marketops-signal-assurance-registrar` | `signalops-marketops` | Deployment / worker | stateless | cluster-internal | default | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-signal-assurance-worker` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-sri-holdings-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-sri-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-syncratic-intelligence-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-tactical-valuation-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `marketops-timescaledb` | `signalops-marketops` | StatefulSet / managed service candidate | persistent data volume | operator/internal; 15435->5432 | default | — | POSTGRES_PASSWORD |
| `marketops-timescaledb-migrate` | `signalops-marketops` | Job | mounts config/artifacts | cluster-internal | marketops-boundary | marketops-timescaledb | PGPASSWORD, SIGNALOPS_DATABASE_URL |
| `marketops-valuation-runner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | marketops-daily | postgres | SIGNALOPS_DATABASE_URL, SIGNALOPS_FMP_API_KEY, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `massive-puller` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | massive-pull | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `massive-scheduler` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | massive-schedule | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `normalizer` | `signalops-marketops` | Deployment / worker | stateless | cluster-internal | default | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `postgres` | `signalops-data` | StatefulSet / managed service candidate | persistent data volume | operator/internal; 15432->5432 | default | — | POSTGRES_PASSWORD |
| `postgres-migrate` | `signalops-data` | Job | mounts config/artifacts | cluster-internal | storage | postgres | SIGNALOPS_DATABASE_URL |
| `raw-worker` | `signalops-connect` | Deployment / worker | stateless | cluster-internal | default | redpanda, topic-bootstrap | — |
| `redpanda` | `signalops-data` | StatefulSet / managed service candidate | persistent data volume | operator/internal; 19092->19092, 18081->18081, 18082->18082, 19644->9644 | default | — | — |
| `redpanda-console` | `signalops-data` | Deployment | stateless | operator/internal; 18080->8080 | default | redpanda | — |
| `replay-worker` | `signalops-marketops` | Deployment / worker | stateless | cluster-internal | default | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `retention-governor` | `signalops-data` | CronJob / Job worker | stateless | cluster-internal | retention-governance | postgres | SIGNALOPS_DATABASE_URL |
| `retry-replayer` | `signalops-data` | CronJob / Job worker | stateless | cluster-internal | retry-replay | redpanda, topic-bootstrap | — |
| `signal-persister` | `signalops-marketops` | Deployment / worker | stateless | cluster-internal | default | postgres, redpanda, timescaledb, topic-bootstrap | SIGNALOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_DATABASE_URL, SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `storage-monitor` | `signalops-data` | CronJob / Job worker | mounts config/artifacts | cluster-internal | storage-monitoring | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `subscriber-global-annual-financial-refresh` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_FMP_API_KEY, SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL |
| `subscriber-global-annual-financial-task-worker` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_FMP_API_KEY, SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL |
| `subscriber-global-annual-valuation-materializer` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL |
| `subscriber-global-catalog-admission` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY |
| `subscriber-global-eod-history-materializer` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL, SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL |
| `subscriber-global-eod-shadow-planner` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_DATABASE_URL |
| `subscriber-global-intraday-shadow-capture` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL |
| `subscriber-global-marketops-evidence-materializer` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL |
| `subscriber-global-marketops-parity-manifest` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL |
| `subscriber-global-ranking-import` | `signalops-marketops` | CronJob / Job worker | mounts config/artifacts | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_DATABASE_URL |
| `subscriber-global-saf-benchmark-materializer` | `signalops-marketops` | CronJob / Job worker | stateless | cluster-internal | subscriber-global-evidence | — | SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL, SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL |
| `temporal-backfill` | `signalops-data` | Job | mounts config/artifacts | cluster-internal | storage | postgres, timescaledb | SIGNALOPS_DATABASE_URL, SIGNALOPS_MASSIVE_API_KEY, SIGNALOPS_TEMPORAL_DATABASE_URL |
| `timescaledb` | `signalops-data` | StatefulSet / managed service candidate | persistent data volume | operator/internal; 15433->5432 | default | — | POSTGRES_PASSWORD |
| `timescaledb-migrate` | `signalops-data` | Job | mounts config/artifacts | cluster-internal | storage | timescaledb | SIGNALOPS_DATABASE_URL |
| `topic-bootstrap` | `signalops-data` | Job | mounts config/artifacts | cluster-internal | default | redpanda | — |
| `web` | `signalops-app` | Deployment | stateless | ingress-facing; 15173->8080 | default | gateway | — |

## Immediate build sequence

1. Scaffold namespace manifests for `signalops-app`, `signalops-connect`, `signalops-marketops`, `signalops-cyberops`, `signalops-identity`, `signalops-data`, and `signalops-observability`.
2. Add service-account and secret-class manifests before workload manifests; access boundaries should exist before replicas scale.
3. Convert stateless app workloads first: `web`, `gateway`, and read-only admin surfaces, with readiness/liveness probes and DB pool caps.
4. Convert MarketOps scheduled jobs second as CronJobs with `Forbid` concurrency and the same completion tables used by Admin Operations Health.
5. Convert Signal-Connect ingestion separately with ingress rate limits, DLQ/backpressure policy, and no broad database/admin privileges.
6. Keep stateful services on the current production host or managed services until a dedicated data-plane migration and restore rehearsal gate is approved.

## Known gaps before K8s staging

- Identity and observability are target planes but are not fully represented in this Compose package.
- PodDisruptionBudget, resource requests/limits, HorizontalPodAutoscaler, Traefik route manifests, and concrete secret-provider manifests are not yet scaffolded. Default-deny plus first-pass allow NetworkPolicies and secret-class policy documentation are now present under `deploy/kubernetes/base`.
- The live host still needs the updated MarketOps systemd timer files installed/reloaded so scheduler status displays the same 16:20/16:30/17:05/17:20 ET cadence now present in source.
- Backup/restore evidence for the final K8s data-plane topology remains a separate gate.

## Network and secret scaffold — 2026-09-09

The base scaffold now includes first-pass allow NetworkPolicies for:

- Traefik ingress into `signalops-app` web/gateway pods;
- gateway egress to identity, data, DNS, and approved public HTTPS APIs;
- app-to-Keycloak ingress in `signalops-identity`;
- platform-plane ingress into `signalops-data` database/broker boundary pods;
- MarketOps egress to data/broker, DNS, and provider HTTPS;
- Signal-Connect and CyberOps egress to data/broker and DNS;
- observability scrape egress to platform namespaces.

OpenBao is selected as the production secret backend. The scaffold includes one OpenBao secret-reader service account per SignalOps plane and a planning Agent Injector policy for app, connect, marketops, cyberops, identity, data, and observability secret classes. No runtime secrets are committed. Read-only cluster evidence confirmed the `openbao` namespace and `openbao-agent-injector-svc`; External Secrets Operator CRDs are not installed, so the next gate is OpenBao KV/auth policy validation before adding workload-specific injector annotations.
