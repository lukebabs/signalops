#!/usr/bin/env bash
set -euo pipefail

APP_NAMESPACE="${SIGNALOPS_K8S_STAGING_NAMESPACE:-signalops-app}"
APP_MANIFEST_DIR="${SIGNALOPS_K8S_STAGING_APP_MANIFEST_DIR:-deploy/kubernetes/staging/app}"
MESH_MANIFEST_DIR="${SIGNALOPS_K8S_MESH2_MANIFEST_DIR:-deploy/kubernetes/staging/mesh-route}"
DOTENV_PATH="${SIGNALOPS_E2E_ENV_FILE:-.env}"
HOSTNAME="${SIGNALOPS_K8S_SIGNALOPS_STAGING_HOSTNAME:-signalops-staging.syncratic.co}"
GATEWAY_NAMESPACE="${SIGNALOPS_K8S_ISTIO_NAMESPACE:-istio-system}"
GATEWAY_NAME="${SIGNALOPS_K8S_ISTIO_GATEWAY:-public-ingress}"
GATEWAY_SERVICE="${SIGNALOPS_K8S_ISTIO_GATEWAY_SERVICE:-public-ingress-istio}"
RUN_ID="${SIGNALOPS_K8S_STRIPE_WEBHOOK_RUN_ID:-k8s-stripe-webhook-$(date -u +%Y%m%dT%H%M%SZ)}"

fail() {
  printf 'signalops_k8s_stripe_webhook_route_parity_failed: %s\n' "$*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$DOTENV_PATH"
: "${STRIPE_WEBHOOK_SECRET:?STRIPE_WEBHOOK_SECRET is required for signed K3s webhook route parity}"

cleanup() {
  kubectl scale deployment/signalops-gateway deployment/signalops-web -n "$APP_NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
}
trap cleanup EXIT

scripts/verify_k8s_mesh1_istio_readiness.sh >/dev/null
scripts/verify_k8s_staging_app_manifests.sh "$APP_MANIFEST_DIR" >/dev/null
scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh "$MESH_MANIFEST_DIR" >/dev/null
scripts/verify_k8s_mesh4_ingress_dns_cutover_plan.sh >/dev/null

admin_token="${OPENBAO_TOKEN:-${BAO_TOKEN:-${VAULT_TOKEN:-${OPENBAO_ADMIN_TOKEN:-}}}}"
[[ -n "$admin_token" ]] || fail "OPENBAO_ADMIN_TOKEN, OPENBAO_TOKEN, BAO_TOKEN, or VAULT_TOKEN is required to reconcile app OpenBao role"
OPENBAO_TOKEN="$admin_token" scripts/provision_openbao_signalops_app_staging.sh >/dev/null

kubectl apply -k "$APP_MANIFEST_DIR" >/dev/null
kubectl scale deployment/signalops-gateway deployment/signalops-web -n "$APP_NAMESPACE" --replicas=0 >/dev/null
kubectl rollout status deployment/signalops-gateway -n "$APP_NAMESPACE" --timeout=60s >/dev/null || true
kubectl set resources deployment/signalops-gateway -n "$APP_NAMESPACE" -c gateway --requests=cpu=25m,memory=128Mi --limits=cpu=500m,memory=768Mi >/dev/null
kubectl patch deployment/signalops-gateway -n "$APP_NAMESPACE" --type=merge --patch '{"spec":{"strategy":{"type":"Recreate","rollingUpdate":null}}}' >/dev/null
kubectl patch deployment/signalops-gateway -n "$APP_NAMESPACE" --type=merge --patch '{"spec":{"template":{"metadata":{"annotations":{"vault.hashicorp.com/agent-requests-cpu":"25m","vault.hashicorp.com/agent-requests-mem":"32Mi","vault.hashicorp.com/agent-limits-cpu":"200m","vault.hashicorp.com/agent-limits-mem":"128Mi"}}}}}' >/dev/null
kubectl apply -k "$MESH_MANIFEST_DIR" >/dev/null
app_runtime_env="${SIGNALOPS_K8S_APP_RUNTIME_ENV_FILE:-/tmp/signalops-openbao-app-runtime-staging.env}"
scripts/create_k8s_signalops_app_runtime_staging_env.sh "$DOTENV_PATH" "$app_runtime_env" >/dev/null
scripts/provision_openbao_signalops_app_runtime_staging.sh "$app_runtime_env" >/dev/null
scripts/bootstrap_k8s_staging_enrollment_schema.sh >/dev/null 2>&1
DATA_NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_DATA_NAMESPACE:-signalops-data}"
DATA_POD="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_POD:-marketops-postgres-staging-0}"
DATA_SECRET="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_SECRET:-marketops-postgres-staging-auth}"
DATA_DB="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_DB:-marketops}"
DATA_USER="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_USER:-signalops}"
checkout_ref="subcheckout-${RUN_ID}"
data_password="$(kubectl get secret "$DATA_SECRET" -n "$DATA_NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
kubectl exec -i -n "$DATA_NAMESPACE" "$DATA_POD" -- env PGPASSWORD="$data_password" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$DATA_USER" -d "$DATA_DB" <<SQL >/dev/null
INSERT INTO subscriber_checkout_sessions
  (checkout_ref, tenant_id, subject, product_key, billing_period, stripe_price_id, stripe_session_id, status, checkout_url_returned, actor_subject, correlation_id)
