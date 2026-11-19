#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
REGISTRY="${SIGNALOPS_K8S_REGISTRY:-ghcr.io}"
IMAGE_OWNER="${SIGNALOPS_K8S_IMAGE_OWNER:-syncratic-inc}"
POSTGRES_IMAGE="${SIGNALOPS_K8S_POSTGRES_IMAGE:-${REGISTRY}/${IMAGE_OWNER}/signalops-postgres-pgbackrest}"
TIMESCALE_IMAGE="${SIGNALOPS_K8S_TIMESCALE_IMAGE:-${REGISTRY}/${IMAGE_OWNER}/signalops-marketops-timescaledb-pgbackrest}"
SOURCE_TAG="${SIGNALOPS_K8S_IMAGE_SOURCE_TAG:-$(git rev-parse --short=12 HEAD)}"
PUBLISH_TAG="${SIGNALOPS_K8S_IMAGE_PUBLISH_TAG:-production}"

fail() {
  echo "signalops_k8s_database_image_publish_failed: $*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"
# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"

command -v docker >/dev/null 2>&1 || fail "docker is required"
[[ -n "${GHCR_KEY:-}" ]] || fail "GHCR_KEY is required in ${ENV_FILE}"
GHCR_USER="${GHCR_USER:-${GITHUB_ACTOR:-lukebabs}}"

printf '%s' "$GHCR_KEY" | docker login "$REGISTRY" -u "$GHCR_USER" --password-stdin >/dev/null

docker build -f deploy/postgres/pgbackrest.Dockerfile -t "${POSTGRES_IMAGE}:${SOURCE_TAG}" . >/dev/null
docker build -f deploy/postgres/marketops-timescaledb-pgbackrest.Dockerfile -t "${TIMESCALE_IMAGE}:${SOURCE_TAG}" . >/dev/null

docker tag "${POSTGRES_IMAGE}:${SOURCE_TAG}" "${POSTGRES_IMAGE}:${PUBLISH_TAG}"
docker tag "${TIMESCALE_IMAGE}:${SOURCE_TAG}" "${TIMESCALE_IMAGE}:${PUBLISH_TAG}"

docker push "${POSTGRES_IMAGE}:${SOURCE_TAG}" >/dev/null
docker push "${POSTGRES_IMAGE}:${PUBLISH_TAG}" >/dev/null
docker push "${TIMESCALE_IMAGE}:${SOURCE_TAG}" >/dev/null
docker push "${TIMESCALE_IMAGE}:${PUBLISH_TAG}" >/dev/null

docker manifest inspect "${POSTGRES_IMAGE}:${PUBLISH_TAG}" >/dev/null
docker manifest inspect "${TIMESCALE_IMAGE}:${PUBLISH_TAG}" >/dev/null

cat <<EOF
signalops_k8s_database_image_publish_verified
registry=${REGISTRY}
user=${GHCR_USER}
postgres_image=${POSTGRES_IMAGE}
timescale_image=${TIMESCALE_IMAGE}
source_tag=${SOURCE_TAG}
production_tag=${PUBLISH_TAG}
permission=write
EOF
