# SignalOps Dagster administrator observability

Dagster is exposed to platform administrators at:

`https://signalops.syncratic.io/admin/orchestration/dagster/`

The SignalOps gateway remains the authentication boundary. The administrator
navigation first bootstraps a short-lived, path-scoped session from the normal
OIDC bearer token, then embeds the same-origin Dagster UI in the native
Orchestration page. Only
`super_admin` and `signalops:admin` are accepted. Operators and subscription
administrators do not receive the navigation item or proxy access.

The public Istio route sends this path to the gateway. The gateway proxies to
the private read-only Dagster webserver service in `signalops-orchestration`.
Dagster's direct ingress remains disabled, and the read-only service rejects
or omits orchestration mutations. Kubernetes NetworkPolicy permits only the
production SignalOps gateway to reach that service.

This integration is observability-only: it does not change Dagster schedules,
run history, workers, databases, or provider polling behavior.

## Deployment evidence

On 2026-10-15, the Gateway and Web K3s Deployments were rebuilt and rolled out
with immutable GHCR tag `469ddff46d67-dagster-iframe`. Dagster Helm release
revision 12 created the ready service
`signalops-dagster-dagster-webserver-read-only` in
`signalops-orchestration`. Its readiness check uses a localhost request to the
path-prefixed `/admin/orchestration/dagster/server_info` endpoint; this avoids
requiring a broad kubelet ingress exception under the namespace NetworkPolicy.

The public route was verified unauthenticated and returned the expected
redirect to `/auth/login?redirect=...`. K3s remains the serving authority;
Docker Compose is retained only for development and controlled recovery.

## Scheduler-authority migration status

On 2026-10-15 the Dagster scheduler became authoritative for the 16 MarketOps
workflows. The definitions image is
`20261015-authoritative-scheduler-v2`. Dagster has least-privilege RBAC to read
CronJob templates and create/supervise child Jobs in `signalops-marketops`.

A bounded non-provider operations-monitor smoke created and completed a Dagster
child Job successfully. Cilium egress is limited to the Kubernetes API
endpoints (host/service API ports only) for the Dagster definitions pods. All
equivalent MarketOps Kubernetes CronJobs are now suspended, so duplicate
scheduling is prevented. The 16 Dagster schedules report `RUNNING`, while the
Dagster daemon, webserver, and definitions deployment are healthy.

Rollback is deterministic: stop the 16 Dagster schedules, unsuspend the
equivalent CronJobs, and retain the immutable definitions image tag.

### Schedule gRPC recovery

On 2026-10-15 the operations-monitor schedule reported
`DagsterUserCodeUnreachableError` after its user-code gRPC request exceeded 60
seconds. The definitions pod was healthy, but its Dagster PostgreSQL connection
was timing out because the scheduler egress policy allowed MarketOps and
OpenBao only; it did not allow the orchestration PostgreSQL service. The policy
now permits only the Dagster PostgreSQL master endpoint on TCP/5432. The
schedule retried successfully, launched run
`fd3b7243-8227-4473-92c5-3ea627cc9095`, and its child Job completed successfully.
