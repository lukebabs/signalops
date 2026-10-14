#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
IMAGE_NAME="${SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE:-ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner}"
GHCR_USER="${GHCR_USER:-}"

fail() {
  echo "signalops_k8s_marketops_job_runner_publish_failed: $*" >&2
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
local_tag="signalops-marketops-k8s-job-runner:${source_tag}"

printf '%s' "$GHCR_KEY" | docker login ghcr.io -u "$GHCR_USER" --password-stdin >/dev/null
docker build --no-cache --target marketops-k8s-job-runner -t "$local_tag" . >/dev/null
docker tag "$local_tag" "${IMAGE_NAME}:${source_tag}"
docker tag "$local_tag" "${IMAGE_NAME}:staging"
docker push "${IMAGE_NAME}:${source_tag}" >/dev/null
docker push "${IMAGE_NAME}:staging" >/dev/null
docker manifest inspect "${IMAGE_NAME}:${source_tag}" >/dev/null
docker manifest inspect "${IMAGE_NAME}:staging" >/dev/null

cat <<EOF
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=${GHCR_USER}
image=${IMAGE_NAME}
source_tag=${source_tag}
staging_tag=staging
permission=write
EOF
