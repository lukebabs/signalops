# K8S-5 raw-worker processing shadow — 2026-09-13

Status: closed for bounded staging processing. No production traffic moved.

## Scope

This gate validated the Python `raw-worker` as the normalized-event algorithm worker in K3s staging. It is separate from the Go Signal-Connect persister/outbox shadow gate.

The smoke used the internal staging Redpanda broker and standard `kubernetes-staging` topics. It seeded one deterministic MarketOps normalized EOD event, scaled `signalops-raw-worker` from zero to one replica, verified that the worker emitted the expected DSM signal, then returned the Deployment to zero replicas.

## Root-cause correction before closure

The first processing smoke failed because the synthetic fixture used `source_adapter=market_data.k8s-smoke`. The DSM taxonomy detector intentionally accepts only the governed MarketOps adapter scope, so the worker consumed and processed the event but emitted no signal. The fixture was corrected to use the governed normalized adapter value `market_data.massive` while retaining synthetic-smoke provenance in source id, metadata, and evidence.

The staging raw-worker Deployment was also hardened with `imagePullPolicy: Always` because the mutable `:staging` tag should not be reused from a node cache during migration canaries.

## Processing smoke evidence

```text
signalops_k8s_raw_worker_processing_smoke_verified
namespace=signalops-connect
deployment=signalops-raw-worker
run_id=raw-worker-smoke-20260913T163832Z
input_topic=signalops.kubernetes-staging.normalized.v1
signal_topic=signalops.kubernetes-staging.signal.v1
signal_type=marketops.dsm.accumulation
messages_processed=1
replicas_restored_to_zero=true
provider_polling=false
production_cutover_allowed=false
```

## Safety boundaries

- No provider polling was performed.
- No production DNS, ingress, or scheduler authority changed.
- The worker remains dormant at `0/0` after the smoke.
- The synthetic event is deterministic and scoped to the staging broker.
- Production cutover remains disabled until ingress/DNS rollback planning, Stripe webhook parity, capacity/load validation, and explicit authority transfer close.
