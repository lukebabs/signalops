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

[[ "$cronjobs" -eq 13 ]] || fail "expected 13 CronJobs, found ${cronjobs}"
[[ "$networkpolicies" -eq 1 ]] || fail "expected 1 NetworkPolicy, found ${networkpolicies}"
[[ "$secrets" -eq 0 ]] || fail "expected 0 Kubernetes Secrets, found ${secrets}"
[[ "$services" -eq 0 ]] || fail "expected 0 Services, found ${services}"
[[ "$deployments" -eq 0 ]] || fail "expected 0 Deployments, found ${deployments}"
[[ "$statefulsets" -eq 0 ]] || fail "expected 0 StatefulSets, found ${statefulsets}"

for job in marketops-intraday marketops-warm-eod marketops-daily-postclose marketops-sri-refresh marketops-sri-holdings-refresh marketops-fmp-continuation marketops-fmp-annual-financial marketops-task-retry marketops-postclose-recovery marketops-risk-reward marketops-operations-monitor marketops-retention-governance marketops-saf-benchmark; do
  grep -q "name: ${job}" <<<"$rendered" || fail "CronJob ${job} missing"
  grep -q "signalops.syncratic.io/job-id: ${job}" <<<"$rendered" || fail "job-id label missing for ${job}"
  grep -q -- "- ${job}" <<<"$rendered" || fail "entrypoint arg missing for ${job}"
done

[[ "$(grep -c 'suspend: true' <<<"$rendered")" -eq 13 ]] || fail "all CronJobs must be suspended in staging scaffold"
[[ "$(grep -c 'concurrencyPolicy: Forbid' <<<"$rendered")" -eq 13 ]] || fail "all CronJobs must forbid concurrency"
grep -q 'timeZone: America/New_York' <<<"$rendered" || fail "timezone policy missing"
grep -q 'signalops/data/k8s/marketops/marketops-worker-runtime-staging' <<<"$rendered" || fail "OpenBao marketops worker runtime path missing"
grep -q 'vault.hashicorp.com/role: signalops-marketops' <<<"$rendered" || fail "OpenBao marketops role missing"
grep -q 'vault.hashicorp.com/service: https://openbao.openbao.svc:8200' <<<"$rendered" || fail "OpenBao HTTPS service annotation missing"
grep -q 'vault.hashicorp.com/tls-secret: signalops-openbao-ca' <<<"$rendered" || fail "OpenBao CA trust secret annotation missing"
grep -q 'vault.hashicorp.com/ca-cert: /vault/tls/ca.crt' <<<"$rendered" || fail "OpenBao CA cert path annotation missing"
if grep -q 'vault.hashicorp.com/tls-skip-verify' <<<"$rendered"; then
  fail "tls-skip-verify must not be present in MarketOps staging CronJobs"
fi
grep -q 'image: ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging' <<<"$rendered" || fail "staging job-runner image missing"
grep -q 'SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED' <<<"$rendered" || fail "data-boundary env guard missing"
grep -q 'production-cutover-allowed: "false"' <<<"$rendered" || fail "production cutover guard missing"

kubectl apply -k "$manifest_dir" --dry-run=server >/dev/null

cat <<EOF
signalops_k8s_marketops_scheduled_jobs_manifests_verified
cronjobs=${cronjobs}
admin_scheduler_parity=complete
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
