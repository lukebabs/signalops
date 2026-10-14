#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s\n' \
    'Usage: scripts/subscriber_project_backup_runner_preflight.sh' \
    '' \
    'Checks that a production backup runner is ready to use pgBackRest safely.' \
    'It is read-only: it never creates a backup, enables WAL archiving, or writes to S3.' \
    '' \
    'Required environment:' \
    '  AWS_PROFILE                         profile that assumes the dedicated backup role' \
    '  SIGNALOPS_BACKUP_ROLE_ARN           expected role ARN' \
    '  SIGNALOPS_BACKUP_BUCKET             dedicated recovery bucket' \
    '  SIGNALOPS_BACKUP_PGBACKREST_CONFIG  absolute pgBackRest config path' \
    '' \
    'The profile must resolve to an STS assumed-role session for the supplied role.' \
    'Do not run with an AWS account-root identity or long-lived application credential.'
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

for command in aws pgbackrest; do
  command -v "$command" >/dev/null 2>&1 || {
    printf 'Backup runner preflight failed: %s is required.\n' "$command" >&2
    exit 3
  }
done

aws_profile="${AWS_PROFILE:-}"
expected_role_arn="${SIGNALOPS_BACKUP_ROLE_ARN:-}"
bucket="${SIGNALOPS_BACKUP_BUCKET:-}"
config_path="${SIGNALOPS_BACKUP_PGBACKREST_CONFIG:-}"

[[ -n "$aws_profile" ]] || { printf 'Backup runner preflight failed: AWS_PROFILE is required.\n' >&2; exit 2; }
[[ "$expected_role_arn" =~ ^arn:aws:iam::[0-9]{12}:role/[A-Za-z0-9+=,.@_-]+$ ]] || {
  printf 'Backup runner preflight failed: SIGNALOPS_BACKUP_ROLE_ARN must be an IAM role ARN.\n' >&2
  exit 2
}
[[ "$bucket" =~ ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$ ]] || {
  printf 'Backup runner preflight failed: SIGNALOPS_BACKUP_BUCKET is not a valid S3 bucket name.\n' >&2
  exit 2
}
[[ "$config_path" == /* && -f "$config_path" && -r "$config_path" ]] || {
  printf 'Backup runner preflight failed: SIGNALOPS_BACKUP_PGBACKREST_CONFIG must be a readable absolute file path.\n' >&2
  exit 2
}

identity_arn="$(aws --profile "$aws_profile" sts get-caller-identity --query Arn --output text)"
account_id="$(aws --profile "$aws_profile" sts get-caller-identity --query Account --output text)"
expected_account_id="${expected_role_arn#arn:aws:iam::}"
expected_account_id="${expected_account_id%%:role/*}"
expected_role_name="${expected_role_arn##*/}"

[[ "$identity_arn" != *':root' ]] || {
  printf 'Backup runner preflight failed: AWS account-root identity is forbidden.\n' >&2
  exit 4
}
[[ "$account_id" == "$expected_account_id" ]] || {
  printf 'Backup runner preflight failed: assumed identity belongs to unexpected AWS account.\n' >&2
  exit 4
}
[[ "$identity_arn" == "arn:aws:sts::${expected_account_id}:assumed-role/${expected_role_name}/"* ]] || {
  printf 'Backup runner preflight failed: identity must be an assumed-role session for %s.\n' "$expected_role_arn" >&2
  exit 4
}

aws --profile "$aws_profile" s3api get-bucket-location --bucket "$bucket" >/dev/null
pgbackrest --config="$config_path" info --output=json >/dev/null

printf 'Backup runner preflight passed: dedicated assumed role, reachable recovery bucket, and readable pgBackRest repository metadata verified. No backup or WAL operation was performed.\n'
