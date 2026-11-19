#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
GHCR_USER="${GHCR_USER:-}"

fail() {
  echo "signalops_ghcr_pull_token_failed: $*" >&2
  exit 1
}

[[ -f "$ENV_FILE" ]] || fail "env file not found: ${ENV_FILE}"
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

[[ -n "${GHCR_KEY:-}" ]] || fail "GHCR_KEY is required"

if [[ -z "$GHCR_USER" ]]; then
  GHCR_USER="$(python3 - <<'PY'
import json, os, urllib.request, sys
req = urllib.request.Request(
    "https://api.github.com/user",
    headers={"Authorization": "Bearer " + os.environ["GHCR_KEY"], "Accept": "application/vnd.github+json"},
)
try:
    with urllib.request.urlopen(req, timeout=20) as r:
        login = json.load(r).get("login", "")
except Exception as exc:
    print(f"github_user_lookup_failed:{exc.__class__.__name__}", file=sys.stderr)
    sys.exit(1)
if not login:
    print("github_user_lookup_failed:missing_login", file=sys.stderr)
    sys.exit(1)
print(login)
PY
)" || fail "could not identify GHCR token owner"
fi

printf '%s' "$GHCR_KEY" | docker login ghcr.io -u "$GHCR_USER" --password-stdin >/dev/null

docker manifest inspect ghcr.io/syncratic-inc/signalops-web:staging >/dev/null || fail "token cannot read ghcr.io/syncratic-inc/signalops-web:staging"
docker manifest inspect ghcr.io/syncratic-inc/signalops-gateway:staging >/dev/null || fail "token cannot read ghcr.io/syncratic-inc/signalops-gateway:staging"

cat <<EOF
signalops_ghcr_pull_token_verified
registry=ghcr.io
user=${GHCR_USER}
packages=signalops-web,signalops-gateway
permission=read
EOF
