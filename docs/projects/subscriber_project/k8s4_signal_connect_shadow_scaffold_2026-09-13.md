# K8S-4 Signal-Connect shadow scaffold — 2026-09-13

Status: source-ready and manifest-verified. No production traffic has moved and no Signal-Connect consumer has been scaled above zero.

## Outcome

This slice starts moving Signal-Connect into K3s by creating the staging workload shape for the concrete Connect workers that exist today:

- `signalops-connect-persister` — wraps the existing `signalops-cyberops-connect-persister` binary.
- `signalops-connect-outbox` — wraps the existing `signalops-cyberops-connect-outbox` binary.

Both are staged in namespace `signalops-connect` as Deployments with `replicas: 0`. They use a shared K3s worker image target and a bounded entrypoint that accepts only the two known worker IDs.

## Controls

The scaffold intentionally keeps this as a shadow-readiness step rather than a production cutover:

- `replicas: 0` for both Deployments.
- No Service and no Ingress are created.
- No Kubernetes Secret is committed.
- Runtime configuration is projected from OpenBao path `signalops/data/k8s/connect/connect-worker-runtime-staging`.
- OpenBao role is `signalops-connect`; app, MarketOps, and data-plane roles are not reused.
- OpenBao CA trust is explicit through `signalops-openbao-ca` and `/vault/tls/ca.crt`.
- `SIGNALOPS_CONNECT_SHADOW_MODE=true` is set.
- `production-cutover-allowed=false` is present on the workload shape.

## Validation

Manifest validation passed:

```text
signalops_k8s_connect_staging_manifests_verified
deployments=2
networkpolicies=1
secrets=0
services=0
cronjobs=0
statefulsets=0
replicas=0
openbao_secret_path=signalops/data/k8s/connect/connect-worker-runtime-staging
shadow_mode=true
provider_polling=false
production_cutover_allowed=false
applied=false
```

The broader non-cutover readiness report now includes:

```text
k8s_signal_connect_manifest_guard=verified
pending_signal_connect_ingestion_shadow=true
production_cutover_allowed=false
```

`pending_signal_connect_ingestion_shadow` remains true because the workloads have not yet been published, applied, scaled to one, or validated against staging broker/database flow.

## Deliberate boundary

The Compose `raw-worker` is not claimed as closed by this slice. It is a Python normalized-event algorithm worker, not one of the two Go Signal-Connect persistence/outbox workers. It should receive its own K3s worker-image and shadow-flow validation gate so ingestion, normalization, retry, and DLQ behavior can be tested without conflating responsibilities.

## Next gate

1. Publish `ghcr.io/syncratic-inc/signalops-connect-k8s-worker` from this committed source.
2. Verify the private GHCR pull in `signalops-connect`.
3. Provision placeholder-only or non-production OpenBao Connect runtime values.
4. Apply the replicas-zero scaffold in-cluster.
5. Run one bounded scale-to-one shadow smoke against staging broker/database, then scale back to zero.
