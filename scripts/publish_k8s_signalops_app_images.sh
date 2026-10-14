#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
GATEWAY_IMAGE="${SIGNALOPS_K8S_GATEWAY_IMAGE:-ghcr.io/syncratic-inc/signalops-gateway}"
WEB_IMAGE="${SIGNALOPS_K8S_WEB_IMAGE:-ghcr.io/syncratic-inc/signalops-web}"
GHCR_USER="${GHCR_USER:-}"

fail() {
  echo "signalops_k8s_app_image_publish_failed: $*" >&2
  exit 1
}

[[ -f "$ENV_FILE" ]] || fail "env file not found: ${ENV_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"

[[ -n "${GHCR_KEY:-}" ]] || fail "GHCR_KEY is required"

if [[ -z "$GHCR_USER" ]]; then
  GHCR_USER="$(python3 - <<'PYGH'
import json, os, sys, urllib.request
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
PYGH
)" || fail "could not identify GHCR token owner"
fi

source_tag="$(git rev-parse --short=12 HEAD)"
printf '%s' "$GHCR_KEY" | docker login ghcr.io -u "$GHCR_USER" --password-stdin >/dev/null

gateway_local="signalops-gateway:${source_tag}"
web_local="signalops-web:${source_tag}"

docker build --target gateway -t "$gateway_local" . >/dev/null
docker build   --build-arg VITE_SIGNALOPS_AUTH_ENABLED="${VITE_SIGNALOPS_AUTH_ENABLED:-true}"   --build-arg VITE_SIGNALOPS_AUTH_ISSUER="${VITE_SIGNALOPS_AUTH_ISSUER:-https://auth.syncratic.co/realms/syncratic}"   --build-arg VITE_SIGNALOPS_AUTH_REALM="${VITE_SIGNALOPS_AUTH_REALM:-syncratic}"   --build-arg VITE_SIGNALOPS_AUTH_CLIENT_ID="${VITE_SIGNALOPS_AUTH_CLIENT_ID:-signalops-web}"   --build-arg VITE_SIGNALOPS_AUTH_AUDIENCE="${VITE_SIGNALOPS_AUTH_AUDIENCE:-signalops-api}"   --build-arg VITE_SIGNALOPS_AUTH_SIGNUP_URL="${VITE_SIGNALOPS_AUTH_SIGNUP_URL:-}"   --build-arg VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES="${VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES:-30}"   --build-arg VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS="${VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS:-60}"   --target web -t "$web_local" ./web >/dev/null

docker tag "$gateway_local" "${GATEWAY_IMAGE}:${source_tag}"
docker tag "$gateway_local" "${GATEWAY_IMAGE}:production"
docker tag "$web_local" "${WEB_IMAGE}:${source_tag}"
docker tag "$web_local" "${WEB_IMAGE}:production"

docker push "${GATEWAY_IMAGE}:${source_tag}" >/dev/null
docker push "${GATEWAY_IMAGE}:production" >/dev/null
docker push "${WEB_IMAGE}:${source_tag}" >/dev/null
docker push "${WEB_IMAGE}:production" >/dev/null

docker manifest inspect "${GATEWAY_IMAGE}:production" >/dev/null
docker manifest inspect "${WEB_IMAGE}:production" >/dev/null

cat <<EOF
signalops_k8s_app_image_publish_verified
registry=ghcr.io
user=${GHCR_USER}
gateway_image=${GATEWAY_IMAGE}
web_image=${WEB_IMAGE}
source_tag=${source_tag}
production_tag=production
permission=write
EOF
