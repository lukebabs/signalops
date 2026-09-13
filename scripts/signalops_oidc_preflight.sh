#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

for required in SIGNALOPS_AUTH_ISSUER SIGNALOPS_AUTH_JWKS_URL SIGNALOPS_AUTH_AUDIENCE; do
  if [[ -z "${!required:-}" ]]; then
    printf 'Missing %s. This preflight requires the same explicit OIDC configuration as an auth-enabled gateway.\n' "$required" >&2
    exit 2
  fi
done

for dependency in curl jq; do
  if ! command -v "$dependency" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$dependency" >&2
    exit 2
  fi
done

issuer="${SIGNALOPS_AUTH_ISSUER%/}"
expected_jwks="$SIGNALOPS_AUTH_JWKS_URL"
discovery_url="${issuer}/.well-known/openid-configuration"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

printf 'OIDC preflight: issuer=%s audience=%s\n' "$issuer" "$SIGNALOPS_AUTH_AUDIENCE"
printf 'Checking discovery document: %s\n' "$discovery_url"
curl --fail --location --silent --show-error --connect-timeout 10 --max-time 20 "$discovery_url" -o "$work_dir/discovery.json"

if ! jq -e --arg issuer "$issuer" --arg jwks "$expected_jwks" '.issuer == $issuer and .jwks_uri == $jwks and (.authorization_endpoint | type == "string") and (.token_endpoint | type == "string")' "$work_dir/discovery.json" >/dev/null; then
  printf 'OIDC discovery does not match the configured issuer/JWKS URL or is missing required endpoints.\n' >&2
  exit 3
fi

printf 'Checking JWKS document: %s\n' "$expected_jwks"
curl --fail --location --silent --show-error --connect-timeout 10 --max-time 20 "$expected_jwks" -o "$work_dir/jwks.json"

if ! jq -e '.keys | type == "array" and length > 0 and any(.[]; .kty == "RSA" and (.kid | type == "string") and (.kid | length > 0))' "$work_dir/jwks.json" >/dev/null; then
  printf 'JWKS has no usable RSA key with a key ID; SignalOps only accepts RS256 JWTs.\n' >&2
  exit 4
fi

printf 'OIDC discovery/JWKS preflight passed. Run the real-browser checklist before enabling auth in production.\n'
