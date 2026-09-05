#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this provisioning command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
key_json_path="${1:-/tmp/signalops-postgres-backup-runner-access-key.json}"
secret_dir=/etc/signalops
source_env="$secret_dir/pgbackrest-source.env"
cipher_pass_file="$secret_dir/pgbackrest.cipher-pass"
config_path="$secret_dir/pgbackrest.conf"

for command in aws docker jq openssl install rm systemctl; do
  command -v "$command" >/dev/null 2>&1 || {
    printf 'required command not found: %s\n' "$command" >&2
    exit 3
  }
done

[[ -f "$key_json_path" ]] || {
  printf 'restricted backup-runner access-key handoff not found: %s\n' "$key_json_path" >&2
  exit 3
}

access_key_id="$(jq -er '.AccessKey.AccessKeyId' "$key_json_path")"
secret_access_key="$(jq -er '.AccessKey.SecretAccessKey' "$key_json_path")"

install -d -m 0750 -o root -g root "$secret_dir"
umask 0077
source_tmp="$(mktemp "$secret_dir/.pgbackrest-source.env.XXXXXX")"
trap 'rm -f "$source_tmp"' EXIT
printf '%s\n' \
  "SIGNALOPS_BACKUP_RUNNER_ACCESS_KEY_ID=$access_key_id" \
  "SIGNALOPS_BACKUP_RUNNER_SECRET_ACCESS_KEY=$secret_access_key" \
  "SIGNALOPS_BACKUP_ROLE_ARN=arn:aws:iam::354918409279:role/signalops-postgres-backup" \
  "SIGNALOPS_BACKUP_BUCKET=signalops-production-postgres-backups-354918409279-us-east-1" \
  "SIGNALOPS_PGBACKREST_CIPHER_PASS_FILE=$cipher_pass_file" \
  "SIGNALOPS_PGBACKREST_CONFIG_PATH=$config_path" \
  "AWS_REGION=us-east-1" \
  > "$source_tmp"
install -m 0600 -o root -g root "$source_tmp" "$source_env"

if [[ ! -s "$cipher_pass_file" ]]; then
  cipher_tmp="$(mktemp "$secret_dir/.pgbackrest.cipher-pass.XXXXXX")"
  trap 'rm -f "$source_tmp" "$cipher_tmp"' EXIT
  openssl rand -base64 48 | tr -d '\r\n' > "$cipher_tmp"
  install -m 0600 -o root -g root "$cipher_tmp" "$cipher_pass_file"
fi

set -a
# shellcheck disable=SC1090
. "$source_env"
set +a
"$root_dir/scripts/refresh_signalops_pgbackrest_credentials.sh"

compose=(docker compose -f "$root_dir/compose.yaml" -f "$root_dir/compose.pgbackrest.yaml")
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" up -d --build postgres
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
  pgbackrest --stanza=signalops stanza-create
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T postgres \
  psql -U signalops -d signalops -v ON_ERROR_STOP=1 \
  -c "ALTER SYSTEM SET archive_mode = 'on';" \
  -c "ALTER SYSTEM SET archive_command = 'pgbackrest --stanza=signalops archive-push %p';" \
  -c "ALTER SYSTEM SET archive_timeout = '15min';"
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" restart postgres
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
  pgbackrest --stanza=signalops check
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
  pgbackrest --stanza=signalops --type=full backup
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres postgres \
  pgbackrest --stanza=signalops info --output=json
"$root_dir/scripts/install_signalops_postgres_pgbackrest_system_timer.sh"

rm -f "$key_json_path"
printf '%s\n' 'pgBackRest provisioning completed. The bootstrap access-key handoff was removed; its protected source credential now exists only at /etc/signalops/pgbackrest-source.env.'
