# K8S-5 raw-worker scaffold — 2026-09-13

Status: source-ready, manifest-verified, private GHCR image-published, pull-smoked, and applied dormant in-cluster. No production traffic moved and the worker has not processed staging messages yet.

## Scope

This gate covers the Python `raw-worker` from Docker Compose. It is the normalized-event algorithm worker and is intentionally separate from the Go Signal-Connect persister/outbox workers closed in K8S-4.

The staged Kubernetes workload is:

- namespace: `signalops-connect`;
- deployment: `signalops-raw-worker`;
- image: `ghcr.io/syncratic-inc/signalops-python-worker:staging`;
- broker: `signalops-redpanda-staging.signalops-data.svc.cluster.local:9092`;
- input topic: `signalops.kubernetes-staging.normalized.v1`;
- retry topic: `signalops.kubernetes-staging.retry.algorithm.v1`;
- DLQ topic: `signalops.kubernetes-staging.dlq.algorithm.v1`;
- signal topic: `signalops.kubernetes-staging.signal.v1`;
- max messages: `1`;
- replicas: `0`;
- production guard: `production-cutover-allowed=false`.

## Manifest evidence

```text
signalops_k8s_raw_worker_manifests_verified
namespace=signalops-connect
deployment=signalops-raw-worker
image=ghcr.io/syncratic-inc/signalops-python-worker:staging
replicas=0
broker=signalops-redpanda-staging.signalops-data.svc.cluster.local:9092
external_exposure=false
provider_polling=false
production_cutover_allowed=false
applied=false
```

## Image publication evidence

```text
signalops_k8s_python_worker_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-python-worker
source_tag=fe1b1d7c01f1
staging_tag=staging
permission=write
```

## Pull smoke evidence

```text
signalops_k8s_raw_worker_pull_smoke_verified
namespace=signalops-connect
registry=ghcr.io
image=ghcr.io/syncratic-inc/signalops-python-worker:staging
secret=signalops-ghcr-pull
pod_succeeded=true
worker_executed=false
provider_polling=false
production_cutover_allowed=false
```

## Dormant apply evidence

```text
deployment.apps/signalops-raw-worker created
signalops-raw-worker 0/0 ghcr.io/syncratic-inc/signalops-python-worker:staging
```

## Remaining gate

The next raw-worker gate is a bounded one-message processing smoke. It should seed one deterministic normalized event into the staging broker, scale `signalops-raw-worker` to one, verify exactly one worker run/output path, and restore replicas to zero. This requires a generated fixture that satisfies the normalized-event schema and detector assumptions; it should be handled as a separate controlled migration step.
