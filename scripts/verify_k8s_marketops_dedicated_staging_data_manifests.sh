#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
manifest_dir="$repo_dir/deploy/kubernetes/staging/marketops-data"

fail() {
  echo "signalops_k8s_marketops_dedicated_staging_data_manifests_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -d "$manifest_dir" ]] || fail "manifest directory missing: ${manifest_dir}"

rendered="$(kubectl kustomize "$manifest_dir")"

service_count="$(grep -c '^kind: Service$' <<<"$rendered" || true)"
statefulset_count="$(grep -c '^kind: StatefulSet$' <<<"$rendered" || true)"
secret_count="$(grep -c '^kind: Secret$' <<<"$rendered" || true)"
job_count="$(grep -Ec '^kind: (Job|CronJob)$' <<<"$rendered" || true)"
ingress_count="$(grep -Ec '^kind: Ingress$' <<<"$rendered" || true)"

[[ "$service_count" == "2" ]] || fail "expected 2 Services, found ${service_count}"
[[ "$statefulset_count" == "2" ]] || fail "expected 2 StatefulSets, found ${statefulset_count}"
[[ "$secret_count" == "0" ]] || fail "manifests must not commit runtime Secrets"
[[ "$job_count" == "0" ]] || fail "data manifests must not create Jobs or CronJobs"
[[ "$ingress_count" == "0" ]] || fail "staging data services must not expose Ingress"

grep -q 'namespace: signalops-data' <<<"$rendered" || fail "resources must be scoped to signalops-data"
grep -q 'name: marketops-postgres-staging' <<<"$rendered" || fail "primary staging service missing"
grep -q 'name: marketops-timescaledb-staging' <<<"$rendered" || fail "temporal staging service missing"
grep -q 'app.kubernetes.io/component: database' <<<"$rendered" || fail "database component labels missing"
grep -q 'signalops.syncratic.io/purpose: marketops-k8s-staging' <<<"$rendered" || fail "staging purpose label missing"
grep -q 'signalops.syncratic.io/production-cutover-allowed: "false"' <<<"$rendered" || fail "production cutover guard label missing"
grep -q 'storageClassName: local-path' <<<"$rendered" || fail "expected disposable local-path storage for staging gate"

cat <<EOF
signalops_k8s_marketops_dedicated_staging_data_manifests_verified
namespace=signalops-data
primary_service=marketops-postgres-staging
temporal_service=marketops-timescaledb-staging
services=${service_count}
statefulsets=${statefulset_count}
committed_secrets=${secret_count}
production_cutover_allowed=false
EOF
