# K8S worker node addition runbook — 2026-09-13

Status: prepared. No node was joined by this documentation update. Subsequent live validation showed CPU headroom currently green because active workload pressure dropped, but the cluster is still single-node and worker capacity remains recommended before production authority transfer.

## Purpose

The K3s capacity gate requires CPU request headroom and production resilience. The cluster has one ready node:

```text
hypernet101  192.168.2.5  v1.33.5+k3s1  allocatable_cpu=16  allocatable_memory=130982772Ki
```

Original capacity pressure at remediation planning time:

```text
allocatable_cpu_m=16000
requested_cpu_m=17665
cpu_request_pct=110.41
```

Latest live validation after active workload pressure changed:

```text
nodes=1
active_pods=78
allocatable_cpu_m=16000
requested_cpu_m=10765
cpu_request_pct=67.28
memory_request_pct=15.30
```

The headroom verifier is currently `status=ok`, but this is still a single-node posture. One additional worker remains the recommended path before production authority transfer because SignalOps, Signal-Connect, MarketOps jobs, Syncratic-core, Keycloak, OpenBao, Longhorn, and Istio are intended to share the same platform. At the original pressure level, the cluster needed at least 20,782m allocatable CPU; a practical minimum remains one additional 8 vCPU worker node.

## Recommended worker profile

Minimum practical worker:

- 8 vCPU
- 32 GB RAM
- Ubuntu 24.04 LTS or platform-standard Linux image
- static or reserved LAN IP
- same low-latency network as `hypernet101`
- persistent disk appropriate for container images and kubelet/containerd state
- time sync enabled

Preferred production worker:

- 16 vCPU
- 64 GB RAM
- dedicated SSD/NVMe
- future Longhorn disk path prepared if this node will host storage replicas

## Network prerequisites

Before joining the worker:

1. Confirm the worker can resolve/reach `hypernet101`.
2. Confirm the worker can reach the K3s API on `192.168.2.5:6443`.
3. Confirm firewall rules allow K3s/Cilium/Longhorn traffic between nodes.
4. Confirm outbound access to GHCR for private image pulls, unless images are mirrored locally.

## Join token handling

The K3s node token is sensitive. Do not commit it to source and do not paste it into chat logs.

On `hypernet101`, retrieve it locally:

```bash
sudo cat /var/lib/rancher/k3s/server/node-token
```

Use it only on the new worker node during installation.

## Worker join command

Run on the new worker node, replacing `<K3S_NODE_TOKEN>` and `<WORKER_NODE_NAME>`:

```bash
curl -sfL https://get.k3s.io | \
  K3S_URL=https://192.168.2.5:6443 \
  K3S_TOKEN=<K3S_NODE_TOKEN> \
  INSTALL_K3S_EXEC='agent --node-name <WORKER_NODE_NAME>' \
  sh -
```

If the platform uses a private registry mirror or custom CNI/bootstrap flags, apply those host-level settings before running the join command. The current SignalOps evidence expects Cilium and Istio to remain the platform networking/mesh layers.

## Post-join validation

Run from the SignalOps workspace on `hypernet101`:

```bash
kubectl get nodes -o wide
kubectl get pods -A -o wide
scripts/report_k8s_capacity_requests.sh
scripts/verify_k8s_capacity_headroom.sh
scripts/verify_k8s_production_cutover_readiness.sh
```

Expected change:

- node count increases from `1` to `2`;
- new worker is `Ready`;
- CPU request percentage falls below `85`;
- memory request percentage remains below `80`;
- `scripts/verify_k8s_capacity_headroom.sh` returns `status=ok`;
- `scripts/verify_k8s_production_cutover_readiness.sh` still keeps `production_cutover_allowed=false` until explicit authority transfer approval.

## Optional labeling strategy

If the new node is intended for SignalOps workloads, label it after it joins:

```bash
kubectl label node <WORKER_NODE_NAME> syncratic.io/workload-plane=signalops
kubectl label node <WORKER_NODE_NAME> syncratic.io/capacity-class=production
```

Do not add hard node selectors to SignalOps workloads until the scheduling design is explicitly approved. Labels can be added safely as future placement metadata without moving workloads by themselves.

## Longhorn/storage caution

If Longhorn schedules storage workloads onto the new worker, verify disk/path design first. Adding the node for compute capacity does not automatically mean it should host persistent replicas.

Storage enrollment should be treated as a separate approval if the worker has not been prepared for Longhorn data paths.

## Rollback

If the worker joins incorrectly and no persistent workload/data has been intentionally scheduled to it, cordon and drain before removal:

```bash
kubectl cordon <WORKER_NODE_NAME>
kubectl drain <WORKER_NODE_NAME> --ignore-daemonsets --delete-emptydir-data
kubectl delete node <WORKER_NODE_NAME>
```

Then uninstall K3s on the worker:

```bash
sudo /usr/local/bin/k3s-agent-uninstall.sh
```

Do not drain/remove a node that has Longhorn replicas or stateful workloads without storage-specific validation.

## Non-authorizations

This runbook does not authorize production traffic cutover, DNS movement, Kubernetes scheduler authority transfer, provider polling, workload deletion, or Longhorn storage enrollment.
