#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_NAMESPACE:-signalops-marketops}"
MANIFEST_DIR="${SIGNALOPS_K8S_MARKETOPS_JOBS_MANIFEST_DIR:-deploy/kubernetes/staging/marketops-jobs}"

fail() {
  echo "signalops_k8s_marketops_suspended_cronjob_apply_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh "$MANIFEST_DIR"
kubectl apply -k "$MANIFEST_DIR" >/dev/null

python3 - "$NAMESPACE" <<'PY'
import json
import subprocess
import sys

namespace = sys.argv[1]
selector = "app.kubernetes.io/name=signalops-marketops-k8s-job-runner"
raw = subprocess.check_output([
    "kubectl", "get", "cronjob", "-n", namespace, "-l", selector, "-o", "json"
], text=True)
payload = json.loads(raw)
items = payload.get("items", [])
expected = {
    "marketops-intraday",
    "marketops-sri-refresh",
    "marketops-sri-holdings-refresh",
    "marketops-fmp-annual-financial",
    "marketops-saf-benchmark",
}
found = {item.get("metadata", {}).get("name") for item in items}
if found != expected:
    raise SystemExit(f"unexpected CronJob set: found={sorted(found)} expected={sorted(expected)}")
for item in items:
    name = item["metadata"]["name"]
    spec = item.get("spec", {})
    if spec.get("suspend") is not True:
        raise SystemExit(f"{name} is not suspended")
    if spec.get("concurrencyPolicy") != "Forbid":
        raise SystemExit(f"{name} concurrencyPolicy is not Forbid")
    status = item.get("status", {})
    if status.get("active"):
        raise SystemExit(f"{name} has active jobs despite being suspended")
    if status.get("lastScheduleTime"):
        raise SystemExit(f"{name} has lastScheduleTime despite suspended apply smoke")

raw_jobs = subprocess.check_output([
    "kubectl", "get", "jobs", "-n", namespace, "-l", selector, "-o", "json"
], text=True)
jobs = json.loads(raw_jobs).get("items", [])
if jobs:
    raise SystemExit(f"unexpected Jobs created: {[j['metadata']['name'] for j in jobs]}")

raw_pods = subprocess.check_output([
    "kubectl", "get", "pods", "-n", namespace, "-l", selector, "-o", "json"
], text=True)
pods = json.loads(raw_pods).get("items", [])
if pods:
    raise SystemExit(f"unexpected Pods created: {[p['metadata']['name'] for p in pods]}")
PY

cat <<EOF
signalops_k8s_marketops_suspended_cronjob_apply_smoke_verified
namespace=${NAMESPACE}
cronjobs=5
suspended=true
concurrency_policy=Forbid
jobs_created=0
pods_created=0
provider_polling=false
production_cutover_allowed=false
EOF
