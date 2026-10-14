#!/usr/bin/env bash
set -euo pipefail

manifest_dir="${1:-deploy/kubernetes/staging/mesh-route}"

fail() {
  echo "signalops_k8s_mesh2_route_manifests_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

rendered="$(kubectl kustomize "$manifest_dir")"
[[ -n "$rendered" ]] || fail "kustomize render produced no output"

count_kind() {
  local kind="$1"
  printf '%s\n' "$rendered" | awk -v kind="$kind" '$0 == "kind: " kind { c++ } END { print c+0 }'
}

namespaces="$(count_kind Namespace)"
httproutes="$(count_kind HTTPRoute)"
networkpolicies="$(count_kind NetworkPolicy)"
ingresses="$(count_kind Ingress)"
gateways="$(count_kind Gateway)"
secrets="$(count_kind Secret)"
services="$(count_kind Service)"
deployments="$(count_kind Deployment)"
cronjobs="$(count_kind CronJob)"
jobs="$(count_kind Job)"
issuers="$(count_kind Issuer)"
certificates="$(count_kind Certificate)"
referencegrants="$(count_kind ReferenceGrant)"

[[ "$namespaces" -eq 1 ]] || fail "expected 1 Namespace label resource, found ${namespaces}"
[[ "$httproutes" -eq 1 ]] || fail "expected 1 HTTPRoute, found ${httproutes}"
[[ "$networkpolicies" -eq 1 ]] || fail "expected 1 NetworkPolicy, found ${networkpolicies}"
[[ "$ingresses" -eq 0 ]] || fail "expected 0 Ingress resources, found ${ingresses}"
[[ "$gateways" -eq 0 ]] || fail "expected 0 Gateway resources; must reuse existing Istio public-ingress, found ${gateways}"
[[ "$secrets" -eq 0 ]] || fail "expected 0 Secrets, found ${secrets}"
[[ "$services" -eq 0 ]] || fail "expected 0 Services, found ${services}"
[[ "$deployments" -eq 0 ]] || fail "expected 0 Deployments, found ${deployments}"
[[ "$cronjobs" -eq 0 ]] || fail "expected 0 CronJobs, found ${cronjobs}"
[[ "$jobs" -eq 0 ]] || fail "expected 0 Jobs, found ${jobs}"
[[ "$issuers" -eq 1 ]] || fail "expected 1 staging TLS Issuer, found ${issuers}"
[[ "$certificates" -eq 1 ]] || fail "expected 1 staging TLS Certificate, found ${certificates}"
[[ "$referencegrants" -eq 1 ]] || fail "expected 1 staging TLS ReferenceGrant, found ${referencegrants}"

printf '%s\n' "$rendered" | grep -q 'hostnames:' || fail "HTTPRoute hostnames missing"
printf '%s\n' "$rendered" | grep -q 'signalops-staging.syncratic.co' || fail "staging hostname missing"
printf '%s\n' "$rendered" | grep -q 'namespace: istio-system' || fail "HTTPRoute must target the existing istio-system Gateway"
printf '%s\n' "$rendered" | grep -q 'sectionName: http' || fail "HTTPRoute must attach to the HTTP listener for Mesh-2 compatibility"
printf '%s\n' "$rendered" | grep -q 'sectionName: signalops-staging-https' || fail "HTTPRoute must attach to the staging HTTPS listener for Mesh-3 auth parity"
printf '%s\n' "$rendered" | grep -q 'name: signalops-web' || fail "web backend missing"
printf '%s\n' "$rendered" | grep -q 'name: signalops-gateway' || fail "gateway backend missing"
printf '%s\n' "$rendered" | grep -q 'gateway.networking.k8s.io/gateway-name: public-ingress' || fail "Istio gateway pod selector missing"
printf '%s\n' "$rendered" | grep -q 'syncratic.io/public-ingress: "true"' || fail "namespace public-ingress attachment label missing"
printf '%s\n' "$rendered" | grep -q 'signalops.syncratic.io/production-cutover-allowed: "false"' || fail "production cutover false label missing"

kubectl apply -k "$manifest_dir" --dry-run=server >/dev/null

cat <<EOF
signalops_k8s_mesh2_route_manifests_verified
namespaces=${namespaces}
httproutes=${httproutes}
networkpolicies=${networkpolicies}
ingresses=${ingresses}
gateways=${gateways}
secrets=${secrets}
services=${services}
deployments=${deployments}
cronjobs=${cronjobs}
jobs=${jobs}
issuers=${issuers}
certificates=${certificates}
referencegrants=${referencegrants}
staging_hostname=signalops-staging.syncratic.co
parent_gateway=istio-system/public-ingress
listener=http+signalops-staging-https
production_cutover_allowed=false
applied=false
EOF
