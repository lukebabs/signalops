# K8S production cutover parity plan

Status: active. Production cutover is not approved.

Last updated: 2026-09-13.

## Purpose

This plan turns the broader Kubernetes production-cutover work into an executable parity gate. It keeps Docker Compose/systemd as the production authority while Kubernetes proves the same application, data, scheduler, identity, ingestion, rollback, and observability behavior under controlled staging conditions.

OpenBao HA is not a production blocker by itself. The production requirement is recoverability and control: backup/restore, seal/unseal recovery, audit logging, per-plane policies, CA trust, and rollback. True OpenBao HA remains a resilience-hardening target. The Kubernetes platform target now includes a service mesh; see [K8S service mesh architecture decision](k8s_service_mesh_architecture.md).

## Current executable verifier

Run from the SignalOps workspace:

```bash
scripts/verify_k8s_production_cutover_readiness.sh
```

The verifier is intentionally non-cutover. It renders manifests and verifies guardrails; it does not apply production traffic, create production Ingress, enable Kubernetes schedules, invoke providers, or change DNS.

Expected current result:

```text
signalops_k8s_production_cutover_readiness_report
production_cutover_allowed=false
compose_systemd_production_authority=true
...
pending_authenticated_keycloak_k8s_parity=false
pending_authenticated_keycloak_mesh_route_parity=false
keycloak_oidc_discovery_reachability=verified
pending_broader_marketops_scheduler_parity=true
k8s_marketops_scheduler_parity_coverage=partial
pending_signal_connect_ingestion_shadow=true
mesh1_istio_control_plane=verified_2026-09-13
proven_service_mesh_signalops_route_parity=authenticated_keycloak_playwright_2026-09-13
pending_service_mesh_ingress_dns_cutover_plan=true
pending_capacity_load_validation=true
```

Use `--strict` only when intentionally checking that final production readiness is not yet satisfied. In the current phase, `--strict` should fail.

## Evidence already closed

| Area | Evidence | Status |
| --- | --- | --- |
| K8S-1 namespace/secret foundation | `k8s1_openbao_staging_foundation_evidence_2026-09-12.md` | Closed for staging |
| K8S-2 app manifests | `k8s2_staging_app_manifest_evidence_2026-09-12.md` | Closed |
| K8S-2 app shell/proxy parity | `k8s2_staging_app_parity_evidence_2026-09-12.md` | Closed without auth callback |
| K8S-3 MarketOps CronJob scaffold | `k8s3_marketops_scheduled_jobs_scaffold_2026-09-12.md` | Closed suspended |
| K8S-3 no-provider CronJob controller smoke | `k8s3_cronjob_unsuspend_resuspend_smoke_2026-09-12.md` | Closed |
| K8S-3 provider-enabled FMP smoke | `k8s3_provider_cronjob_smoke_2026-09-12.md` | Closed for one asset/no retry |
| OpenBao CA trust | `k8s3_openbao_ca_trust_gate_2026-09-12.md` | Closed for staging app and MarketOps jobs |
| Current backup/restore | `pr3_backup_restore_refresh_evidence_2026-09-12.md` | Closed for this cycle |
| Mesh-0 baseline and implementation plan | `k8s_mesh0_implementation_plan_2026-09-12.md` | Drafted / verifier added |
| Mesh-3 authenticated SignalOps route | `k8s_mesh3_keycloak_staging_route_blocker_2026-09-13.md` | Closed for staging |

## Remaining parity gates

### Gate A — authenticated K8S app parity

Status: closed for the staging route on 2026-09-13.

Evidence: `scripts/run_k8s_mesh3_keycloak_staging_route_smoke.sh` passed through `https://signalops-staging.syncratic.co` with browser PKCE, Keycloak login, SignalOps `/auth/callback`, and `/v1/session/enrollment`.

Acceptance met:

- authenticated SignalOps route passes through K8S web/gateway and Istio;
- no production DNS changes;
- no provider polling;
- app workloads scale back to zero after the smoke;
- staging runtime points at the dedicated MarketOps staging data boundary through OpenBao.

Remaining product-route breadth validation for all named pages can be covered by the broader pre-cutover route smoke, but the Keycloak callback blocker itself is closed.

### Gate B — broader MarketOps scheduler parity

The current Kubernetes scheduler evidence proves the controller mechanics and one provider-enabled FMP annual path. It does not yet prove all production MarketOps schedules.

Acceptance:

- dry-run or shadow parity for intraday, daily post-close, warm EOD, SRI refresh, SRI holdings refresh, SAF benchmark/projection, FMP annual, operations monitor, and retention governance;
- each job writes scheduler status to the dedicated MarketOps operations tables;
- every CronJob keeps `concurrencyPolicy: Forbid`, explicit deadline/history limits, and bounded retry posture;
- provider-enabled tests require named approval and one-job/no-retry constraints until production cutover is approved;
- Docker/systemd remains authoritative until the full scheduler parity report is accepted.


