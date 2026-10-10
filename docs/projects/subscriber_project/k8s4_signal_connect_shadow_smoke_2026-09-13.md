# K8S-4 Signal-Connect shadow smoke — 2026-09-13

Status: closed for bounded non-production K3s shadow. Production traffic remains on the existing Docker Compose/systemd authority.

## What was proven

Signal-Connect now has a complete staging path for the two Go workers currently in scope:

- `signalops-connect-persister`
- `signalops-connect-outbox`

The workers were not connected to production traffic. They used:

- namespace: `signalops-connect`;
- image: `ghcr.io/syncratic-inc/signalops-connect-k8s-worker:staging`;
- runtime: OpenBao `signalops/k8s/connect/connect-worker-runtime-staging`;
- database: non-production runtime-smoke PostgreSQL `.svc` endpoint;
- broker: internal staging Redpanda at `signalops-redpanda-staging.signalops-data.svc.cluster.local:9092`;
- mode: `SIGNALOPS_CONNECT_SHADOW_MODE=true`.

## Broker scaffold and topic evidence

A staging-only Redpanda broker was added under `signalops-data` with no external exposure, `emptyDir` storage, and `production-cutover-allowed=false`.

```text
signalops_k8s_connect_broker_manifests_verified
namespace=signalops-data
service=signalops-redpanda-staging
deployment=signalops-redpanda-staging
networkpolicy=allow-redpanda-staging-self-and-dns
broker=redpanda
port=9092
storage=emptyDir
external_exposure=false
provider_polling=false
production_cutover_allowed=false
applied=false
```

Live staging broker smoke created the 12 standard `kubernetes-staging` topics:

```text
signalops_k8s_connect_broker_smoke_verified
namespace=signalops-data
deployment=signalops-redpanda-staging
environment=kubernetes-staging
topics=12
partitions=1
replicas=1
external_exposure=false
provider_polling=false
production_cutover_allowed=false
```

## Runtime and OpenBao evidence

The Connect runtime was generated from non-production Kubernetes service endpoints. The staging database was prepared with the CyberOps Connect ingress tables from migration `000058`. Secret values were not printed.

```text
signalops_k8s_connect_runtime_staging_env_created
path=/tmp/signalops-openbao-connect-runtime-staging.env
mode=0600
database=runtime-smoke
broker=signalops-redpanda-staging.signalops-data.svc.cluster.local:9092
connect_tables=true
secret_values_printed=false
production_cutover_allowed=false
```

OpenBao accepted the non-production runtime and preserved cross-plane denial:

```text
openbao_signalops_connect_runtime_staging_verified
mount=signalops
connect_role=signalops-connect
connect_namespace=signalops-connect
connect_service_account=signalops-connect-worker
secret_path=signalops/k8s/connect/connect-worker-runtime-staging
deny_role=signalops-marketops
deny_namespace=signalops-marketops
cross_plane_denied=true
secret_values=non_production_runtime_supplied
production_cutover_allowed=false
```

## Shadow smoke result

The first attempt surfaced two valid staging blockers and both were fixed before the passing run:

1. `signalops-connect` was missing the `signalops-openbao-ca` trust secret. The existing OpenBao CA secret was mirrored into the Connect namespace without printing certificate contents.
2. The single-node K3s host could not schedule both Connect workers plus OpenBao sidecars concurrently. The staging smoke was changed to validate workers sequentially, and staging resource requests were reduced. This is acceptable for this migration gate; full concurrent capacity remains covered by the separate capacity/load validation gate.

Passing result:

```text
signalops_k8s_connect_shadow_smoke_verified
namespace=signalops-connect
persister=signalops-connect-persister
outbox=signalops-connect-outbox
replicas_scaled_to_one=sequential
replicas_restored_to_zero=true
runtime=openbao_non_production
broker=signalops-redpanda-staging.signalops-data.svc.cluster.local:9092
provider_polling=false
production_cutover_allowed=false
```

## Boundary

This closes Signal-Connect ingestion shadow for the two Go Connect workers, not full production cutover. The Python `raw-worker` remains a separate normalized-event worker migration gate. Production transfer remains blocked by ingress/DNS rollback planning, Stripe webhook parity through K3s route, capacity/load validation, and explicit production authority transfer.
