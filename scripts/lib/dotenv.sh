#!/usr/bin/env bash
# Data-only dotenv loader. Source this trusted library, never an env file.

load_dotenv() {
  local dotenv_path line key value line_number=0
  dotenv_path="${1:-}"
  if [[ -z "$dotenv_path" || ! -r "$dotenv_path" ]]; then
    printf '%s\n' 'dotenv_path_unreadable' >&2
    return 2
  fi

  while IFS= read -r line || [[ -n "$line" ]]; do
    line_number=$((line_number + 1))
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    if [[ "$line" == 'export '* ]]; then
      line="${line#export }"
    fi
    if [[ "$line" != *=* ]]; then
      printf 'dotenv_invalid_assignment_line=%d\n' "$line_number" >&2
      return 2
    fi
    key="${line%%=*}"
    value="${line#*=}"
    if [[ ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
      printf 'dotenv_invalid_key_line=%d\n' "$line_number" >&2
      return 2
    fi
    case "$key" in
      BASH_ENV|BASHOPTS|CDPATH|ENV|IFS|SHELLOPTS|PATH|LD_PRELOAD|LD_LIBRARY_PATH|PYTHONHOME|PYTHONPATH|PERL5OPT|RUBYOPT|NODE_OPTIONS)
        printf 'dotenv_reserved_key_line=%d\n' "$line_number" >&2
        return 2
        ;;
    esac
    if [[ ${#value} -ge 2 ]] && { [[ "${value:0:1}" == '"' && "${value: -1}" == '"' ]] || [[ "${value:0:1}" == "'" && "${value: -1}" == "'" ]]; }; then
      value="${value:1:${#value}-2}"
    fi
    printf -v "$key" '%s' "$value"
    export "$key"
  done < "$dotenv_path"
}
