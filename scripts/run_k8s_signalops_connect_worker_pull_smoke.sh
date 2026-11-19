#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
NAMESPACE="${SIGNALOPS_K8S_CONNECT_NAMESPACE:-signalops-connect}"
IMAGE="${SIGNALOPS_CONNECT_K8S_WORKER_IMAGE:-ghcr.io/syncratic-inc/signalops-connect-k8s-worker:staging}"
POD_NAME="${SIGNALOPS_K8S_CONNECT_PULL_SMOKE_POD:-signalops-connect-worker-pull-smoke}"
GHCR_USER="${GHCR_USER:-}"

fail() {
  echo "signalops_k8s_connect_worker_pull_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
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

kubectl get namespace "$NAMESPACE" >/dev/null || fail "namespace missing: ${NAMESPACE}"
printf '%s' "$GHCR_KEY" | docker login ghcr.io -u "$GHCR_USER" --password-stdin >/dev/null
docker manifest inspect "$IMAGE" >/dev/null || fail "token cannot read ${IMAGE}"

kubectl create secret docker-registry signalops-ghcr-pull \
  --namespace "$NAMESPACE" \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USER" \
  --docker-password="$GHCR_KEY" \
  --dry-run=client -o yaml | kubectl apply -f - >/dev/null

kubectl delete pod "$POD_NAME" -n "$NAMESPACE" --ignore-not-found --wait=true >/dev/null
smoke_yaml="$(mktemp -t signalops-connect-pull-smoke.XXXXXX.yaml)"
trap 'kubectl delete pod "$POD_NAME" -n "$NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true; rm -f "$smoke_yaml"' EXIT
cat >"$smoke_yaml" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: signalops-connect-k8s-worker
    app.kubernetes.io/component: image-pull-smoke
    app.kubernetes.io/part-of: signalops
    signalops.syncratic.io/plane: connect
    signalops.syncratic.io/stage: staging
    signalops.syncratic.io/production-cutover-allowed: "false"
spec:
  restartPolicy: Never
  serviceAccountName: signalops-connect-worker
  imagePullSecrets:
    - name: signalops-ghcr-pull
  containers:
    - name: pull-smoke
      image: ${IMAGE}
      imagePullPolicy: Always
      command:
        - /bin/sh
        - -ec
        - echo signalops_connect_worker_image_pulled
      resources:
        requests:
          cpu: 10m
          memory: 32Mi
        limits:
          cpu: 50m
          memory: 64Mi
EOF
kubectl apply -f "$smoke_yaml" >/dev/null
for _ in $(seq 1 120); do
  phase="$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  case "$phase" in
    Succeeded) break ;;
    Failed) kubectl describe pod "$POD_NAME" -n "$NAMESPACE" >&2 || true; fail "pull smoke pod failed" ;;
  esac
  sleep 1
done
phase="$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
[[ "$phase" == "Succeeded" ]] || { kubectl describe pod "$POD_NAME" -n "$NAMESPACE" >&2 || true; fail "pull smoke pod did not succeed; phase=${phase:-missing}"; }
logs="$(kubectl logs -n "$NAMESPACE" "$POD_NAME")"
[[ "$logs" == *"signalops_connect_worker_image_pulled"* ]] || fail "pull smoke log marker missing"
kubectl delete pod "$POD_NAME" -n "$NAMESPACE" --wait=true >/dev/null
trap - EXIT
rm -f "$smoke_yaml"

cat <<EOF
signalops_k8s_connect_worker_pull_smoke_verified
namespace=${NAMESPACE}
registry=ghcr.io
image=${IMAGE}
secret=signalops-ghcr-pull
pod_succeeded=true
worker_executed=false
provider_polling=false
production_cutover_allowed=false
EOF