### Scheduler parity coverage report — 2026-09-13

A source-controlled parity verifier now compares the production Admin scheduler catalog with the staged Kubernetes CronJob catalog and the Kubernetes job-runner entrypoint. Current result:

```text
signalops_k8s_marketops_scheduler_parity_coverage_report
status=partial
admin_marketops_jobs=12
k8s_cronjobs=5
k8s_entrypoint_jobs=5
fully_represented=marketops-fmp-annual-financial,marketops-intraday,marketops-operations-monitor,marketops-retention-governance,marketops-sri-holdings-refresh,marketops-sri-refresh
missing_cronjob=marketops-daily-postclose,marketops-fmp-continuation,marketops-postclose-recovery,marketops-risk-reward,marketops-task-retry,marketops-warm-eod
missing_entrypoint=marketops-daily-postclose,marketops-fmp-continuation,marketops-postclose-recovery,marketops-risk-reward,marketops-task-retry,marketops-warm-eod
extra_cronjob=marketops-saf-benchmark
provider_polling=false
production_cutover_allowed=false
```

Recommended porting order:

1. `marketops-operations-monitor` and `marketops-retention-governance`, because they are operational/non-provider and prove control-plane hygiene.
2. `marketops-task-retry`, `marketops-postclose-recovery`, and `marketops-risk-reward`, because they close the post-close recovery/completion loop without introducing a broad provider surface.
3. `marketops-warm-eod` and `marketops-daily-postclose`, because they are the production-critical EOD pipelines and require the most careful provider/data-boundary validation.
4. `marketops-fmp-continuation`, because it is a weekend/continuation workflow that should be validated after the base EOD loop is proven.

The verifier is intentionally non-cutover and makes the gap explicit instead of treating the existing five staged CronJobs as full production scheduler parity.


### Scheduler parity operational slice — 2026-09-13

Closed the first low-risk scheduler-porting slice by adding suspended Kubernetes CronJobs and entrypoint support for:

- `marketops-operations-monitor`
- `marketops-retention-governance`

Both are K8s-staging dry-run only until production scheduler cutover is separately approved. The operation monitor path validates dedicated primary/temporal DB reachability and scheduler-status table access. The retention path executes the existing `signalops-retention-governor` binary for `subscriber.user_activity_180d` on `tenant-local` and `tenant-pilot-b` without enforcement.

Passing evidence:

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
job_id=marketops-operations-monitor
dry_run=true
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false

signalops_k8s_marketops_non_provider_dry_run_job_verified
job_id=marketops-retention-governance
dry_run=true
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

Coverage after this slice:

```text
fully_represented=marketops-fmp-annual-financial,marketops-intraday,marketops-operations-monitor,marketops-retention-governance,marketops-sri-holdings-refresh,marketops-sri-refresh
missing_cronjob=marketops-daily-postclose,marketops-fmp-continuation,marketops-postclose-recovery,marketops-risk-reward,marketops-task-retry,marketops-warm-eod
missing_entrypoint=marketops-daily-postclose,marketops-fmp-continuation,marketops-postclose-recovery,marketops-risk-reward,marketops-task-retry,marketops-warm-eod
```

### Gate C — Signal-Connect ingestion shadow

Signal-Connect is part of SignalOps and should move as its own ingestion plane inside the same SaaS platform architecture.

Acceptance:

- Signal-Connect ingress/webhook endpoints run behind staging ingress with rate limits and backpressure;
- accepted raw events and outbox/DLQ behavior match contract expectations;
- Connect cannot read app, identity, MarketOps, or CyberOps secrets;
- no production detector or MarketOps consumer switches until shadow evidence passes.

### Gate D — service mesh, ingress/DNS, and rollback proposal

Mesh-0 has a live baseline in [Mesh-0 implementation plan — 2026-09-12](k8s_mesh0_implementation_plan_2026-09-12.md). Mesh-1 closed on 2026-09-13 with Istio selected and verified: `istio-base` and `istiod` are deployed, `GatewayClass/istio` is accepted, and `istio-system/public-ingress` is programmed at `192.168.2.233`. Docker Traefik remains the public rollback/reference edge for SignalOps until SignalOps route parity and cutover are separately approved.

This gate prepares but does not execute the traffic move.

Acceptance:

- target Gateway API / Istio ingress mode and hostnames documented;
- unauthenticated SignalOps staging route parity is proven through Istio;
- Keycloak OIDC discovery/JWKS endpoints are reachable without WAF challenge, and redirect URIs/web origins include the target K8S hostnames;
- Stripe webhook endpoint behavior is validated for the K8S route;
- rollback path returns traffic from mesh/Gateway API to Docker Compose/systemd without data loss;
- backup/restore evidence is current;
- cutover can be performed as a small timed window with clear abort criteria.

