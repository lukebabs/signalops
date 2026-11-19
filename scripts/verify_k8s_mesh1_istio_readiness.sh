#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "signalops_k8s_mesh1_istio_readiness_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v helm >/dev/null 2>&1 || fail "helm is required"

istio_base_status="$(helm status istio-base -n istio-system -o json 2>/dev/null | sed -n 's/.*"status":"\([^"]*\)".*/\1/p' | head -1)"
[[ "$istio_base_status" == "deployed" ]] || fail "istio-base is not deployed"

istiod_status="$(helm status istiod -n istio-system -o json 2>/dev/null | sed -n 's/.*"status":"\([^"]*\)".*/\1/p' | head -1)"
[[ "$istiod_status" == "deployed" ]] || fail "istiod is not deployed"

kubectl get pod -n istio-system -l app=istiod --no-headers 2>/dev/null | grep -q 'Running' || fail "no running istiod pod found"

for crd in \
  gateways.gateway.networking.k8s.io \
  httproutes.gateway.networking.k8s.io \
  gatewayclasses.gateway.networking.k8s.io \
  authorizationpolicies.security.istio.io \
  peerauthentications.security.istio.io \
  virtualservices.networking.istio.io; do
  kubectl get crd "$crd" >/dev/null 2>&1 || fail "required CRD missing: $crd"
done

gateway_class="$(kubectl get gatewayclass istio -o jsonpath='{.status.conditions[?(@.type=="Accepted")].status}' 2>/dev/null || true)"
[[ "$gateway_class" == "True" ]] || fail "GatewayClass istio is not accepted"

public_gateway_programmed="$(kubectl get gateway public-ingress -n istio-system -o jsonpath='{.status.conditions[?(@.type=="Programmed")].status}' 2>/dev/null || true)"
[[ "$public_gateway_programmed" == "True" ]] || fail "istio-system/public-ingress Gateway is not programmed"

public_gateway_address="$(kubectl get gateway public-ingress -n istio-system -o jsonpath='{.status.addresses[0].value}' 2>/dev/null || true)"
[[ -n "$public_gateway_address" ]] || fail "istio-system/public-ingress Gateway has no address"

public_gateway_service="$(kubectl get svc public-ingress-istio -n istio-system -o jsonpath='{.spec.type}|{.status.loadBalancer.ingress[0].ip}|{.spec.ports[?(@.port==80)].nodePort}|{.spec.ports[?(@.port==443)].nodePort}' 2>/dev/null || true)"
[[ -n "$public_gateway_service" ]] || fail "istio-system/public-ingress-istio service not found"

for ns in signalops-app signalops-connect signalops-cyberops signalops-data signalops-identity signalops-marketops signalops-observability; do
  kubectl get namespace "$ns" >/dev/null 2>&1 || fail "SignalOps namespace missing: $ns"
  injection="$(kubectl get namespace "$ns" -o jsonpath='{.metadata.labels.istio-injection}' 2>/dev/null || true)"
  dataplane="$(kubectl get namespace "$ns" -o jsonpath='{.metadata.labels.istio\.io/dataplane-mode}' 2>/dev/null || true)"
  [[ -z "$injection" ]] || fail "$ns has istio-injection=$injection; SignalOps injection is not approved"
  [[ -z "$dataplane" ]] || fail "$ns has istio.io/dataplane-mode=$dataplane; SignalOps mesh enrollment is not approved"
done

cat <<EOF
signalops_k8s_mesh1_istio_readiness_verified
istio_base=${istio_base_status}
istiod=${istiod_status}
istiod_pod=running
gateway_api_crds=present
istio_crds=present
gateway_class_istio=accepted
public_gateway=istio-system/public-ingress
public_gateway_programmed=true
public_gateway_address=${public_gateway_address}
public_gateway_service=${public_gateway_service}
signalops_namespace_auto_injection=false
signalops_dataplane_enrollment=false
production_cutover_allowed=false
recommended_next_gate=mesh2_signalops_staging_route_parity
EOF
