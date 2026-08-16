#!/usr/bin/env bash
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec marketops_compose --profile subscriber-global-evidence run --rm subscriber-global-annual-financial-task-worker --execute --max-assets 1000
