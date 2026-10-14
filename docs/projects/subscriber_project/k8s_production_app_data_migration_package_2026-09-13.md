# K8S production app/data migration package — 2026-09-13

Status: prepared and server-side dry-run verified. Production traffic has not moved.

## Decision update

The production migration target is Kubernetes-native data access, not a default dependency on the Docker-hosted databases. The earlier Docker-host fallback remains useful for emergency rollback or a deliberately staged app-only migration, but it is no longer the generated default for the production OpenBao runtime handoff.

The preferred order is:

1. Stand up production K8s database services in `signalops-data` with retained Longhorn PVCs.
2. Restore or replicate the current production database state into those K8s database services.
3. Provision `signalops/data/k8s/app/signalops-gateway-runtime-production` in OpenBao with Kubernetes `.svc.cluster.local` database URLs.
4. Apply the production Gateway/Web overlay with replicas behind the Istio route.
5. Run authenticated and webhook route validation through the K8s route.
6. Move public traffic authority only under a named production traffic cutover approval.
7. Transfer MarketOps scheduler authority separately after app traffic is stable.

## Production data package

The new production data overlay is `deploy/kubernetes/production/data`.

It defines internal-only ClusterIP services and StatefulSets for:

- `signalops-postgres-production`
- `signalops-timescaledb-production`
- `marketops-postgres-production`
- `marketops-timescaledb-production`

The PVCs use `syncratic-data-retain`, not `local-path`, so accidental manifest deletion should not delete the underlying retained volumes. The database services are not exposed by Ingress or LoadBalancer. Bootstrap database passwords are injected from OpenBao path `signalops/data/k8s/data/signalops-databases-runtime-production` under the `signalops-data` Kubernetes auth role; the production data overlay no longer references Kubernetes Secret password objects.

Validation evidence:

```text
signalops_k8s_production_data_package_verified
production_data_overlay=verified
namespace=signalops-data
storage_class=syncratic-data-retain
secret_source=openbao
openbao_path=signalops/data/k8s/data/signalops-databases-runtime-production
services=signalops-postgres-production,signalops-timescaledb-production,marketops-postgres-production,marketops-timescaledb-production
server_side_dry_run=passed
production_traffic_moved=false
production_cutover_allowed=false
```

## Production app package

The production app overlay is `deploy/kubernetes/production/app`, and the production route overlay is `deploy/kubernetes/production/mesh-route`.

The Gateway runtime uses OpenBao path `signalops/data/k8s/app/signalops-gateway-runtime-production`. The generated runtime env now rewrites Compose-local database names to Kubernetes service DNS names such as `marketops-postgres-production.signalops-data.svc.cluster.local:5432`. It does not generate Docker host-port URLs by default.

Validation evidence:

```text
signalops_k8s_production_app_cutover_package_verified
production_app_overlay=verified
production_mesh_route_overlay=verified
openbao_runtime_path=signalops/data/k8s/app/signalops-gateway-runtime-production
images=ghcr.io/syncratic-inc/signalops-gateway:production,ghcr.io/syncratic-inc/signalops-web:production
host=signalops.syncratic.io
server_side_dry_run=passed
production_traffic_moved=false
production_cutover_allowed=false
```

## Current readiness report

The production-readiness verifier now reports the K8s production app/data packages as verified while keeping cutover blocked on the remaining gates:

```text
k8s_production_data_package=verified
k8s_production_app_cutover_package=verified
pending_production_app_runtime_openbao=false
pending_production_data_runtime_openbao=false
pending_production_database_restore_or_replication=true
pending_production_traffic_authority_transfer=true
pending_capacity_load_validation=true
```

## Explicit non-authorizations

This package does not authorize:

- DNS movement for `signalops.syncratic.io`;
- production traffic movement from Docker/Traefik to Istio;
- database restore into K8s production PVCs;
- Kubernetes scheduler authority transfer;
- provider polling from K8s production jobs.

Those remain named-approval gates.

## September 14, 2026 execution note

The approved production runtime provisioning gate reached an environmental blocker: OpenBao was sealed (`Sealed=true`, unseal progress `0/3`). Runtime env rendering succeeded, but production app/data runtime secrets were not written and the production data overlay was not applied. Applying the overlay while OpenBao is sealed would strand the database pods without injected bootstrap passwords, so the correct state is to unseal OpenBao first, then rerun the provisioning gate.

