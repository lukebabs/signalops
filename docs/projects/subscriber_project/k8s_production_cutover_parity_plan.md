# K8S production cutover parity plan

Status: active. Production cutover is not approved.

Last updated: 2026-09-12.

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
pending_authenticated_keycloak_k8s_parity=true
pending_authenticated_keycloak_mesh_route_parity=true
keycloak_oidc_discovery_reachability=blocked_http_503_2026-09-13
pending_broader_marketops_scheduler_parity=true
pending_signal_connect_ingestion_shadow=true
mesh1_istio_control_plane=verified_2026-09-13
proven_service_mesh_signalops_route_parity=unauthenticated_playwright_2026-09-13
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

## Remaining parity gates

### Gate A — authenticated K8S app parity

Current app parity uses localhost port-forward. That proves the SPA, web-to-gateway proxying, readiness probes, and NetworkPolicies, but it cannot complete the real Keycloak callback journey because the production OIDC client expects the production callback host.

To close this gate, choose one controlled path:

1. create a staging hostname and add it to the Keycloak client redirect URI/web origin allow-list; or
2. create a dedicated staging Keycloak client with the same claims, roles, audience, and token behavior.

Acceptance:

- Dashboard, Watchlists, Assets, Pricing, Profile/Settings, Syncratic Intelligence, and protected-route behavior pass through the K8S web/gateway path;
- tenant-local and tenant-pilot-b identities behave exactly as production;
- no production DNS changes;
- no provider polling;
- app workloads scale back or remain explicitly marked staging-only.

### Gate B — broader MarketOps scheduler parity

The current Kubernetes scheduler evidence proves the controller mechanics and one provider-enabled FMP annual path. It does not yet prove all production MarketOps schedules.

Acceptance:

- dry-run or shadow parity for intraday, daily post-close, warm EOD, SRI refresh, SRI holdings refresh, SAF benchmark/projection, FMP annual, operations monitor, and retention governance;
- each job writes scheduler status to the dedicated MarketOps operations tables;
- every CronJob keeps `concurrencyPolicy: Forbid`, explicit deadline/history limits, and bounded retry posture;
- provider-enabled tests require named approval and one-job/no-retry constraints until production cutover is approved;
- Docker/systemd remains authoritative until the full scheduler parity report is accepted.

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

Kubernetes migration is past scaffold-only. It has real staging app, secret, image, data, and scheduler evidence. The next production-readiness target is authenticated K8S app parity, followed by broader MarketOps scheduler parity.

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
proven_app_parity=port_forward_unauthenticated
proven_scheduler_parity=no_provider_and_one_provider_fmp_smoke
pending_authenticated_keycloak_k8s_parity=true
pending_authenticated_keycloak_mesh_route_parity=true
keycloak_oidc_discovery_reachability=blocked_http_503_2026-09-13
pending_broader_marketops_scheduler_parity=true
pending_signal_connect_ingestion_shadow=true
mesh1_istio_control_plane=verified_2026-09-13
proven_service_mesh_signalops_route_parity=unauthenticated_playwright_2026-09-13
pending_service_mesh_ingress_dns_cutover_plan=true
pending_capacity_load_validation=true
```
