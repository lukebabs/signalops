#!/usr/bin/env bash
# Controlled Stripe Checkout startup canary. Creates Checkout Sessions only; it
# does not complete payment or grant subscription access from the redirect.
set -Eeuo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dotenv_path="${SIGNALOPS_E2E_ENV_FILE:-$repo_dir/.env}"
artifact_dir="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-e2e-artifacts}"
refs_file="$artifact_dir/stripe-checkout-canary-refs.json"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$dotenv_path"

: "${SIGNALOPS_WEB:?SIGNALOPS_WEB must identify the tenant-pilot-b QA account}"
: "${SIGNALOPS_WEB_PASS:?SIGNALOPS_WEB_PASS must be set for the tenant-pilot-b QA account}"

mkdir -p "$artifact_dir"
chmod 700 "$artifact_dir"
rm -f "$refs_file"

export SIGNALOPS_STRIPE_CHECKOUT_CANARY=1
export SIGNALOPS_E2E_USERNAME="$SIGNALOPS_WEB"
export SIGNALOPS_E2E_PASSWORD="$SIGNALOPS_WEB_PASS"
export SIGNALOPS_E2E_WATCHLIST_NAME="${SIGNALOPS_E2E_WATCHLIST_NAME:-First List}"
export SIGNALOPS_E2E_TENANT_ID="${SIGNALOPS_E2E_TENANT_ID:-tenant-pilot-b}"
export SIGNALOPS_E2E_SHARED_TICKERS="${SIGNALOPS_E2E_SHARED_TICKERS:-AAPL,NVDA}"
export SIGNALOPS_E2E_PENDING_TICKERS=""
export SIGNALOPS_E2E_ARTIFACT_DIR="$artifact_dir"

"$repo_dir/.venv/bin/python" -m pytest -q "$repo_dir/python/tests/test_stripe_checkout_canary_ui.py"

[[ -s "$refs_file" ]] || { printf '%s\n' 'stripe_checkout_canary_refs_missing' >&2; exit 1; }

mapfile -t refs < <("$repo_dir/.venv/bin/python" - "$refs_file" <<'PY'
import json, re, sys
items = json.load(open(sys.argv[1]))
for item in items:
    ref = item.get('checkout_ref', '')
    if not re.fullmatch(r'subcheckout-[A-Za-z0-9_]+', ref):
        raise SystemExit(f'invalid checkout_ref {ref!r}')
    print(ref)
PY
)

if [[ "${#refs[@]}" -lt 2 ]]; then
  printf '%s\n' 'stripe_checkout_canary_ref_count_invalid' >&2
  exit 1
fi

sql_in=""
for ref in "${refs[@]}"; do
  if [[ -n "$sql_in" ]]; then sql_in+=","; fi
  sql_in+="'$ref'"
done

verification="$(docker exec signalops-marketops-postgres-1 psql -U signalops -d marketops -Atc "
SELECT count(*)
FROM subscriber_checkout_sessions
WHERE checkout_ref IN ($sql_in)
  AND tenant_id = '${SIGNALOPS_E2E_TENANT_ID}'
  AND status = 'checkout_started'
  AND checkout_url_returned = true
  AND stripe_session_id <> ''
  AND stripe_subscription_id = '';
")"

if [[ "$verification" != "${#refs[@]}" ]]; then
  printf 'stripe_checkout_canary_ledger_mismatch expected=%s actual=%s\n' "${#refs[@]}" "$verification" >&2
  exit 1
fi

printf 'stripe_checkout_canary_verified sessions=%s refs=%s\n' "${#refs[@]}" "$(IFS=,; echo "${refs[*]}")"