### Gate E — capacity/load validation

Acceptance:

- measured gateway p95/p99 latency under target concurrent sessions;
- database connection saturation measured with pool caps and PgBouncer/connection-pooling decision;
- scheduler queue lag measured under MarketOps jobs;
- Syncratic Ask / AI Gateway throughput and timeout behavior measured;
- provider-rate ceilings documented and protected.

## Current conclusion

Kubernetes migration is past scaffold-only. It has real staging app, secret, image, data, scheduler, service-mesh, and authenticated Keycloak route evidence. The next production-readiness target is broader MarketOps scheduler parity, followed by Signal-Connect ingestion shadow, ingress/DNS rollback planning, Stripe webhook parity through the K8S route, and capacity/load validation.

Until those gates close, `production_cutover_allowed=false` remains the correct state and Docker Compose/systemd remains the live production authority.

## Verifier evidence — 2026-09-12

The non-cutover verifier was added and executed successfully after correcting the base scaffold verifier so it distinguishes base resources from intentional staging overlays.

The corrections were:

- exclude staging NetworkPolicies from the base K8S-1 policy count;
- exclude staging ConfigMaps from the base K8S-1 policy ConfigMap count;
- exclude staging pods from the base K8S-1 zero-workload-pod assertion.

This preserves the original K8S-1 contract while allowing later K8S-2/K8S-3 staging resources to coexist in the same namespace set.

Passing report:

```text
signalops_k8s_production_cutover_readiness_report
production_cutover_allowed=false
compose_systemd_production_authority=true
k8s_base_scaffold=verified
k8s_app_manifest_guard=verified
k8s_marketops_jobs_manifest_guard=verified
k8s_marketops_data_manifest_guard=verified
openbao_ca_trust=verified
backup_restore_current=verified_2026-09-12
proven_app_parity=port_forward_unauthenticated+mesh_authenticated_https
proven_scheduler_parity=no_provider_and_one_provider_fmp_smoke
pending_authenticated_keycloak_k8s_parity=false
pending_authenticated_keycloak_mesh_route_parity=false
keycloak_oidc_discovery_reachability=verified
pending_broader_marketops_scheduler_parity=true
k8s_marketops_scheduler_parity_coverage=partial
pending_signal_connect_ingestion_shadow=true
mesh1_istio_control_plane=verified_2026-09-13
proven_service_mesh_signalops_route_parity=authenticated_keycloak_playwright_2026-09-13
pending_service_mesh_ingress_dns_cutover_plan=true
pending_capacity_load_validation=true
```

## Scheduler parity task-retry/warm-EOD slice — 2026-09-13

Status: source-prepared and manifest-verified; live K8S one-shot dry-run evidence follows image publication.

This slice adds K8S-staging representation for two more Admin MarketOps jobs:

- `marketops-task-retry` — dry-run-only handler counts due tactical retries in the dedicated staging MarketOps primary and records scheduler status parity. It does not execute tactical valuation or provider work.
- `marketops-warm-eod` — dry-run-only handler verifies the warm EOD cohort source in the dedicated staging MarketOps primary and records scheduler status parity. It does not invoke Massive and does not materialize EOD evidence.

The staging CronJobs remain suspended with `concurrencyPolicy: Forbid`, OpenBao runtime injection, GHCR private image pull, and `production-cutover-allowed=false`.

Current parity after this source slice: 8 of 12 Admin MarketOps jobs are represented by both a K8S CronJob and an entrypoint handler. The remaining missing jobs are `marketops-daily-postclose`, `marketops-fmp-continuation`, `marketops-postclose-recovery`, and `marketops-risk-reward`.

### Task-retry/warm-EOD live dry-run evidence status

Closed on 2026-09-13. Commit `de5c983` is pushed, and the private GHCR job-runner image is published as `de5c9833fd04` plus `staging`. The dedicated staging schema bootstrap passed with `tables=13`.

Live one-shot K8S dry-run evidence passed for both `marketops-task-retry` and `marketops-warm-eod` against the dedicated staging MarketOps primary/temporal services. Both runs verified DB-backed scheduler status parity and preserved `provider_polling=false` and `production_cutover_allowed=false`. Detailed evidence is captured in [K8S-3 task-retry and warm-EOD dry-run evidence — 2026-09-13](k8s3_task_retry_warm_eod_dry_run_evidence_2026-09-13.md).

The broader scheduler parity gap is now limited to `marketops-daily-postclose`, `marketops-fmp-continuation`, `marketops-postclose-recovery`, and `marketops-risk-reward`.
