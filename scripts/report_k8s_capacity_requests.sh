#!/usr/bin/env bash
set -euo pipefail

LIMIT="${SIGNALOPS_K8S_CAPACITY_REPORT_LIMIT:-12}"

fail() {
  printf 'signalops_k8s_capacity_requests_failed: %s\n' "$*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

nodes_file="$workdir/nodes.json"
pods_file="$workdir/pods.json"

kubectl get nodes -o json >"$nodes_file"
kubectl get pods -A -o json >"$pods_file"

python3 - "$nodes_file" "$pods_file" "$LIMIT" <<'PYREQ'
import json
import re
import sys
from collections import defaultdict

nodes_path, pods_path, limit_arg = sys.argv[1], sys.argv[2], sys.argv[3]
limit = max(1, int(limit_arg))

with open(nodes_path, encoding="utf-8") as fh:
    nodes = json.load(fh)
with open(pods_path, encoding="utf-8") as fh:
    pods = json.load(fh)


def parse_cpu(value):
    if value is None or value == "":
        return 0.0
    value = str(value)
    if value.endswith("m"):
        return float(value[:-1])
    if value.endswith("n"):
        return float(value[:-1]) / 1_000_000.0
    if value.endswith("u"):
        return float(value[:-1]) / 1000.0
    return float(value) * 1000.0


def parse_memory(value):
    if value is None or value == "":
        return 0.0
    value = str(value)
    units = {
        "Ki": 1024,
        "Mi": 1024**2,
        "Gi": 1024**3,
        "Ti": 1024**4,
        "K": 1000,
        "M": 1000**2,
        "G": 1000**3,
        "T": 1000**4,
    }
    for suffix, multiplier in units.items():
        if value.endswith(suffix):
            return float(value[: -len(suffix)]) * multiplier
    return float(value)


def pod_effective_requests(pod):
    app_cpu = 0.0
    app_mem = 0.0
    for container in pod.get("spec", {}).get("containers", []):
        requests = container.get("resources", {}).get("requests", {})
        app_cpu += parse_cpu(requests.get("cpu"))
        app_mem += parse_memory(requests.get("memory"))

    init_cpu = 0.0
    init_mem = 0.0
    for container in pod.get("spec", {}).get("initContainers", []):
        requests = container.get("resources", {}).get("requests", {})
        init_cpu = max(init_cpu, parse_cpu(requests.get("cpu")))
        init_mem = max(init_mem, parse_memory(requests.get("memory")))

    return max(app_cpu, init_cpu), max(app_mem, init_mem)


def workload_for_pod(pod):
    namespace = pod.get("metadata", {}).get("namespace", "default")
    pod_name = pod.get("metadata", {}).get("name", "unknown-pod")
    owners = pod.get("metadata", {}).get("ownerReferences", [])
    if not owners:
        return namespace, "Pod", pod_name
    owner = owners[0]
    kind = owner.get("kind", "Owner")
    name = owner.get("name", pod_name)
    if kind == "ReplicaSet":
        deployment_name = re.sub(r"-[a-f0-9]{8,12}$", "", name)
        if deployment_name != name:
            return namespace, "Deployment", deployment_name
    return namespace, kind, name


alloc_cpu = 0.0
alloc_mem = 0.0
for node in nodes.get("items", []):
    alloc = node.get("status", {}).get("allocatable", {})
    alloc_cpu += parse_cpu(alloc.get("cpu"))
    alloc_mem += parse_memory(alloc.get("memory"))

namespace_totals = defaultdict(lambda: {"pods": 0, "cpu": 0.0, "mem": 0.0})
workload_totals = defaultdict(lambda: {"pods": 0, "cpu": 0.0, "mem": 0.0})
active_pods = 0
requested_cpu = 0.0
requested_mem = 0.0

for pod in pods.get("items", []):
    if pod.get("status", {}).get("phase") in {"Succeeded", "Failed"}:
        continue
    active_pods += 1
    cpu, mem = pod_effective_requests(pod)
    requested_cpu += cpu
    requested_mem += mem

    namespace = pod.get("metadata", {}).get("namespace", "default")
    namespace_totals[namespace]["pods"] += 1
    namespace_totals[namespace]["cpu"] += cpu
    namespace_totals[namespace]["mem"] += mem

    workload_key = workload_for_pod(pod)
    workload_totals[workload_key]["pods"] += 1
    workload_totals[workload_key]["cpu"] += cpu
    workload_totals[workload_key]["mem"] += mem

cpu_pct = (requested_cpu / alloc_cpu) * 100 if alloc_cpu else 0.0
mem_pct = (requested_mem / alloc_mem) * 100 if alloc_mem else 0.0

print("signalops_k8s_capacity_requests_report")
print(f"nodes={len(nodes.get('items', []))}")
print(f"active_pods={active_pods}")
print(f"allocatable_cpu_m={alloc_cpu:.2f}")
print(f"requested_cpu_m={requested_cpu:.2f}")
print(f"cpu_request_pct={cpu_pct:.2f}")
print(f"allocatable_memory_mib={alloc_mem / 1024 / 1024:.2f}")
print(f"requested_memory_mib={requested_mem / 1024 / 1024:.2f}")
print(f"memory_request_pct={mem_pct:.2f}")
print("top_namespace_cpu_requests_begin")
for namespace, values in sorted(namespace_totals.items(), key=lambda item: item[1]["cpu"], reverse=True)[:limit]:
    cpu_share = (values["cpu"] / alloc_cpu) * 100 if alloc_cpu else 0.0
    mem_mib = values["mem"] / 1024 / 1024
    print(
        f"namespace={namespace} active_pods={values['pods']} "
        f"cpu_m={values['cpu']:.2f} cpu_pct_of_cluster={cpu_share:.2f} "
        f"memory_mib={mem_mib:.2f}"
    )
print("top_namespace_cpu_requests_end")
print("top_workload_cpu_requests_begin")
for (namespace, kind, name), values in sorted(workload_totals.items(), key=lambda item: item[1]["cpu"], reverse=True)[:limit]:
    cpu_share = (values["cpu"] / alloc_cpu) * 100 if alloc_cpu else 0.0
    mem_mib = values["mem"] / 1024 / 1024
    print(
        f"workload={namespace}/{kind}/{name} active_pods={values['pods']} "
        f"cpu_m={values['cpu']:.2f} cpu_pct_of_cluster={cpu_share:.2f} "
        f"memory_mib={mem_mib:.2f}"
    )
print("top_workload_cpu_requests_end")
print("production_cutover_allowed=false")
PYREQ