A shell compatibility issue was also fixed: generated OpenBao pod-side payloads now use POSIX `set -eu` rather than `set -euo pipefail`, because the OpenBao pod executes payloads with `/bin/sh`.

## September 14, 2026 OpenBao unsealed production data readiness

After OpenBao was unsealed, the approved production runtime provisioning gate completed without exposing secret values.

Verified evidence:

```text
openbao_signalops_data_runtime_production_verified
openbao_signalops_app_runtime_production_verified
cross_plane_denied=true
secret_values=production_runtime_supplied
production_traffic_moved=false
```

The production data overlay was then applied into `signalops-data` using retained Longhorn PVCs and OpenBao Agent Injector runtime delivery. Initial database startup exposed the standard Longhorn/ext filesystem `lost+found` issue when mounting a PVC directly as PostgreSQL's data directory. The manifest now sets `PGDATA=/var/lib/postgresql/data/pgdata` for all four production database pods so PostgreSQL initializes inside a subdirectory while preserving the PVC boundary.

Current K8s production data evidence:

```text
signalops-postgres-production-0: 2/2 Running, pg_isready accepting connections
signalops-timescaledb-production-0: 2/2 Running, pg_isready accepting connections
marketops-postgres-production-0: 2/2 Running, pg_isready accepting connections
marketops-timescaledb-production-0: 2/2 Running, pg_isready accepting connections
services: ClusterIP only
production_traffic_moved=false
provider_polling_invoked=false
k8s_schedulers_enabled=false
```

The remaining production cutover gates are database restore/replication into the K8s production PVCs, final capacity/load validation, and explicit traffic authority transfer.

## September 14, 2026 database replication preparation

A non-destructive replication readiness tool is now source-controlled:

- `scripts/start_signalops_docker_database_sources_for_k8s_replication.sh` starts only the Docker database source containers needed for a K8s copy: shared SignalOps primary/temporal and dedicated MarketOps primary/temporal. It does not restart web/gateway, start schedulers, or invoke providers.
- `scripts/replicate_signalops_docker_databases_to_k8s_production.sh` is currently dry-run only. It verifies Docker source availability, K8s target pod readiness, ClusterIP-only database services, and table/size evidence without copying data.
- Deployment-agent actions were added only for `k8s-production-db-sources-start` and `k8s-production-db-replication-dry-run`. No persistent destructive execute action exists.

Current dry-run evidence showed the K8s targets are ready and empty, while Docker source availability is incomplete until the missing Docker DB source containers are started:

```text
signalops: source ready, target ready, source_tables=159, target_tables=0
signalops_temporal: source missing_or_stopped, target ready
marketops: source missing_or_stopped, target ready
marketops_temporal: source missing_or_stopped, target ready
ready_for_execute=false
```

Actual MarketOps database copy remains a separate named-approval gate because it will replace the initialized contents inside the K8s production MarketOps database PVCs, even though public traffic has not moved there.

Required approval text for the one-time copy gate:

```text
I, luke@strategiclabs.io, approve copying current Docker production SignalOps and MarketOps databases into the K8s production database PVCs, replacing only the K8s production target database contents, with no DNS cutover, no public traffic movement, no K8s scheduler enablement, and no provider polling.
```

## September 14, 2026 MarketOps-only replication correction

During the first approved copy attempt, the original replication script followed the broader platform phrase "SignalOps and MarketOps databases" and started with the shared SignalOps primary database. That was stopped before any MarketOps target copy began. No public traffic points to the K8s production data services. The K8s shared SignalOps primary target is therefore treated as an incomplete, non-serving partial restore and is not part of the MarketOps cutover evidence.

The replication tool has been corrected so the default scope is now `marketops-only`. Shared SignalOps platform database migration is a separate platform gate and should not be coupled to MarketOps database cutover readiness.

Latest MarketOps-only dry-run evidence:

```text
scope=marketops-only
marketops: source ready, target ready, source_tables=190, target_tables=0
marketops_temporal: source ready, target ready, source_tables=6, target_tables=0
ready_for_execute=true
production_traffic_moved=false
k8s_schedulers_enabled=false
provider_polling=false
```

Required approval text for the corrected MarketOps-only copy gate:

```text
I, luke@strategiclabs.io, approve copying current Docker production MarketOps databases into the K8s production MarketOps database PVCs, replacing only the K8s production MarketOps target database contents, with no SignalOps shared database copy, no DNS cutover, no public traffic movement, no K8s scheduler enablement, and no provider polling.
```
