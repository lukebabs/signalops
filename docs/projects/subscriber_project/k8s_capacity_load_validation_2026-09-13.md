# K8S capacity/load validation — 2026-09-13

Status: partial. The bounded K3s staging route/runtime load smoke passed, but production-grade capacity remains blocked by cluster CPU reservation headroom.

## Purpose

This gate validates whether the K3s SignalOps runtime can carry basic gateway traffic and webhook-backed subscription writes before production authority moves from Docker Compose/systemd to Kubernetes.

## What was tested

`scripts/run_k8s_capacity_load_validation_smoke.sh` was added as a non-cutover staging smoke. It:

1. verifies Istio, staging app manifests, staging route manifests, and ingress/DNS rollback planning;
2. provisions OpenBao app runtime without printing secrets;
3. bootstraps the staging subscription/enrollment schema;
4. seeds synthetic checkout references in the staging MarketOps database;
5. scales only `signalops-gateway` to one and keeps `signalops-web` at zero;
6. exercises the Istio staging route with `curl --resolve` against `signalops-staging.syncratic.co`;
7. measures `/healthz`, `/readyz`, and signed `checkout.session.completed` webhook reconciliation;
8. scales staging app workloads back to zero.

## Passing route/runtime smoke evidence

The bounded smoke passed:

```text
signalops_k8s_capacity_load_validation_verified
namespace=signalops-app
staging_hostname=signalops-staging.syncratic.co
gateway=istio-system/public-ingress
gateway_ip=192.168.2.233
run_id=k8s-capacity-20260913T174000Z
healthz_requests=120
healthz_concurrency=12
healthz_error_rate_pct=0.0
healthz_p95_ms=12.52
healthz_p99_ms=13.05
healthz_rps=1186.22
readyz_requests=80
readyz_concurrency=8
readyz_error_rate_pct=0.0
readyz_p95_ms=10.39
readyz_p99_ms=10.71
readyz_rps=931.77
stripe_webhook_checkout_requests=24
stripe_webhook_checkout_concurrency=4
stripe_webhook_checkout_error_rate_pct=0.0
stripe_webhook_checkout_p95_ms=22.42
stripe_webhook_checkout_p99_ms=22.69
stripe_webhook_checkout_rps=313.04
stripe_provider_called=false
production_dns_changed=false
production_traffic_moved=false
production_cutover_allowed=false
staging_self_signed_tls_allowed=true
staging_zero_cpu_reservation_used=true
scaled_back_to_zero=true
```

## Capacity headroom blocker

`scripts/verify_k8s_capacity_headroom.sh` was added to make cluster headroom evidence explicit. Current result:

```text
signalops_k8s_capacity_headroom_report
status=blocked
nodes=1
active_pods=101
allocatable_cpu_m=16000.0
requested_cpu_m=17665.0
cpu_request_pct=110.41
max_cpu_request_pct=85
allocatable_memory_mib=127912.86
requested_memory_mib=33904.0
memory_request_pct=26.51
max_memory_request_pct=80
production_cutover_allowed=false
```

The smoke had to use `staging_zero_cpu_reservation_used=true` because the single K3s node was already above the production-readiness CPU request threshold. That makes the route/runtime result useful, but it does not prove production-grade capacity.

## Production implication

This gate does not block because the SignalOps gateway is slow; it blocks because the cluster has insufficient schedulable CPU reservation headroom for a clean production cutover. Before production authority transfer, one of the following must happen:

- add worker node capacity and rerun the headroom verifier until `status=ok`;
- reduce or right-size CPU requests for existing Syncratic/SignalOps workloads;
- isolate SignalOps production workloads onto dedicated node capacity with explicit requests/limits;
- rerun a production-shaped load test with non-zero CPU requests and realistic concurrent authenticated routes.

## Non-authorizations

This gate did not move production DNS, did not move production traffic, did not call Stripe, did not enable Kubernetes schedules, and did not transfer production authority.
