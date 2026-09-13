#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

fail() {
  echo "signalops_k8s_connect_broker_manifests_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

rendered="$(mktemp)"
trap 'rm -f "$rendered"' EXIT

kubectl kustomize deploy/kubernetes/staging/connect-broker >"$rendered"

python3 - "$rendered" <<'PYCHECK'
import sys
from pathlib import Path

text = Path(sys.argv[1]).read_text()

def count_kind(kind: str) -> int:
    return sum(1 for doc in text.split('\n---\n') if f'\nkind: {kind}\n' in f'\n{doc}\n')

expected = {
    'Service': 1,
    'Deployment': 1,
    'NetworkPolicy': 1,
    'Secret': 0,
    'Ingress': 0,
    'Gateway': 0,
    'HTTPRoute': 0,
}
for kind, want in expected.items():
    got = count_kind(kind)
    if got != want:
        raise SystemExit(f'expected {want} {kind}, got {got}')

required = [
    'namespace: signalops-data',
    'name: signalops-redpanda-staging',
    'app.kubernetes.io/component: broker',
    'signalops.syncratic.io/purpose: signal-connect-k8s-staging',
    'signalops.syncratic.io/production-cutover-allowed: "false"',
    'type: ClusterIP',
    'port: 9092',
    'redpandadata/redpanda:v25.3.1',
    'signalops-redpanda-staging.signalops-data.svc.cluster.local:9092',
    'emptyDir: {}',
    'name: allow-redpanda-staging-self-and-dns',
    'kubernetes.io/metadata.name: kube-system',
]
for marker in required:
    if marker not in text:
        raise SystemExit(f'missing required marker: {marker}')

for forbidden in ['NodePort', 'LoadBalancer', 'hostPort:', 'persistentVolumeClaim:', 'storageClassName:', 'tls-skip-verify']:
    if forbidden in text:
        raise SystemExit(f'forbidden marker present: {forbidden}')
PYCHECK

kubectl apply -k deploy/kubernetes/staging/connect-broker --dry-run=server >/dev/null

cat <<'EOF'
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
EOF
