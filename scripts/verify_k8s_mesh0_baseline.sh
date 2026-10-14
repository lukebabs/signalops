#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "signalops_k8s_mesh0_baseline_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v helm >/dev/null 2>&1 || fail "helm is required"

server_version="$(kubectl version 2>/dev/null | awk -F': ' '/Server Version/ {print $2; exit}')"
[[ -n "$server_version" ]] || fail "could not read Kubernetes server version"

cilium_release="$(helm list -n kube-system 2>/dev/null | awk '$1 == "cilium" {print $1 "|" $8 "|" $9; exit}')"
[[ -n "$cilium_release" ]] || fail "cilium Helm release not found"

kubectl get pod -n kube-system -l k8s-app=cilium --no-headers | grep -q 'Running' || fail "no running Cilium pod found"
kubectl get pod -n kube-system -l k8s-app=cilium-envoy --no-headers | grep -q 'Running' || fail "no running Cilium Envoy pod found"

for crd in gatewayclasses.gateway.networking.k8s.io gateways.gateway.networking.k8s.io httproutes.gateway.networking.k8s.io; do
  kubectl get crd "$crd" >/dev/null || fail "Gateway API CRD missing: $crd"
done

gateway_classes="$(kubectl get gatewayclass --no-headers 2>/dev/null | wc -l | tr -d ' ')"

nginx_service="$(kubectl get svc ingress-nginx-controller -n ingress-nginx -o jsonpath='{.spec.type}|{.status.loadBalancer.ingress[0].ip}|{.spec.ports[?(@.port==80)].nodePort}|{.spec.ports[?(@.port==443)].nodePort}' 2>/dev/null || true)"
[[ -n "$nginx_service" ]] || fail "ingress-nginx controller service not found"

traefik_line="$(docker ps --filter name=traefik --format '{{.Names}}|{{.Image}}|{{.Ports}}' | head -1)"
[[ -n "$traefik_line" ]] || fail "Docker Traefik container not found"
printf '%s
' "$traefik_line" | grep -q '0.0.0.0:80->80/tcp' || fail "Traefik does not expose host port 80"
printf '%s
' "$traefik_line" | grep -q '0.0.0.0:443->443/tcp' || fail "Traefik does not expose host port 443"

if kubectl get crd virtualservices.networking.istio.io >/dev/null 2>&1; then
  istio_crds=true
else
  istio_crds=false
fi

cat <<EOF
signalops_k8s_mesh0_baseline_verified
kubernetes_server_version=${server_version}
cilium_release=${cilium_release}
cilium_pod=running
cilium_envoy_pod=running
gateway_api_crds=present
gateway_classes=${gateway_classes}
istio_crds_present=${istio_crds}
ingress_nginx_service=${nginx_service}
docker_traefik_edge=present_80_443
public_traffic_cutover_allowed=false
recommended_next_gate=mesh0_implementation_mode_and_rollback_plan
EOF
