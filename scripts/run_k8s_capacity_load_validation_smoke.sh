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
RUN_ID="${SIGNALOPS_K8S_CAPACITY_RUN_ID:-k8s-capacity-$(date -u +%Y%m%dT%H%M%SZ)}"
HEALTH_REQUESTS="${SIGNALOPS_K8S_CAPACITY_HEALTH_REQUESTS:-120}"
HEALTH_CONCURRENCY="${SIGNALOPS_K8S_CAPACITY_HEALTH_CONCURRENCY:-12}"
READY_REQUESTS="${SIGNALOPS_K8S_CAPACITY_READY_REQUESTS:-80}"
READY_CONCURRENCY="${SIGNALOPS_K8S_CAPACITY_READY_CONCURRENCY:-8}"
WEBHOOK_REQUESTS="${SIGNALOPS_K8S_CAPACITY_WEBHOOK_REQUESTS:-24}"
WEBHOOK_CONCURRENCY="${SIGNALOPS_K8S_CAPACITY_WEBHOOK_CONCURRENCY:-4}"
MAX_ERROR_RATE_PCT="${SIGNALOPS_K8S_CAPACITY_MAX_ERROR_RATE_PCT:-1.0}"
MAX_HEALTH_P95_MS="${SIGNALOPS_K8S_CAPACITY_MAX_HEALTH_P95_MS:-1500}"
MAX_READY_P95_MS="${SIGNALOPS_K8S_CAPACITY_MAX_READY_P95_MS:-1500}"
MAX_WEBHOOK_P95_MS="${SIGNALOPS_K8S_CAPACITY_MAX_WEBHOOK_P95_MS:-3000}"

