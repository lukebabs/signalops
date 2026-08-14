#!/usr/bin/env bash
set -euo pipefail

required=(
  SIGNALOPS_BACKUP_RUNNER_ACCESS_KEY_ID
  SIGNALOPS_BACKUP_RUNNER_SECRET_ACCESS_KEY
  SIGNALOPS_PGBACKREST_CIPHER_PASS_FILE
  SIGNALOPS_PGBACKREST_CONFIG_PATH
)
for name in "${required[@]}"; do
  [[ -n "${!name:-}" ]] || { printf 'missing required environment: %s\n' "$name" >&2; exit 2; }
done

role_arn="${SIGNALOPS_BACKUP_ROLE_ARN:-arn:aws:iam::354918409279:role/signalops-postgres-backup}"
bucket="${SIGNALOPS_BACKUP_BUCKET:-signalops-production-postgres-backups-354918409279-us-east-1}"
region="${AWS_REGION:-us-east-1}"
config_path="$SIGNALOPS_PGBACKREST_CONFIG_PATH"
cipher_pass_file="$SIGNALOPS_PGBACKREST_CIPHER_PASS_FILE"

for command in aws install mktemp dirname jq tr hostname cat rm chown chmod mv; do
  command -v "$command" >/dev/null 2>&1 || { printf 'required command not found: %s\n' "$command" >&2; exit 3; }
done

[[ "$config_path" == /* && "$cipher_pass_file" == /* ]] || {
  printf 'credential and configuration paths must be absolute\n' >&2
  exit 2
}

if [[ ! -s "$cipher_pass_file" ]]; then
  printf 'pgBackRest cipher pass file is absent or empty: %s\n' "$cipher_pass_file" >&2
  exit 3
fi

session_json="$(AWS_ACCESS_KEY_ID="$SIGNALOPS_BACKUP_RUNNER_ACCESS_KEY_ID" \
  AWS_SECRET_ACCESS_KEY="$SIGNALOPS_BACKUP_RUNNER_SECRET_ACCESS_KEY" \
  AWS_REGION="$region" \
  aws sts assume-role \
    --role-arn "$role_arn" \
    --role-session-name "signalops-pgbackrest-$(hostname -s)" \
    --duration-seconds 3600 \
    --output json)"

access_key_id="$(printf '%s' "$session_json" | jq -r '.Credentials.AccessKeyId')"
secret_access_key="$(printf '%s' "$session_json" | jq -r '.Credentials.SecretAccessKey')"
session_token="$(printf '%s' "$session_json" | jq -r '.Credentials.SessionToken')"
cipher_pass="$(tr -d '\r\n' < "$cipher_pass_file")"

for value in "$access_key_id" "$secret_access_key" "$session_token" "$cipher_pass"; do
  [[ -n "$value" && "$value" != null ]] || { printf 'AWS role credential rendering failed\n' >&2; exit 4; }
done

config_dir="$(dirname "$config_path")"
install -d -m 0750 -o root -g root "$config_dir"
rendered="$(mktemp "$config_dir/.pgbackrest.conf.XXXXXX")"
trap 'rm -f "$rendered"' EXIT

umask 0077
cat > "$rendered" <<EOF
[global]
repo1-type=s3
repo1-path=/signalops-postgres
repo1-s3-bucket=$bucket
repo1-s3-region=$region
repo1-s3-endpoint=s3.amazonaws.com
repo1-s3-key=$access_key_id
repo1-s3-key-secret=$secret_access_key
repo1-s3-token=$session_token
repo1-cipher-type=aes-256-cbc
repo1-cipher-pass=$cipher_pass
repo1-retention-full=12
repo1-retention-full-type=count
repo1-retention-diff=35
start-fast=y
process-max=2
archive-async=y
spool-path=/var/spool/pgbackrest
log-level-console=info

[signalops]
pg1-path=/var/lib/postgresql/data
pg1-user=signalops

[marketops-primary]
pg1-path=/var/lib/postgresql/data
pg1-user=signalops

[marketops-temporal]
pg1-path=/var/lib/postgresql/data
pg1-user=signalops
EOF

chown root:70 "$rendered"
chmod 0640 "$rendered"
mv -f "$rendered" "$config_path"
trap - EXIT
printf 'Rendered renewed pgBackRest assumed-role configuration at %s.\n' "$config_path"