VALUES
  ('$checkout_ref', 'tenant-local', 'k8s-stripe-webhook-route-parity-subject', 'explorer', 'monthly', 'price_k8s_route_parity', 'cs_$RUN_ID', 'checkout_started', true, 'k8s-stripe-webhook-route-parity', '$RUN_ID')
ON CONFLICT (checkout_ref) DO UPDATE SET
  stripe_session_id=EXCLUDED.stripe_session_id,
  status=EXCLUDED.status,
  updated_at=now();
SQL
kubectl wait --for=condition=Ready certificate/signalops-staging-tls -n "$APP_NAMESPACE" --timeout=120s >/dev/null
scripts/provision_k8s_signalops_staging_https_listener.sh >/dev/null
kubectl scale deployment/signalops-gateway -n "$APP_NAMESPACE" --replicas=1 >/dev/null
kubectl rollout status deployment/signalops-gateway -n "$APP_NAMESPACE" --timeout=120s >/dev/null

route_accepted="$(kubectl get httproute signalops-staging-route -n "$APP_NAMESPACE" -o jsonpath='{.status.parents[?(@.parentRef.sectionName=="signalops-staging-https")].conditions[?(@.type=="Accepted")].status}' 2>/dev/null || true)"
route_refs="$(kubectl get httproute signalops-staging-route -n "$APP_NAMESPACE" -o jsonpath='{.status.parents[?(@.parentRef.sectionName=="signalops-staging-https")].conditions[?(@.type=="ResolvedRefs")].status}' 2>/dev/null || true)"
[[ "$route_accepted" == "True" ]] || fail "HTTPRoute was not accepted by the Istio HTTPS Gateway listener"
[[ "$route_refs" == "True" ]] || fail "HTTPRoute HTTPS backend references were not resolved"

gateway_ip="$(kubectl get gateway "$GATEWAY_NAME" -n "$GATEWAY_NAMESPACE" -o jsonpath='{.status.addresses[0].value}' 2>/dev/null || true)"
if [[ -z "$gateway_ip" ]]; then
  gateway_ip="$(kubectl get service "$GATEWAY_SERVICE" -n "$GATEWAY_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
fi
[[ -n "$gateway_ip" ]] || fail "could not resolve Istio Gateway address"

workdir="$(mktemp -d)"
body_file="$workdir/body.json"
valid_header_file="$workdir/valid_header.txt"
invalid_response="$workdir/invalid_response.json"
valid_response="$workdir/valid_response.json"
trap 'cleanup; rm -rf "$workdir"' EXIT

python3 - "$RUN_ID" "$STRIPE_WEBHOOK_SECRET" "$body_file" "$valid_header_file" <<'PYSIGN'
import hashlib, hmac, json, sys, time
run_id, secret, body_path, header_path = sys.argv[1:]
ts = int(time.time())
safe = run_id.replace('-', '_')
event = {
    "id": f"evt_{safe}",
    "type": "checkout.session.completed",
    "data": {
        "object": {
            "id": f"cs_{safe}",
            "customer": f"cus_{safe}",
            "subscription": f"sub_{safe}",
            "status": "complete",
            "payment_status": "paid",
            "metadata": {"checkout_ref": f"subcheckout-{run_id}", "k8s_route_parity": run_id, "product_key": "explorer", "billing_period": "monthly"},
        }
    },
}
body = json.dumps(event, separators=(",", ":"), sort_keys=True).encode()
sig = hmac.new(secret.encode(), f"{ts}.".encode() + body, hashlib.sha256).hexdigest()
open(body_path, "wb").write(body)
open(header_path, "w", encoding="utf-8").write(f"t={ts},v1={sig}")
PYSIGN

invalid_status="$(curl --insecure -sS --resolve "${HOSTNAME}:443:${gateway_ip}" -o "$invalid_response" -w '%{http_code}' \
  -X POST "https://${HOSTNAME}/v1/billing/stripe/webhook" \
  -H 'Content-Type: application/json' \
  -H 'Stripe-Signature: t=1,v1=invalid' \
  --data-binary "@$body_file")"
[[ "$invalid_status" == "400" ]] || fail "invalid signature was not rejected through K3s route: status=$invalid_status"
grep -q 'invalid_stripe_signature' "$invalid_response" || fail "invalid signature response missing expected error"

valid_header="$(cat "$valid_header_file")"
valid_status="$(curl --insecure -sS --resolve "${HOSTNAME}:443:${gateway_ip}" -o "$valid_response" -w '%{http_code}' \
  -X POST "https://${HOSTNAME}/v1/billing/stripe/webhook" \
  -H 'Content-Type: application/json' \
  -H "Stripe-Signature: ${valid_header}" \
  --data-binary "@$body_file")"
[[ "$valid_status" == "200" ]] || { cat "$valid_response" >&2; fail "valid signed webhook failed through K3s route: status=$valid_status"; }
grep -q 'checkout.session.completed' "$valid_response" || fail "valid webhook response missing event type"

cleanup
trap - EXIT
rm -rf "$workdir"

cat <<EOF
signalops_k8s_stripe_webhook_route_parity_verified
namespace=${APP_NAMESPACE}
staging_hostname=${HOSTNAME}
gateway=${GATEWAY_NAMESPACE}/${GATEWAY_NAME}
gateway_ip=${gateway_ip}
invalid_signature_rejected=true
valid_signature_processed=true
event_type=checkout.session.completed
run_id=${RUN_ID}
stripe_provider_called=false
production_dns_changed=false
production_traffic_moved=false
production_cutover_allowed=false
staging_self_signed_tls_allowed=true
scaled_back_to_zero=true
EOF
