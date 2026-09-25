#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

fail() {
  echo "signalops_k8s_raw_worker_manifests_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

rendered="$(mktemp)"
trap 'rm -f "$rendered"' EXIT

kubectl kustomize deploy/kubernetes/staging/raw-worker >"$rendered"

python3 - "$rendered" <<'PYCHECK'
import sys
from pathlib import Path

text = Path(sys.argv[1]).read_text()

def count_kind(kind: str) -> int:
    return sum(1 for doc in text.split('\n---\n') if f'\nkind: {kind}\n' in f'\n{doc}\n')

expected = {
    'Deployment': 1,
    'Secret': 0,
    'Service': 0,
    'Ingress': 0,
    'Gateway': 0,
    'HTTPRoute': 0,
    'CronJob': 0,
}
for kind, want in expected.items():
    got = count_kind(kind)
    if got != want:
        raise SystemExit(f'expected {want} {kind}, got {got}')

required = [
    'namespace: signalops-connect',
    'name: signalops-raw-worker',
    'app.kubernetes.io/component: normalized-algorithm-worker',
    'replicas: 0',
    'image: ghcr.io/syncratic-inc/signalops-python-worker:staging',
    'name: signalops-ghcr-pull',
    'SIGNALOPS_BROKER_BROKERS',
    'signalops-redpanda-staging.signalops-data.svc.cluster.local:9092',
    'signalops.kubernetes-staging.normalized.v1',
    'signalops.kubernetes-staging.retry.algorithm.v1',
    'signalops.kubernetes-staging.dlq.algorithm.v1',
    'signalops.kubernetes-staging.signal.v1',
    'SIGNALOPS_WORKER_MAX_MESSAGES',
    'production-cutover-allowed: "false"',
]
for marker in required:
    if marker not in text:
        raise SystemExit(f'missing required marker: {marker}')

for forbidden in ['NodePort', 'LoadBalancer', 'hostPort:', 'vault.hashicorp.com', 'tls-skip-verify']:
    if forbidden in text:
        raise SystemExit(f'forbidden marker present: {forbidden}')
PYCHECK

kubectl apply -k deploy/kubernetes/staging/raw-worker --dry-run=server >/dev/null

cat <<'EOF'
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
EOF
