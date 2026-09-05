#!/usr/bin/env bash
set -euo pipefail

# Stripe webhook canary for SignalOps subscription billing.
#
# Default mode is non-mutating: send an invalidly signed event and require the
# gateway to reject it before persistence. Use --allow-persistent-ledger to send
# a valid signed event; that creates or reuses one webhook ledger row.

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

base_url="${SIGNALOPS_STRIPE_CANARY_BASE_URL:-https://signalops.syncratic.io}"
event_type="${SIGNALOPS_STRIPE_CANARY_EVENT_TYPE:-customer.subscription.updated}"
subscription_id="${SIGNALOPS_STRIPE_CANARY_SUBSCRIPTION_ID:-sub_signalops_unmatched_canary}"
customer_id="${SIGNALOPS_STRIPE_CANARY_CUSTOMER_ID:-cus_signalops_canary}"
allow_persistent="false"

for arg in "$@"; do
  case "$arg" in
    --allow-persistent-ledger) allow_persistent="true" ;;
    *) echo "Unknown argument: $arg" >&2; exit 2 ;;
  esac
done

if [[ "$allow_persistent" == "true" ]]; then
  : "${STRIPE_WEBHOOK_SECRET:?STRIPE_WEBHOOK_SECRET must be configured for a valid signed webhook canary}"
fi

python3 - "$base_url" "$event_type" "$subscription_id" "$customer_id" "$allow_persistent" "${STRIPE_WEBHOOK_SECRET:-}" <<'PY'
import hashlib
import hmac
import json
import sys
import time
import urllib.error
import urllib.request

base_url, event_type, subscription_id, customer_id, allow_persistent, secret = sys.argv[1:]
timestamp = int(time.time())
event_id = f"evt_signalops_canary_{timestamp}"
payload = {
    "id": event_id,
    "type": event_type,
    "data": {
        "object": {
            "id": subscription_id,
            "customer": customer_id,
            "status": "active",
            "current_period_end": timestamp + 30 * 24 * 60 * 60,
        }
    },
}
body = json.dumps(payload, separators=(",", ":"), sort_keys=True).encode()
if allow_persistent == "true":
    signature = hmac.new(secret.encode(), f"{timestamp}.".encode() + body, hashlib.sha256).hexdigest()
else:
    signature = "invalid"
headers = {
    "Content-Type": "application/json",
    "Stripe-Signature": f"t={timestamp},v1={signature}",
}
request = urllib.request.Request(f"{base_url.rstrip('/')}/v1/billing/stripe/webhook", data=body, headers=headers, method="POST")
try:
    with urllib.request.urlopen(request, timeout=20) as response:
        response_body = response.read().decode()
        status = response.status
except urllib.error.HTTPError as exc:
    response_body = exc.read().decode()
    status = exc.code

if allow_persistent == "true":
    if status != 200:
        raise SystemExit(f"valid Stripe webhook canary failed: status={status} body={response_body}")
    result = json.loads(response_body)
    if result.get("provider_event_id") != event_id or result.get("event_type") != event_type:
        raise SystemExit(f"unexpected Stripe webhook canary response: {response_body}")
    print(json.dumps({"status": result.get("status"), "provider_event_id": result.get("provider_event_id"), "event_type": result.get("event_type")}, sort_keys=True))
else:
    if status == 503 and "stripe_webhook_disabled" in response_body:
        print(json.dumps({"status": "stripe_webhook_disabled", "persistent_ledger_write": False}, sort_keys=True))
    elif status != 400 or "invalid_stripe_signature" not in response_body:
        raise SystemExit(f"invalid-signature canary did not fail closed: status={status} body={response_body}")
    else:
        print(json.dumps({"status": "rejected_invalid_signature", "persistent_ledger_write": False}, sort_keys=True))
PY