fail() {
  printf 'signalops_k8s_capacity_load_validation_failed: %s\n' "$*" >&2
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
: "${STRIPE_WEBHOOK_SECRET:?STRIPE_WEBHOOK_SECRET is required for signed K3s capacity webhook validation}"

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
kubectl set resources deployment/signalops-gateway -n "$APP_NAMESPACE" -c gateway --requests=cpu=0,memory=128Mi --limits=cpu=500m,memory=768Mi >/dev/null
kubectl patch deployment/signalops-gateway -n "$APP_NAMESPACE" --type=merge --patch '{"spec":{"strategy":{"type":"Recreate","rollingUpdate":null}}}' >/dev/null
kubectl patch deployment/signalops-gateway -n "$APP_NAMESPACE" --type=merge --patch '{"spec":{"template":{"metadata":{"annotations":{"vault.hashicorp.com/agent-requests-cpu":"0","vault.hashicorp.com/agent-requests-mem":"32Mi","vault.hashicorp.com/agent-limits-cpu":"200m","vault.hashicorp.com/agent-limits-mem":"128Mi"}}}}}' >/dev/null
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
data_password="$(kubectl get secret "$DATA_SECRET" -n "$DATA_NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
kubectl exec -i -n "$DATA_NAMESPACE" "$DATA_POD" -- env PGPASSWORD="$data_password" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$DATA_USER" -d "$DATA_DB" <<SQL >/dev/null
INSERT INTO subscriber_checkout_sessions
  (checkout_ref, tenant_id, subject, product_key, billing_period, stripe_price_id, stripe_session_id, status, checkout_url_returned, actor_subject, correlation_id)
SELECT
  'subcheckout-${RUN_ID}-' || gs::text,
  'tenant-local',
  'k8s-capacity-subject-' || gs::text,
  'explorer',
  'monthly',
  'price_k8s_capacity',
  'cs_${RUN_ID}_' || gs::text,
  'checkout_started',
  true,
  'k8s-capacity-load-validation',
  '${RUN_ID}'
FROM generate_series(1, ${WEBHOOK_REQUESTS}) AS gs
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

result_file="$(mktemp)"
trap 'cleanup; rm -f "$result_file"' EXIT

python3 - "$gateway_ip" "$HOSTNAME" "$STRIPE_WEBHOOK_SECRET" "$RUN_ID" "$result_file" \
  "$HEALTH_REQUESTS" "$HEALTH_CONCURRENCY" "$READY_REQUESTS" "$READY_CONCURRENCY" "$WEBHOOK_REQUESTS" "$WEBHOOK_CONCURRENCY" \
  "$MAX_ERROR_RATE_PCT" "$MAX_HEALTH_P95_MS" "$MAX_READY_P95_MS" "$MAX_WEBHOOK_P95_MS" <<'PYLOAD'
import concurrent.futures
import hashlib
import hmac
import json
import math
import subprocess
import sys
import time
from statistics import median

(
    gateway_ip,
    hostname,
    secret,
    run_id,
    result_file,
    health_requests,
    health_concurrency,
    ready_requests,
    ready_concurrency,
    webhook_requests,
    webhook_concurrency,
    max_error_rate_pct,
    max_health_p95_ms,
    max_ready_p95_ms,
    max_webhook_p95_ms,
) = sys.argv[1:]
health_requests = int(health_requests)
health_concurrency = int(health_concurrency)
ready_requests = int(ready_requests)
ready_concurrency = int(ready_concurrency)
webhook_requests = int(webhook_requests)
webhook_concurrency = int(webhook_concurrency)
thresholds = {
    "healthz": float(max_health_p95_ms),
    "readyz": float(max_ready_p95_ms),
    "stripe_webhook_checkout": float(max_webhook_p95_ms),
}
max_error_rate_pct = float(max_error_rate_pct)
def percentile(values, pct):
    if not values:
        return 0.0
    ordered = sorted(values)
    index = max(0, min(len(ordered) - 1, math.ceil((pct / 100.0) * len(ordered)) - 1))
    return ordered[index]

def request(method, path, body=None, headers=None, expected=200):
    cmd = [
        "curl",
        "--insecure",
        "--silent",
        "--show-error",
        "--resolve",
        f"{hostname}:443:{gateway_ip}",
        "--output",
        "/dev/null",
        "--write-out",
        "%{http_code}",
        "--max-time",
        "10",
        "-X",
        method,
    ]
    for key, value in (headers or {}).items():
        cmd.extend(["-H", f"{key}: {value}"])
    if body is not None:
        cmd.extend(["--data-binary", "@-"])
    cmd.append(f"https://{hostname}{path}")
    start = time.perf_counter()
    status = 0
    error = ""
    try:
        completed = subprocess.run(cmd, input=body, capture_output=True, timeout=12, check=False)
        stdout = completed.stdout.decode("utf-8", errors="replace").strip()
        stderr = completed.stderr.decode("utf-8", errors="replace").strip()
        status = int(stdout[-3:]) if len(stdout) >= 3 and stdout[-3:].isdigit() else 0
        if completed.returncode != 0:
            error = f"curl_exit={completed.returncode} stderr={stderr[:180]}"
        elif status != expected:
            error = f"unexpected_status={status}"
    except Exception as exc:  # noqa: BLE001
        error = type(exc).__name__ + ": " + str(exc)
    elapsed_ms = (time.perf_counter() - start) * 1000.0
    return {"elapsed_ms": elapsed_ms, "status": status, "ok": status == expected and not error, "error": error}

def signed_checkout(index):
    ts = int(time.time())
    safe_run = run_id.replace('-', '_')
    event = {
        "id": f"evt_{safe_run}_{index}",
        "type": "checkout.session.completed",
        "data": {
            "object": {
                "id": f"cs_{safe_run}_{index}",
                "customer": f"cus_{safe_run}_{index}",
                "subscription": f"sub_{safe_run}_{index}",
                "status": "complete",
                "payment_status": "paid",
                "metadata": {"checkout_ref": f"subcheckout-{run_id}-{index}", "k8s_capacity": run_id, "product_key": "explorer", "billing_period": "monthly"},
            }
        },
    }
    body = json.dumps(event, separators=(",", ":"), sort_keys=True).encode()
    sig = hmac.new(secret.encode(), f"{ts}.".encode() + body, hashlib.sha256).hexdigest()
    return request("POST", "/v1/billing/stripe/webhook", body=body, headers={"Content-Type": "application/json", "Stripe-Signature": f"t={ts},v1={sig}"}, expected=200)

def run_phase(name, total, concurrency, fn):
    started = time.perf_counter()
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = [pool.submit(fn, i + 1) for i in range(total)]
        for future in concurrent.futures.as_completed(futures):
            results.append(future.result())
    duration = time.perf_counter() - started
    latencies = [r["elapsed_ms"] for r in results]
    errors = [r for r in results if not r["ok"]]
    error_rate = (len(errors) / total) * 100.0 if total else 0.0
    summary = {
        "name": name,
        "requests": total,
        "concurrency": concurrency,
        "duration_seconds": round(duration, 3),
        "rps": round(total / duration, 2) if duration > 0 else 0,
        "errors": len(errors),
        "error_rate_pct": round(error_rate, 3),
        "p50_ms": round(median(latencies), 2) if latencies else 0,
        "p95_ms": round(percentile(latencies, 95), 2),
        "p99_ms": round(percentile(latencies, 99), 2),
        "max_ms": round(max(latencies), 2) if latencies else 0,
        "threshold_p95_ms": thresholds[name],
        "passed": error_rate <= max_error_rate_pct and percentile(latencies, 95) <= thresholds[name],
        "first_error": errors[0]["error"][:240] if errors else "",
    }
    return summary

summaries = [
    run_phase("healthz", health_requests, health_concurrency, lambda _: request("GET", "/healthz", expected=200)),
    run_phase("readyz", ready_requests, ready_concurrency, lambda _: request("GET", "/readyz", expected=200)),
    run_phase("stripe_webhook_checkout", webhook_requests, webhook_concurrency, signed_checkout),
]
with open(result_file, "w", encoding="utf-8") as fh:
    json.dump({"run_id": run_id, "summaries": summaries}, fh, indent=2, sort_keys=True)
failed = [s for s in summaries if not s["passed"]]
for summary in summaries:
    print(f"phase={summary['name']} requests={summary['requests']} concurrency={summary['concurrency']} errors={summary['errors']} error_rate_pct={summary['error_rate_pct']} p50_ms={summary['p50_ms']} p95_ms={summary['p95_ms']} p99_ms={summary['p99_ms']} rps={summary['rps']} passed={str(summary['passed']).lower()}")
    if summary["first_error"]:
        print(f"phase={summary['name']} first_error={summary['first_error']}")
if failed:
    raise SystemExit(2)
PYLOAD

summary_lines="$(python3 - "$result_file" <<'PYSUM'
import json, sys
with open(sys.argv[1], encoding='utf-8') as fh:
    data = json.load(fh)
for item in data["summaries"]:
    key = item["name"]
    print(f"{key}_requests={item['requests']}")
    print(f"{key}_concurrency={item['concurrency']}")
    print(f"{key}_error_rate_pct={item['error_rate_pct']}")
    print(f"{key}_p95_ms={item['p95_ms']}")
    print(f"{key}_p99_ms={item['p99_ms']}")
    print(f"{key}_rps={item['rps']}")
PYSUM
)"

cleanup
trap - EXIT
rm -f "$result_file"

cat <<EOF
signalops_k8s_capacity_load_validation_verified
namespace=${APP_NAMESPACE}
staging_hostname=${HOSTNAME}
gateway=${GATEWAY_NAMESPACE}/${GATEWAY_NAME}
gateway_ip=${gateway_ip}
run_id=${RUN_ID}
${summary_lines}
stripe_provider_called=false
production_dns_changed=false
production_traffic_moved=false
production_cutover_allowed=false
staging_self_signed_tls_allowed=true
staging_zero_cpu_reservation_used=true
scaled_back_to_zero=true
EOF
