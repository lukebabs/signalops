#!/bin/sh
# Run inside the existing Keycloak container with its injected admin environment.
# Creates only one tenant-local, client-credentials reader. Never emits its secret.
set -eu

REALM=syncratic
CLIENT=syncratic-narrativeops-reader
ROLE=narrativeops:reader
KCADM=/opt/keycloak/bin/kcadm.sh
CFG=/tmp/syncratic-narrativeops-reader-kcadm-$$.config
trap 'rm -f "$CFG"' EXIT HUP INT TERM

test -n "${KEYCLOAK_ADMIN:-}" && test -n "${KEYCLOAK_ADMIN_PASSWORD:-}" || {
    echo 'keycloak_admin_environment_missing' >&2
    exit 1
}
"$KCADM" config credentials --config "$CFG" --server http://localhost:8080 \
    --realm master --user "$KEYCLOAK_ADMIN" --password "$KEYCLOAK_ADMIN_PASSWORD" >/dev/null 2>&1

# Fail closed if another operator has already created this exact client.
existing=$("$KCADM" get clients --config "$CFG" -r "$REALM" \
    -q "clientId=$CLIENT" --fields id,clientId)
test "$existing" = '[ ]' || {
    echo 'narrativeops_reader_client_already_exists' >&2
    exit 1
}

if ! "$KCADM" get "roles/$ROLE" --config "$CFG" -r "$REALM" --fields name >/dev/null 2>&1; then
    "$KCADM" create roles --config "$CFG" -r "$REALM" \
        -s "name=$ROLE" -s 'description=Read-only NarrativeOps narratives for exact Syncratic client' >/dev/null
fi

# Keep the new client disabled until mappers and the sole reader role are verified.
client_uuid=$("$KCADM" create clients --config "$CFG" -r "$REALM" -i \
    -s "clientId=$CLIENT" -s enabled=false -s publicClient=false \
    -s serviceAccountsEnabled=true -s standardFlowEnabled=false \
    -s directAccessGrantsEnabled=false -s implicitFlowEnabled=false \
    -s fullScopeAllowed=true -s protocol=openid-connect)
test -n "$client_uuid" || { echo 'narrativeops_reader_client_create_failed' >&2; exit 1; }

"$KCADM" create "clients/$client_uuid/protocol-mappers/models" --config "$CFG" -r "$REALM" \
    -b '{"name":"tenant_id","protocol":"openid-connect","protocolMapper":"oidc-hardcoded-claim-mapper","consentRequired":false,"config":{"claim.name":"tenant_id","claim.value":"tenant-local","jsonType.label":"String","access.token.claim":"true","id.token.claim":"false","userinfo.token.claim":"false","introspection.token.claim":"true"}}' >/dev/null
"$KCADM" create "clients/$client_uuid/protocol-mappers/models" --config "$CFG" -r "$REALM" \
    -b '{"name":"signalops-api-audience","protocol":"openid-connect","protocolMapper":"oidc-audience-mapper","consentRequired":false,"config":{"included.custom.audience":"signalops-api","access.token.claim":"true","id.token.claim":"false","userinfo.token.claim":"false","introspection.token.claim":"true"}}' >/dev/null

account=$("$KCADM" get "clients/$client_uuid/service-account-user" --config "$CFG" -r "$REALM" --fields id,username)
printf '%s' "$account" | grep -q '"username" : "service-account-syncratic-narrativeops-reader"' || {
    echo 'narrativeops_reader_service_account_mismatch' >&2
    exit 1
}
account_uuid=$(printf '%s\n' "$account" | sed -n 's/.*"id" : "\([^"]*\)".*/\1/p')
test -n "$account_uuid" || { echo 'narrativeops_reader_service_account_id_missing' >&2; exit 1; }
"$KCADM" add-roles --config "$CFG" -r "$REALM" --uid "$account_uuid" --rolename "$ROLE" >/dev/null

roles=$("$KCADM" get "users/$account_uuid/role-mappings/realm" --config "$CFG" -r "$REALM" --fields name)
printf '%s' "$roles" | grep -q '"name" : "narrativeops:reader"' || {
    echo 'narrativeops_reader_role_missing' >&2
    exit 1
}
if printf '%s' "$roles" | grep -Eq '"name" : "(signalops:admin|super_admin)"'; then
    echo 'narrativeops_reader_admin_role_present' >&2
    exit 1
fi
mapper_names=$("$KCADM" get "clients/$client_uuid/protocol-mappers/models" --config "$CFG" -r "$REALM" --fields name)
printf '%s' "$mapper_names" | grep -q '"name" : "tenant_id"' || exit 1
printf '%s' "$mapper_names" | grep -q '"name" : "signalops-api-audience"' || exit 1

"$KCADM" update "clients/$client_uuid" --config "$CFG" -r "$REALM" -s enabled=true >/dev/null
echo 'narrativeops_reader_keycloak_provisioned=true tenant=tenant-local role=narrativeops:reader audience=signalops-api secret_printed=false'
