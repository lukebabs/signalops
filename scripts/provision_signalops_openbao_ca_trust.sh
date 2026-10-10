#!/usr/bin/env bash
set -euo pipefail

SOURCE_NAMESPACE="${OPENBAO_CA_SOURCE_NAMESPACE:-syncratic-connect}"
SOURCE_CONFIGMAP="${OPENBAO_CA_SOURCE_CONFIGMAP:-syncratic-openbao-ca}"
SOURCE_KEY="${OPENBAO_CA_SOURCE_KEY:-ca.crt}"
OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_TLS_SECRET="${OPENBAO_TLS_SECRET_SOURCE:-openbao-server-tls}"
TARGET_SECRET="${SIGNALOPS_OPENBAO_CA_SECRET:-signalops-openbao-ca}"
TARGET_NAMESPACES_CSV="${SIGNALOPS_OPENBAO_CA_TARGET_NAMESPACES:-signalops-app,signalops-marketops}"

afail() {
  echo "signalops_openbao_ca_trust_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || afail "kubectl is required"
command -v openssl >/dev/null 2>&1 || afail "openssl is required"
command -v base64 >/dev/null 2>&1 || afail "base64 is required"

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

ca_file="$tmpdir/ca.crt"
server_file="$tmpdir/openbao-server.crt"

kubectl get configmap "$SOURCE_CONFIGMAP" -n "$SOURCE_NAMESPACE" -o go-template="{{ index .data \"${SOURCE_KEY}\" }}" >"$ca_file"
[[ -s "$ca_file" ]] || afail "source CA bundle missing or empty: ${SOURCE_NAMESPACE}/${SOURCE_CONFIGMAP}:${SOURCE_KEY}"

kubectl get secret "$OPENBAO_TLS_SECRET" -n "$OPENBAO_NAMESPACE" -o jsonpath='{.data.tls\.crt}' | base64 -d >"$server_file"
[[ -s "$server_file" ]] || afail "OpenBao server certificate missing or empty: ${OPENBAO_NAMESPACE}/${OPENBAO_TLS_SECRET}:tls.crt"

openssl verify -CAfile "$ca_file" "$server_file" >/dev/null || afail "source CA bundle does not validate current OpenBao server certificate"

IFS=',' read -r -a target_namespaces <<<"$TARGET_NAMESPACES_CSV"
for namespace in "${target_namespaces[@]}"; do
  namespace="${namespace//[[:space:]]/}"
  [[ -n "$namespace" ]] || continue
  kubectl get namespace "$namespace" >/dev/null || afail "target namespace missing: ${namespace}"
  kubectl create secret generic "$TARGET_SECRET" -n "$namespace" \
    --from-file=ca.crt="$ca_file" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  key_count="$(kubectl get secret "$TARGET_SECRET" -n "$namespace" -o go-template='{{len .data}}')"
  [[ "$key_count" == "1" ]] || afail "unexpected key count in ${namespace}/${TARGET_SECRET}: ${key_count}"
done

cat <<EOF
signalops_openbao_ca_trust_verified
source_configmap=${SOURCE_NAMESPACE}/${SOURCE_CONFIGMAP}
source_key=${SOURCE_KEY}
target_secret=${TARGET_SECRET}
target_namespaces=${TARGET_NAMESPACES_CSV}
server_certificate_validated=true
secret_values=public_ca_bundle_only
production_cutover_allowed=false
EOF
