#!/usr/bin/env bash
set -euo pipefail

MAX_CPU_REQUEST_PCT="${SIGNALOPS_K8S_CAPACITY_MAX_CPU_REQUEST_PCT:-85}"
MAX_MEMORY_REQUEST_PCT="${SIGNALOPS_K8S_CAPACITY_MAX_MEMORY_REQUEST_PCT:-80}"
mode="${1:---report}"

fail() {
  printf 'signalops_k8s_capacity_headroom_failed: %s\n' "$*" >&2
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

report="$(python3 - "$MAX_CPU_REQUEST_PCT" "$MAX_MEMORY_REQUEST_PCT" "$nodes_file" "$pods_file" <<'PYCAP'
import json
import sys

max_cpu_pct = float(sys.argv[1])
max_mem_pct = float(sys.argv[2])
with open(sys.argv[3], encoding='utf-8') as fh:
    nodes = json.load(fh)
with open(sys.argv[4], encoding='utf-8') as fh:
    pods = json.load(fh)

def parse_cpu(value):
    if value is None or value == "":
        return 0.0
    value = str(value)
    if value.endswith('m'):
        return float(value[:-1])
    if value.endswith('n'):
        return float(value[:-1]) / 1_000_000.0
    if value.endswith('u'):
        return float(value[:-1]) / 1000.0
    return float(value) * 1000.0

def parse_memory(value):
    if value is None or value == "":
        return 0.0
    value = str(value)
    units = {
        'Ki': 1024,
        'Mi': 1024**2,
        'Gi': 1024**3,
        'Ti': 1024**4,
        'K': 1000,
        'M': 1000**2,
        'G': 1000**3,
        'T': 1000**4,
    }
    for suffix, mult in units.items():
        if value.endswith(suffix):
            return float(value[:-len(suffix)]) * mult
    return float(value)

alloc_cpu = 0.0
alloc_mem = 0.0
for node in nodes.get('items', []):
    alloc = node.get('status', {}).get('allocatable', {})
    alloc_cpu += parse_cpu(alloc.get('cpu'))
    alloc_mem += parse_memory(alloc.get('memory'))

requested_cpu = 0.0
requested_mem = 0.0
active_pods = 0
for pod in pods.get('items', []):
    phase = pod.get('status', {}).get('phase')
    if phase in {'Succeeded', 'Failed'}:
        continue
    active_pods += 1
    containers = list(pod.get('spec', {}).get('containers', []))
    init_containers = list(pod.get('spec', {}).get('initContainers', []))
    app_cpu = 0.0
    app_mem = 0.0
    for container in containers:
        requests = container.get('resources', {}).get('requests', {})
        app_cpu += parse_cpu(requests.get('cpu'))
        app_mem += parse_memory(requests.get('memory'))
    init_cpu = 0.0
    init_mem = 0.0
    for container in init_containers:
        requests = container.get('resources', {}).get('requests', {})
        init_cpu = max(init_cpu, parse_cpu(requests.get('cpu')))
        init_mem = max(init_mem, parse_memory(requests.get('memory')))
    requested_cpu += max(app_cpu, init_cpu)
    requested_mem += max(app_mem, init_mem)

cpu_pct = (requested_cpu / alloc_cpu) * 100.0 if alloc_cpu else 0.0
mem_pct = (requested_mem / alloc_mem) * 100.0 if alloc_mem else 0.0
status = 'ok' if cpu_pct <= max_cpu_pct and mem_pct <= max_mem_pct else 'blocked'
print('signalops_k8s_capacity_headroom_report')
print(f'status={status}')
print(f'nodes={len(nodes.get("items", []))}')
print(f'active_pods={active_pods}')
print(f'allocatable_cpu_m={round(alloc_cpu, 2)}')
print(f'requested_cpu_m={round(requested_cpu, 2)}')
print(f'cpu_request_pct={round(cpu_pct, 2)}')
print(f'max_cpu_request_pct={max_cpu_pct:g}')
print(f'allocatable_memory_mib={round(alloc_mem / 1024 / 1024, 2)}')
print(f'requested_memory_mib={round(requested_mem / 1024 / 1024, 2)}')
print(f'memory_request_pct={round(mem_pct, 2)}')
print(f'max_memory_request_pct={max_mem_pct:g}')
print('production_cutover_allowed=false')
PYCAP
)"

printf '%s\n' "$report"
status="$(printf '%s\n' "$report" | awk -F= '$1=="status"{print $2; exit}')"
if [[ "$mode" == "--strict" && "$status" != "ok" ]]; then
  fail "capacity headroom status=${status}"
fi
