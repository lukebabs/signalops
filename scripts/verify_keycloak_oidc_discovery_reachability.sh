#!/usr/bin/env bash
set -euo pipefail

ISSUER="${SIGNALOPS_AUTH_ISSUER:-https://auth.syncratic.co/realms/syncratic}"
DISCOVERY_URL="${ISSUER%/}/.well-known/openid-configuration"

fail() {
  echo "signalops_keycloak_oidc_discovery_reachability_failed: $*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"

tmp_body="$(mktemp -t signalops-oidc-discovery.XXXXXX.json)"
tmp_headers="$(mktemp -t signalops-oidc-discovery.XXXXXX.headers)"
cleanup() {
  rm -f "$tmp_body" "$tmp_headers"
}
trap cleanup EXIT

status="$(curl -sS -L -D "$tmp_headers" -o "$tmp_body" -w '%{http_code}' "$DISCOVERY_URL" || true)"
[[ "$status" == "200" ]] || fail "OIDC discovery returned HTTP ${status}; expected 200 for ${DISCOVERY_URL}"

content_type="$(grep -i '^content-type:' "$tmp_headers" | tail -1 | tr -d '\r' || true)"
case "$content_type" in
  *application/json*) ;;
  *) fail "OIDC discovery returned non-JSON content type: ${content_type:-missing}" ;;
esac

if grep -qiE 'request unsuccessful|incident_id|captcha|incapsula incident' "$tmp_body"; then
  fail "OIDC discovery returned a WAF/challenge page instead of JSON metadata"
fi

grep -q '"authorization_endpoint"' "$tmp_body" || fail "OIDC discovery metadata missing authorization_endpoint"
grep -q '"jwks_uri"' "$tmp_body" || fail "OIDC discovery metadata missing jwks_uri"

jwks_uri="$(python3 - "$tmp_body" <<'PYJSON'
import json, sys
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    print(json.load(f).get('jwks_uri', ''))
PYJSON
)"
[[ -n "$jwks_uri" ]] || fail "jwks_uri could not be parsed"

jwks_status="$(curl -sS -L -o /dev/null -w '%{http_code}' "$jwks_uri" || true)"
[[ "$jwks_status" == "200" ]] || fail "JWKS returned HTTP ${jwks_status}; expected 200 for ${jwks_uri}"

cat <<EOF
signalops_keycloak_oidc_discovery_reachability_verified
issuer=${ISSUER%/}
discovery_status=${status}
jwks_status=${jwks_status}
waf_challenge=false
production_cutover_allowed=false
EOF
