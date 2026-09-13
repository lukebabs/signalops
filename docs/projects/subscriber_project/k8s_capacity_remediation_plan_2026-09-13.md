# K8S capacity remediation plan — 2026-09-13

Status: active production-readiness blocker. No workloads were changed by this gate.

## Purpose

The K3s route/runtime load smoke proved that the staged SignalOps gateway can answer bounded health, readiness, and synthetic webhook traffic through the Istio staging route. That does not by itself prove production cutover readiness because the cluster is already over its CPU request budget.

This plan makes the capacity pressure attributable by namespace and workload so the remediation can be deliberate, reversible, and separated from production authority transfer.

## Current capacity report

Command:

```bash
scripts/report_k8s_capacity_requests.sh
```

Current result:

```text
signalops_k8s_capacity_requests_report
nodes=1
active_pods=101
allocatable_cpu_m=16000.00
requested_cpu_m=17665.00
cpu_request_pct=110.41
allocatable_memory_mib=127912.86
requested_memory_mib=33904.00
memory_request_pct=26.51
production_cutover_allowed=false
```

The immediate blocker is CPU reservation headroom, not memory. Memory requests are currently 26.51% of allocatable memory, while CPU requests are 110.41% of allocatable CPU. The production-readiness threshold remains 85% CPU request and 80% memory request.

## Top namespace pressure

```text
namespace=syncratic active_pods=33 cpu_m=7650.00 cpu_pct_of_cluster=47.81 memory_mib=16000.00
namespace=syncratic-runtime-smoke active_pods=14 cpu_m=3550.00 cpu_pct_of_cluster=22.19 memory_mib=7424.00
namespace=syncratic-capacity-rehearsal active_pods=5 cpu_m=2750.00 cpu_pct_of_cluster=17.19 memory_mib=5376.00
namespace=longhorn-system active_pods=19 cpu_m=1920.00 cpu_pct_of_cluster=12.00 memory_mib=0.00
namespace=istio-system active_pods=4 cpu_m=950.00 cpu_pct_of_cluster=5.94 memory_mib=3456.00
namespace=kube-system active_pods=6 cpu_m=300.00 cpu_pct_of_cluster=1.88 memory_mib=150.00
namespace=signalops-data active_pods=3 cpu_m=200.00 cpu_pct_of_cluster=1.25 memory_mib=768.00
namespace=syncratic-data-plane active_pods=2 cpu_m=200.00 cpu_pct_of_cluster=1.25 memory_mib=384.00
```

SignalOps staging app and MarketOps job workloads are intentionally scaled to zero outside bounded smokes. The present CPU pressure is dominated by Syncratic platform/runtime workloads, smoke/rehearsal namespaces, Longhorn, and Istio.

## Top workload pressure

```text
workload=longhorn-system/InstanceManager/instance-manager-0301e50a87a56c094d03e9a9c3aef3d5 active_pods=1 cpu_m=1920.00 cpu_pct_of_cluster=12.00 memory_mib=0.00
workload=syncratic-capacity-rehearsal/Deployment/syncratic-capacity-bge-embeddings active_pods=1 cpu_m=1000.00 cpu_pct_of_cluster=6.25 memory_mib=2048.00
workload=syncratic/Deployment/syncratic-syncratic-gateway active_pods=2 cpu_m=1000.00 cpu_pct_of_cluster=6.25 memory_mib=2048.00
workload=istio-system/Deployment/istiod active_pods=1 cpu_m=500.00 cpu_pct_of_cluster=3.12 memory_mib=2048.00
workload=syncratic-capacity-rehearsal/StatefulSet/syncratic-capacity-neo4j active_pods=1 cpu_m=500.00 cpu_pct_of_cluster=3.12 memory_mib=1024.00
workload=syncratic-capacity-rehearsal/StatefulSet/syncratic-capacity-postgres active_pods=1 cpu_m=500.00 cpu_pct_of_cluster=3.12 memory_mib=1024.00
workload=syncratic-capacity-rehearsal/StatefulSet/syncratic-capacity-qdrant active_pods=1 cpu_m=500.00 cpu_pct_of_cluster=3.12 memory_mib=1024.00
workload=syncratic-runtime-smoke/Deployment/syncratic-refactor-smoke-syncratic-phase1-gateway active_pods=1 cpu_m=500.00 cpu_pct_of_cluster=3.12 memory_mib=1024.00
```

## Remediation options

### Option A — add worker capacity

Preferred for production. Add at least one additional worker node, then rerun:

```bash
scripts/verify_k8s_capacity_headroom.sh
scripts/report_k8s_capacity_requests.sh
```

This preserves current workload posture and avoids changing unrelated Syncratic runtime services while SignalOps migration continues.

### Option B — clean up stale smoke/rehearsal namespaces

`syncratic-runtime-smoke` and `syncratic-capacity-rehearsal` together reserve 6,300m CPU, or 39.38% of the current cluster CPU. If they are obsolete, scaling/removing them would likely bring the cluster below the 85% CPU threshold.

This requires explicit owner approval because these namespaces may belong to active Syncratic-core migration or capacity-rehearsal work. SignalOps should not delete or scale them as part of its own production migration without that approval.

### Option C — right-size requests

Review CPU requests for Syncratic platform workloads, Longhorn, Istio, and future SignalOps workloads against observed usage. This should be paired with metrics, not guessed values. Right-sizing can improve density, but it is not a substitute for production capacity if traffic will grow.

### Option D — dedicated SignalOps node capacity

Create a dedicated SignalOps node pool or labeled worker capacity and bind SignalOps app, MarketOps jobs, Signal-Connect, and data-plane workloads through node selectors/affinity and tolerations. This makes production ownership clearer and avoids cross-product contention.

## Recommended path

1. Keep Docker Compose/systemd as live production authority.
2. Keep SignalOps K3s staging app/workers at zero replicas outside bounded smokes.
3. Add worker capacity or explicitly approve cleanup of stale smoke/rehearsal namespaces.
4. Rerun `scripts/verify_k8s_capacity_headroom.sh`; require `status=ok`.
5. Rerun the route/runtime load smoke with non-zero production-shaped CPU requests.
6. Only then prepare the separate production authority transfer approval.

## Non-authorizations

This gate did not scale, delete, or restart any workload. It did not move production DNS, move production traffic, enable Kubernetes schedules, call providers, or transfer production authority.
