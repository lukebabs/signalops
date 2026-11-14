# SignalOps runtime stack and dependency contract

**Status:** production K3s rollout documented  
**Effective:** 2026-10-15  
**Audience:** engineering, operations, and platform administrators

This document records the current production-oriented stack after the K3s and
Dagster observability rollout. It complements the chronological
`docs/build_journal.md` by describing the steady-state topology and execution
order.

## Runtime topology

```text
Internet/WAF -> Istio ingress gateway (K3s) -> SignalOps HTTPRoute
                                         +--> signalops-web (SPA)
                                         +--> signalops-gateway (OIDC/API/SSE)
                                                  +--> dedicated MarketOps stores
                                                  +--> shared platform stores
                                                  +--> private Dagster read-only service

Signal-Connect -> ingestion/normalization -> MarketOps workers -> projections
Dagster -> dependency watermarks/task ledger (read-only graph + constrained record)
OpenBao -> runtime secret injection; GHCR -> immutable images
```

K3s is the active production authority for public traffic and MarketOps. Istio
owns ingress routing. Docker Compose remains a development/recovery surface and
must not be treated as a second production scheduler or source of truth.

## Code stack

- **Gateway:** Go (`cmd/gateway`, `internal/api`) handles OIDC authorization,
  tenant/subscription policy, MarketOps APIs, SSE, and the authenticated
  read-only Dagster proxy at `/admin/orchestration/dagster/`.
- **Web:** React/Vite (`web/`) provides the Admin Orchestration route. It
  bootstraps a short-lived path-scoped Dagster session and embeds a same-origin
  iframe; Dagster is never directly public.
- **Workers:** Go binaries under `cmd/` run from the immutable
  `signalops-marketops-k8s-job-runner` image. They require dedicated MarketOps
  database URLs when the boundary is enabled and resolve the mutable
  `marketops_primary_assets` list at run time.
- **Ingestion:** Signal-Connect supplies provider/event ingestion. Normalizer,
  persister, and materializers preserve lineage and write to the correct plane.
- **Orchestration:** Dagster 1.13.26 runs in `signalops-orchestration` and is
  the MarketOps scheduler authority. It launches one Kubernetes child Job per
  schedule, records dependency watermarks, and exposes read-only observability
  through the Admin workbench. Equivalent MarketOps CronJobs remain suspended
  to prevent duplicate execution.
- **Secrets and images:** OpenBao injects runtime credentials; private GHCR
  supplies immutable application and worker images. Neither secrets nor
  mutable runtime files belong in Git.

## Data-plane boundaries

1. **MarketOps primary PostgreSQL** — operational tables, task ledger, job
   status, algorithms, projections, and MarketOps subscription/tenant data.
2. **MarketOps temporal store** — normalized events, signals, features, and
   replayable time-series data.
3. **Shared SignalOps stores** — platform authentication, non-MarketOps data,
   CyberOps records, and rollback/recovery evidence.

MarketOps workers must not silently fall back to shared databases. Missing
dedicated URLs fail closed. The governed mutable asset list is the single
selector for the primary cohort; tasks bind to its list revision and stable
`global_asset_id`, not a hard-coded symbol array.

## Job and dependency contract

The post-close chain is: (1) preflight/session/list revision, (2) provider
capture, (3) normalization and EOD history, (4) Market State and technical
features, (5) Risk/Reward, Valuation, VC/DOSM, EROC, EEOM, and Options, (6) SRI
and ETF holdings, (7) SAF daily evaluation, (8) Syncratic Intelligence and
narrative projection, (9) guarded global dashboard projection, and (10)
operations monitoring and dependency-aware retry.

Each stage consumes the current list revision and current-session evidence.
Missing inputs create explicit partial/unavailable rows rather than silently
removing assets. Dashboard projection requires exact dependency watermark parity
(`expected=ready=projected`) before publishing. SAF daily validation is separate
from matured effectiveness. Failed provider calls are item-level evidence and
must not abort the cohort.

Production CronJobs use bounded retries, `Forbid` concurrency, immutable worker
images, and the task ledger. Dagster compares dependency watermarks and exposes
blocked/failed/completed states; it must not manufacture output to satisfy a
count. Scheduled Dagster runs carry the job id, scheduler authority, and
reconciliation attempt in run tags. A failure sensor automatically dispatches
one bounded retry through the same immutable Kubernetes Job template. Retry
attempts are capped at one at the Dagster workflow layer; per-asset retries
remain governed by the worker task manager. Missed historical ticks still
require an explicit session reconciliation because Dagster does not infer
backfill intent after an outage.

## Deployment and recovery authority

- **K3s/Istio:** apply production Kustomizations/Helm, update the immutable
  image tag, and wait for rollout readiness. Istio routes public traffic.
- **OpenBao:** must be unsealed and reachable before application/job pods start;
  NetworkPolicies allow only required namespace/service paths.
- **Docker Compose:** local development and controlled recovery only. A legacy
  Docker gateway may fail when the removed Docker-only `marketops-postgres`
  hostname is absent; this is not a K3s serving failure.

After changes, validate:

```bash
kubectl -n signalops-app rollout status deployment/signalops-gateway-production
kubectl -n signalops-app rollout status deployment/signalops-web-production
kubectl -n signalops-orchestration rollout status deployment/signalops-dagster-dagster-webserver-read-only
curl -skI https://signalops.syncratic.io/admin/orchestration/dagster/
```

The final request should redirect unauthenticated users to normal SignalOps
login. Platform administrators then see the native Orchestration page and
read-only Dagster iframe. Rollback uses the prior immutable image/Helm revision
and preserves encrypted pgBackRest baselines.
