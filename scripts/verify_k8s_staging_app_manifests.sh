#!/usr/bin/env bash
set -euo pipefail

manifest_dir="${1:-deploy/kubernetes/staging/app}"

fail() {
  echo "signalops_k8s_staging_app_manifests_failed: $*" >&2
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

services="$(count_kind Service)"
deployments="$(count_kind Deployment)"
ingresses="$(count_kind Ingress)"
cronjobs="$(count_kind CronJob)"
jobs="$(count_kind Job)"
statefulsets="$(count_kind StatefulSet)"
secrets="$(count_kind Secret)"

[[ "$services" -eq 2 ]] || fail "expected 2 Services, found ${services}"
[[ "$deployments" -eq 2 ]] || fail "expected 2 Deployments, found ${deployments}"
[[ "$ingresses" -eq 0 ]] || fail "expected 0 Ingress resources, found ${ingresses}"
[[ "$cronjobs" -eq 0 ]] || fail "expected 0 CronJobs, found ${cronjobs}"
[[ "$jobs" -eq 0 ]] || fail "expected 0 Jobs, found ${jobs}"
[[ "$statefulsets" -eq 0 ]] || fail "expected 0 StatefulSets, found ${statefulsets}"
[[ "$secrets" -eq 0 ]] || fail "expected 0 Kubernetes Secrets, found ${secrets}"

printf '%s
' "$rendered" | grep -q 'signalops.syncratic.io/production-cutover-allowed: "false"' || fail "production cutover false label missing"
printf '%s
' "$rendered" | grep -q 'vault.hashicorp.com/agent-inject: "true"' || fail "OpenBao injector annotation missing"
printf '%s
' "$rendered" | grep -q 'signalops/data/k8s/app/signalops-gateway-runtime-staging' || fail "staging OpenBao secret path missing"
printf '%s
' "$rendered" | grep -q 'image: signalops-gateway:staging' || fail "staging gateway image missing"
printf '%s
' "$rendered" | grep -q 'image: signalops-web:staging' || fail "staging web image missing"
printf '%s
' "$rendered" | grep -q 'SIGNALOPS_DATABASE_MAX_OPEN_CONNS' || fail "gateway DB pool cap env missing"
printf '%s
' "$rendered" | grep -q 'SIGNALOPS_MARKETOPS_DATABASE_MAX_OPEN_CONNS' || fail "MarketOps DB pool cap env missing"

kubectl apply -k "$manifest_dir" --dry-run=server >/dev/null

cat <<EOF
signalops_k8s_staging_app_manifests_verified
services=${services}
deployments=${deployments}
ingresses=${ingresses}
cronjobs=${cronjobs}
jobs=${jobs}
statefulsets=${statefulsets}
secrets=${secrets}
openbao_secret_path=signalops/data/k8s/app/signalops-gateway-runtime-staging
production_cutover_allowed=false
applied=false
EOF
