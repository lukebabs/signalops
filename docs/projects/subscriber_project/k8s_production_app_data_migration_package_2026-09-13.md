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
pending_production_app_runtime_openbao=true
pending_production_data_runtime_openbao=true
pending_production_database_restore_or_replication=true
pending_production_traffic_authority_transfer=true
pending_capacity_load_validation=true
```

## Explicit non-authorizations

This package does not authorize:

- DNS movement for `signalops.syncratic.io`;
- production traffic movement from Docker/Traefik to Istio;
- production app runtime secret migration into OpenBao;
- production data bootstrap secret migration into OpenBao;
- database restore into K8s production PVCs;
- Kubernetes scheduler authority transfer;
- provider polling from K8s production jobs.

Those remain named-approval gates.

## September 14, 2026 execution note

The approved production runtime provisioning gate reached an environmental blocker: OpenBao was sealed (`Sealed=true`, unseal progress `0/3`). Runtime env rendering succeeded, but production app/data runtime secrets were not written and the production data overlay was not applied. Applying the overlay while OpenBao is sealed would strand the database pods without injected bootstrap passwords, so the correct state is to unseal OpenBao first, then rerun the provisioning gate.

A shell compatibility issue was also fixed: generated OpenBao pod-side payloads now use POSIX `set -eu` rather than `set -euo pipefail`, because the OpenBao pod executes payloads with `/bin/sh`.
