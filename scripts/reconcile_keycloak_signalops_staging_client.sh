#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${SYNCRATIC_KEYCLOAK_NAMESPACE:-syncratic}"
REALM="${SYNCRATIC_KEYCLOAK_REALM:-syncratic}"
CLIENT_ID="${SIGNALOPS_KEYCLOAK_CLIENT_ID:-signalops-web}"
STAGING_ORIGIN="${SIGNALOPS_K8S_SIGNALOPS_STAGING_ORIGIN:-https://signalops-staging.syncratic.co}"
ADMIN_SECRET="${SYNCRATIC_KEYCLOAK_ADMIN_SECRET:-syncratic-keycloak-admin}"
POD_SELECTOR="${SYNCRATIC_KEYCLOAK_POD_SELECTOR:-app.kubernetes.io/component=keycloak}"

fail() {
  echo "signalops_keycloak_staging_client_reconcile_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

case "$STAGING_ORIGIN" in
  https://*) ;;
  *) fail "SIGNALOPS_K8S_SIGNALOPS_STAGING_ORIGIN must be https for browser PKCE" ;;
esac

pod="$(kubectl get pods -n "$NAMESPACE" -l "$POD_SELECTOR" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
[[ -n "$pod" ]] || fail "no Keycloak pod found in ${NAMESPACE} with selector ${POD_SELECTOR}"

admin_user="$(kubectl get secret "$ADMIN_SECRET" -n "$NAMESPACE" -o jsonpath='{.data.KEYCLOAK_ADMIN}' | base64 -d)"
admin_password="$(kubectl get secret "$ADMIN_SECRET" -n "$NAMESPACE" -o jsonpath='{.data.KEYCLOAK_ADMIN_PASSWORD}' | base64 -d)"
[[ -n "$admin_user" && -n "$admin_password" ]] || fail "Keycloak admin credentials missing from ${NAMESPACE}/${ADMIN_SECRET}"

kubectl exec -n "$NAMESPACE" "$pod" -c keycloak -- /opt/keycloak/bin/kcadm.sh config credentials   --server http://localhost:8080   --realm master   --user "$admin_user"   --password "$admin_password" >/dev/null

tmp_client="$(mktemp -t signalops-keycloak-client.XXXXXX.json)"
tmp_update="$(mktemp -t signalops-keycloak-client-update.XXXXXX.sh)"
cleanup() {
  rm -f "$tmp_client" "$tmp_update"
}
trap cleanup EXIT

kubectl exec -n "$NAMESPACE" "$pod" -c keycloak -- /opt/keycloak/bin/kcadm.sh get clients -r "$REALM" -q "clientId=$CLIENT_ID" >"$tmp_client"

python3 - "$tmp_client" "$tmp_update" "$STAGING_ORIGIN" <<'PYCLIENT'
import json, shlex, sys
client_path, update_path, origin = sys.argv[1:4]
clients = json.load(open(client_path, encoding='utf-8'))
if not clients:
    raise SystemExit('client not found')
client = clients[0]
client_id = client.get('id') or ''
if not client_id:
    raise SystemExit('client id missing')
redirects = list(client.get('redirectUris') or [])
origins = list(client.get('webOrigins') or [])
required_redirects = [
    f'{origin}/auth/callback',
    f'{origin}/auth/silent-renew',
    f'{origin}/auth/signed-out',
]
for item in required_redirects:
    if item not in redirects:
        redirects.append(item)
if origin not in origins:
    origins.append(origin)
post_logout = client.get('attributes', {}).get('post.logout.redirect.uris', '')
post_logout_values = [v for v in post_logout.split(' ') if v]
for item in [f'{origin}/auth/signed-out', f'{origin}/marketops/pricing', f'{origin}/marketops/dashboard']:
    if item not in post_logout_values:
        post_logout_values.append(item)
lines = [
    f'CLIENT_UUID={shlex.quote(client_id)}',
    'REDIRECT_URIS=' + shlex.quote(json.dumps(redirects, separators=(",", ":"))),
    'WEB_ORIGINS=' + shlex.quote(json.dumps(origins, separators=(",", ":"))),
    'POST_LOGOUT=' + shlex.quote(' '.join(post_logout_values)),
]
open(update_path, 'w', encoding='utf-8').write('\n'.join(lines) + '\n')
PYCLIENT
# shellcheck source=/dev/null
source "$tmp_update"

kubectl exec -n "$NAMESPACE" "$pod" -c keycloak -- /opt/keycloak/bin/kcadm.sh update "clients/${CLIENT_UUID}" -r "$REALM"   -s "redirectUris=${REDIRECT_URIS}"   -s "webOrigins=${WEB_ORIGINS}"   -s "attributes.\"post.logout.redirect.uris\"=${POST_LOGOUT}" >/dev/null

cat <<EOF
signalops_keycloak_staging_client_reconcile_verified
namespace=${NAMESPACE}
realm=${REALM}
client_id=${CLIENT_ID}
staging_origin=${STAGING_ORIGIN}
redirects_added_or_present=true
web_origin_added_or_present=true
production_cutover_allowed=false
EOF
