#!/usr/bin/env bash
# Data-only, allowlisted loader for the root-owned MarketOps boundary secret.
# Source this trusted helper; never source the boundary env file itself.

load_marketops_boundary_env() {
  local env_path line key value line_number=0 mode owner
  env_path="${1:-}"
  [[ -n "$env_path" && -r "$env_path" ]] || {
    printf '%s\n' 'marketops_boundary_env_unreadable' >&2
    return 2
  }
  owner="$(stat -c '%u' "$env_path")"
  mode="$(stat -c '%a' "$env_path")"
  [[ "$owner" == "0" && "$mode" =~ ^[0-7]{3,4}$ && $((8#$mode & 8#077)) -eq 0 ]] || {
    printf '%s\n' 'marketops_boundary_env_permissions_invalid' >&2
    return 2
  }

  unset SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD
  while IFS= read -r line || [[ -n "$line" ]]; do
    line_number=$((line_number + 1))
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" == 'export '* ]] && line="${line#export }"
    [[ "$line" == *=* ]] || {
      printf 'marketops_boundary_env_invalid_assignment_line=%d\n' "$line_number" >&2
      return 2
    }
    key="${line%%=*}"
    value="${line#*=}"
    if [[ ${#value} -ge 2 ]] && { [[ "${value:0:1}" == '"' && "${value: -1}" == '"' ]] || [[ "${value:0:1}" == "'" && "${value: -1}" == "'" ]]; }; then
      value="${value:1:${#value}-2}"
    fi
    case "$key" in
      SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD|SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD)
        [[ -z "${!key+x}" ]] || {
          printf 'marketops_boundary_env_duplicate_key_line=%d\n' "$line_number" >&2
          return 2
        }
        printf -v "$key" '%s' "$value"
        export "$key"
        ;;
      *)
        printf 'marketops_boundary_env_unexpected_key_line=%d\n' "$line_number" >&2
        return 2
        ;;
    esac
  done < "$env_path"

  for key in SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD; do
    [[ "${!key:-}" =~ ^[A-Za-z0-9]{32,}$ ]] || {
      printf 'marketops_boundary_env_invalid_required_value\n' >&2
      return 2
    }
  done
}
