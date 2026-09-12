#!/usr/bin/env bash
set -euo pipefail

manifest_dir="${1:-deploy/kubernetes/staging/marketops-jobs}"

fail() {
  echo "signalops_k8s_marketops_scheduled_jobs_manifests_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

rendered="$(kubectl kustomize "$manifest_dir")"
[[ -n "$rendered" ]] || fail "kustomize render produced no output"

count_kind() {
  local kind="$1"
  printf '%s
' "$rendered" | awk -v kind="$kind" '$0 == "kind: " kind { c++ } END { print c+0 }'
}

cronjobs="$(count_kind CronJob)"
networkpolicies="$(count_kind NetworkPolicy)"
secrets="$(count_kind Secret)"
services="$(count_kind Service)"
deployments="$(count_kind Deployment)"
statefulsets="$(count_kind StatefulSet)"

[[ "$cronjobs" -eq 5 ]] || fail "expected 5 CronJobs, found ${cronjobs}"
[[ "$networkpolicies" -eq 1 ]] || fail "expected 1 NetworkPolicy, found ${networkpolicies}"
[[ "$secrets" -eq 0 ]] || fail "expected 0 Kubernetes Secrets, found ${secrets}"
[[ "$services" -eq 0 ]] || fail "expected 0 Services, found ${services}"
[[ "$deployments" -eq 0 ]] || fail "expected 0 Deployments, found ${deployments}"
[[ "$statefulsets" -eq 0 ]] || fail "expected 0 StatefulSets, found ${statefulsets}"

for job in marketops-intraday marketops-sri-refresh marketops-sri-holdings-refresh marketops-fmp-annual-financial marketops-saf-benchmark; do
  printf '%s
' "$rendered" | grep -q "name: ${job}" || fail "CronJob ${job} missing"
  printf '%s
' "$rendered" | grep -q "signalops.syncratic.io/job-id: ${job}" || fail "job-id label missing for ${job}"
  printf '%s
' "$rendered" | grep -q -- "- ${job}" || fail "entrypoint arg missing for ${job}"
done

[[ "$(printf '%s
' "$rendered" | grep -c 'suspend: true')" -eq 5 ]] || fail "all CronJobs must be suspended in staging scaffold"
[[ "$(printf '%s
' "$rendered" | grep -c 'concurrencyPolicy: Forbid')" -eq 5 ]] || fail "all CronJobs must forbid concurrency"
printf '%s
' "$rendered" | grep -q 'timeZone: America/New_York' || fail "timezone policy missing"
printf '%s
' "$rendered" | grep -q 'signalops/data/k8s/marketops/marketops-worker-runtime-staging' || fail "OpenBao marketops worker runtime path missing"
printf '%s
' "$rendered" | grep -q 'vault.hashicorp.com/role: signalops-marketops' || fail "OpenBao marketops role missing"
printf '%s
' "$rendered" | grep -q 'image: ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging' || fail "staging job-runner image missing"
printf '%s
' "$rendered" | grep -q 'SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED' || fail "data-boundary env guard missing"
printf '%s
' "$rendered" | grep -q 'production-cutover-allowed: "false"' || fail "production cutover guard missing"

kubectl apply -k "$manifest_dir" --dry-run=server >/dev/null

cat <<EOF
signalops_k8s_marketops_scheduled_jobs_manifests_verified
cronjobs=${cronjobs}
networkpolicies=${networkpolicies}
secrets=${secrets}
services=${services}
deployments=${deployments}
statefulsets=${statefulsets}
suspended=true
concurrency_policy=Forbid
openbao_secret_path=signalops/data/k8s/marketops/marketops-worker-runtime-staging
production_cutover_allowed=false
applied=false
EOF
