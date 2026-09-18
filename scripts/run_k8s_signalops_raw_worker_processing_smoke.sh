#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${SIGNALOPS_K8S_CONNECT_NAMESPACE:-signalops-connect}"
DATA_NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
BROKER_DEPLOYMENT="${SIGNALOPS_K8S_CONNECT_BROKER_DEPLOYMENT:-signalops-redpanda-staging}"
RAW_WORKER_DEPLOYMENT="${SIGNALOPS_K8S_RAW_WORKER_DEPLOYMENT:-signalops-raw-worker}"
ENVIRONMENT="${SIGNALOPS_K8S_RAW_WORKER_ENVIRONMENT:-kubernetes-staging}"
RUN_ID="${SIGNALOPS_K8S_RAW_WORKER_SMOKE_RUN_ID:-raw-worker-smoke-$(date -u +%Y%m%dT%H%M%SZ)}"
TIMEOUT_SECONDS="${SIGNALOPS_K8S_RAW_WORKER_TIMEOUT_SECONDS:-120}"

fail() {
  echo "signalops_k8s_raw_worker_processing_smoke_failed: $*" >&2
  cleanup
  exit 1
}

cleanup() {
  kubectl scale deployment "$RAW_WORKER_DEPLOYMENT" -n "$NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
}

command -v kubectl >/dev/null 2>&1 || { echo "signalops_k8s_raw_worker_processing_smoke_failed: kubectl is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "signalops_k8s_raw_worker_processing_smoke_failed: python3 is required" >&2; exit 1; }
trap cleanup EXIT

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

scripts/run_k8s_signalops_connect_broker_smoke.sh >/dev/null
kubectl get deployment "$RAW_WORKER_DEPLOYMENT" -n "$NAMESPACE" >/dev/null || fail "raw-worker deployment missing: ${NAMESPACE}/${RAW_WORKER_DEPLOYMENT}"
kubectl rollout status "deployment/${BROKER_DEPLOYMENT}" -n "$DATA_NAMESPACE" --timeout=120s >/dev/null || fail "broker not ready"

normalized_topic="signalops.${ENVIRONMENT}.normalized.v1"
signal_topic="signalops.${ENVIRONMENT}.signal.v1"
message_key="${RUN_ID}-aapl"

payload_file="$(mktemp)"
trap 'cleanup; rm -f "$payload_file"' EXIT
python3 - "$RUN_ID" >"$payload_file" <<'PYEVENT'
import json, sys
run_id = sys.argv[1]
event_id = f"evt-{run_id}-aapl"
now = "2026-09-13T00:00:00Z"
payload = {
    "tenant_id": "tenant-local",
    "source_id": "src-massive-k8s-raw-worker-smoke",
    "app_id": "marketops",
    "domain": "market_data",
    "use_case": "daily_market_surveillance",
    "source_domain": "market_data",
    "source_adapter": "market_data.massive",
    "ingestion_mode": "scheduled_pull",
    "dataset": "equity_eod_prices",
    "event_id": event_id,
    "event_type": "marketops.equity_eod_price.normalized",
    "schema_id": "signalops.normalized_signal_event.v1",
    "schema_version": "v1",
    "observation_time": now,
    "effective_time": now,
    "processing_time": now,
    "occurred_at": now,
    "observed_at": now,
    "normalized_payload": {
        "symbol": "AAPL",
        "observation_date": "2026-09-13",
        "open": 100.0,
        "high": 104.0,
        "low": 99.0,
        "close": 103.1,
        "volume": 2500000,
        "vwap": 101.0,
        "previous_close": 100.0,
    },
    "entities": [{"type": "ticker", "id": "ticker:AAPL", "external_id": "AAPL"}],
    "confidence": 1.0,
    "metadata": {"quality": {"status": "synthetic_smoke"}, "run_id": run_id},
    "evidence": [{"type": "k8s_smoke", "id": run_id, "description": "bounded non-production raw-worker processing smoke"}],
    "correlation_id": run_id,
    "idempotency_key": event_id,
    "trace_id": run_id,
    "causation_id": event_id,
}
print(json.dumps(payload, separators=(",", ":")))
PYEVENT

kubectl exec -i -n "$DATA_NAMESPACE" "deploy/${BROKER_DEPLOYMENT}" -- \
  rpk topic produce "$normalized_topic" --key "$message_key" <"$payload_file" >/dev/null

kubectl scale deployment "$RAW_WORKER_DEPLOYMENT" -n "$NAMESPACE" --replicas=1 >/dev/null
for _ in $(seq 1 "$TIMEOUT_SECONDS"); do
  pod="$(kubectl get pod -n "$NAMESPACE" -l app.kubernetes.io/name=signalops-raw-worker -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  if [[ -n "$pod" ]]; then
    phase="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    if [[ "$phase" == "Succeeded" || "$phase" == "Running" ]]; then
      logs="$(kubectl logs -n "$NAMESPACE" "$pod" --tail=200 2>/dev/null || true)"
      if grep -q 'worker stopped' <<<"$logs"; then
        break
      fi
      if grep -Eiq 'Traceback|KafkaException|ERROR|CRITICAL' <<<"$logs"; then
        printf '%s\n' "$logs" >&2
        fail "raw-worker emitted error log marker"
      fi
    fi
  fi
  sleep 1
done

pod="$(kubectl get pod -n "$NAMESPACE" -l app.kubernetes.io/name=signalops-raw-worker -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
[[ -n "$pod" ]] || fail "raw-worker pod was not created"
logs="$(kubectl logs -n "$NAMESPACE" "$pod" --tail=200 2>/dev/null || true)"
grep -q 'worker stopped' <<<"$logs" || { printf '%s\n' "$logs" >&2; fail "raw-worker did not stop after one message"; }

signal_output="$(kubectl exec -n "$DATA_NAMESPACE" "deploy/${BROKER_DEPLOYMENT}" -- timeout 20 rpk topic consume "$signal_topic" --num 1 --format '%k %v\n' 2>/dev/null || true)"
grep -q 'marketops.dsm.accumulation' <<<"$signal_output" || { printf '%s\n' "$signal_output" >&2; fail "expected accumulation signal was not observed"; }
grep -q "$RUN_ID" <<<"$signal_output" || { printf '%s\n' "$signal_output" >&2; fail "expected smoke correlation id was not observed"; }

cleanup
trap - EXIT
rm -f "$payload_file"
kubectl rollout status "deployment/${RAW_WORKER_DEPLOYMENT}" -n "$NAMESPACE" --timeout=120s >/dev/null

cat <<EOF
signalops_k8s_raw_worker_processing_smoke_verified
namespace=${NAMESPACE}
deployment=${RAW_WORKER_DEPLOYMENT}
run_id=${RUN_ID}
input_topic=${normalized_topic}
signal_topic=${signal_topic}
signal_type=marketops.dsm.accumulation
messages_processed=1
replicas_restored_to_zero=true
provider_polling=false
production_cutover_allowed=false
EOF
