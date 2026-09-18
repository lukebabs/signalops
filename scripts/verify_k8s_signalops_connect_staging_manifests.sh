#!/usr/bin/env bash
set -euo pipefail

manifest_dir="${1:-deploy/kubernetes/staging/connect}"

fail() {
  echo "signalops_k8s_connect_staging_manifests_failed: $*" >&2
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

deployments="$(count_kind Deployment)"
networkpolicies="$(count_kind NetworkPolicy)"
secrets="$(count_kind Secret)"
services="$(count_kind Service)"
cronjobs="$(count_kind CronJob)"
statefulsets="$(count_kind StatefulSet)"

[[ "$deployments" -eq 2 ]] || fail "expected 2 Deployments, found ${deployments}"
[[ "$networkpolicies" -eq 1 ]] || fail "expected 1 NetworkPolicy, found ${networkpolicies}"
[[ "$secrets" -eq 0 ]] || fail "expected 0 Kubernetes Secrets, found ${secrets}"
[[ "$services" -eq 0 ]] || fail "expected 0 Services, found ${services}"
[[ "$cronjobs" -eq 0 ]] || fail "expected 0 CronJobs, found ${cronjobs}"
[[ "$statefulsets" -eq 0 ]] || fail "expected 0 StatefulSets, found ${statefulsets}"

for worker in connect-persister connect-outbox; do
  grep -q "signalops.syncratic.io/workload-id: ${worker}" <<<"$rendered" || fail "workload label missing for ${worker}"
  grep -q -- "- ${worker}" <<<"$rendered" || fail "entrypoint arg missing for ${worker}"
done

for required in \
  'replicas: 0' \
  'signalops/data/k8s/connect/connect-worker-runtime-staging' \
  'vault.hashicorp.com/role: signalops-connect' \
  'vault.hashicorp.com/service: https://openbao.openbao.svc:8200' \
  'vault.hashicorp.com/tls-secret: signalops-openbao-ca' \
  'vault.hashicorp.com/ca-cert: /vault/tls/ca.crt' \
  'SIGNALOPS_CONNECT_SHADOW_MODE' \
  'production-cutover-allowed: "false"' \
  'image: ghcr.io/syncratic-inc/signalops-connect-k8s-worker:staging'; do
  grep -q "$required" <<<"$rendered" || fail "required marker missing: ${required}"
done

if grep -q 'vault.hashicorp.com/tls-skip-verify' <<<"$rendered"; then
  fail "tls-skip-verify must not be present in Signal-Connect staging workloads"
fi

kubectl apply -k "$manifest_dir" --dry-run=server >/dev/null

cat <<EOF
signalops_k8s_connect_staging_manifests_verified
deployments=${deployments}
networkpolicies=${networkpolicies}
secrets=${secrets}
services=${services}
cronjobs=${cronjobs}
statefulsets=${statefulsets}
replicas=0
openbao_secret_path=signalops/data/k8s/connect/connect-worker-runtime-staging
shadow_mode=true
provider_polling=false
production_cutover_allowed=false
applied=false
EOF
