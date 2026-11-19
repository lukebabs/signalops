#!/usr/bin/env bash
set -euo pipefail

MANIFEST_DIR="${1:-deploy/kubernetes/staging/marketops-jobs}"
ENTRYPOINT="${SIGNALOPS_K8S_MARKETOPS_ENTRYPOINT:-scripts/k8s_marketops_job_entrypoint.sh}"
ADMIN_CATALOG="${SIGNALOPS_MARKETOPS_ADMIN_SCHEDULER_CATALOG:-internal/api/scheduled_jobs.go}"

fail() {
  echo "signalops_k8s_marketops_scheduler_parity_coverage_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"
[[ -f "$ENTRYPOINT" ]] || fail "entrypoint not found: ${ENTRYPOINT}"
[[ -f "$ADMIN_CATALOG" ]] || fail "admin scheduler catalog not found: ${ADMIN_CATALOG}"

rendered="$(kubectl kustomize "$MANIFEST_DIR")"
[[ -n "$rendered" ]] || fail "kustomize render produced no output"

export rendered ENTRYPOINT ADMIN_CATALOG
python3 - <<'PY'
import os
import re
from pathlib import Path

rendered = os.environ["rendered"]
entrypoint = Path(os.environ["ENTRYPOINT"]).read_text()
admin_catalog = Path(os.environ["ADMIN_CATALOG"]).read_text()

admin_jobs = re.findall(r'\{"([^"]+)",\s*"[^"]+",\s*"[^"]+",\s*"[^"]+",\s*"scheduler-run-now:[^"]+"\}', admin_catalog)
admin_marketops_jobs = [job for job in admin_jobs if job.startswith("marketops-")]

cronjobs = []
for doc in re.split(r"(?m)^---\s*$", rendered):
    if not re.search(r"(?m)^kind:\s+CronJob\s*$", doc):
        continue
    match = re.search(r"(?m)^metadata:\s*$.*?^  name:\s+([a-z0-9-]+)\s*$", doc, re.S)
    if match:
        cronjobs.append(match.group(1))
entrypoint_jobs = re.findall(r"(?m)^\s{2}([a-z0-9-]+)\)\s*$", entrypoint)

cron_set = set(cronjobs)
entrypoint_set = set(entrypoint_jobs)
admin_set = set(admin_marketops_jobs)

fully_represented = sorted(admin_set & cron_set & entrypoint_set)
missing_cronjob = sorted(admin_set - cron_set)
missing_entrypoint = sorted(admin_set - entrypoint_set)
extra_cronjob = sorted(cron_set - admin_set)

status = "complete" if not missing_cronjob and not missing_entrypoint else "partial"

print("signalops_k8s_marketops_scheduler_parity_coverage_report")
print(f"status={status}")
print(f"admin_marketops_jobs={len(admin_marketops_jobs)}")
print(f"k8s_cronjobs={len(cronjobs)}")
print(f"k8s_entrypoint_jobs={len(entrypoint_jobs)}")
print("fully_represented=" + ",".join(fully_represented))
print("missing_cronjob=" + (",".join(missing_cronjob) if missing_cronjob else "none"))
print("missing_entrypoint=" + (",".join(missing_entrypoint) if missing_entrypoint else "none"))
print("extra_cronjob=" + (",".join(extra_cronjob) if extra_cronjob else "none"))
print("provider_polling=false")
print("production_cutover_allowed=false")

if not admin_marketops_jobs:
    raise SystemExit("admin scheduler catalog produced no MarketOps jobs")
if not cronjobs:
    raise SystemExit("K8s scheduler manifest produced no CronJobs")
PY
