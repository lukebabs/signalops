# K8S Mesh-4 ingress and DNS cutover plan — 2026-09-13

Status: plan verified; no production DNS or traffic movement approved.

## Purpose

This gate defines how SignalOps can move public user traffic from the current Docker Compose / Traefik front door to the k3s Istio Gateway API front door without losing rollback control. It is a production-readiness planning gate, not a cutover gate.

## Current authority

Production authority remains:

- Docker Compose for public `web` and `gateway` entrypoints;
- Docker Traefik for the public `signalops.syncratic.io` route;
- systemd/deployment-agent for scheduled MarketOps jobs;
- dedicated Docker Compose MarketOps databases as the live MarketOps data stores.

K3s authority is staging-only for SignalOps app, MarketOps scheduled-job parity, Signal-Connect shadow workers, raw-worker shadow processing, OpenBao secret staging, and Istio route parity.

## Target ingress model

The target production path is:

```text
Internet / CDN / WAF
  -> signalops.syncratic.io DNS
  -> k3s Istio Gateway API listener
  -> istio-system/public-ingress
  -> signalops-app/signalops-web or signalops-app/signalops-gateway
```

Route split:

- `/v1/*`, `/healthz`, and `/readyz` route to `signalops-gateway`.
- `/auth/*` routes to `signalops-gateway` so the app-hosted auth facade remains stable.
- `/marketops/*`, `/admin/*`, `/profile`, `/settings`, and static UI routes route to `signalops-web`.
- Stripe webhook endpoints route to `signalops-gateway`, but are not production-ready through K3s until the Stripe webhook parity gate passes.

## Required pre-cutover checks

Before any named production traffic approval, all of the following must pass on the same source revision and image tags intended for cutover:

1. `scripts/verify_k8s_production_cutover_readiness.sh`
2. `scripts/verify_k8s_mesh1_istio_readiness.sh`
3. `scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh`
4. `scripts/run_k8s_mesh3_keycloak_staging_route_smoke.sh`
5. subscriber UI smoke through the current production route
6. subscriber UI smoke through the staging Istio route
7. Stripe checkout canary through current production route
8. Stripe webhook parity through the K3s gateway route
9. capacity/load validation for web, gateway, and critical MarketOps read endpoints
10. backup/restore and shared Postgres archive-health checks

## DNS cutover shape

The preferred cutover should use a low-TTL DNS change at the authoritative DNS provider after the staging Istio route is proven.

Pre-change:

- lower TTL for `signalops.syncratic.io` to a short rollback-friendly value;
- confirm current Docker Traefik route remains healthy;
- confirm K3s Istio gateway address and listener readiness;
- confirm Keycloak redirect URI and web-origin compatibility;
- confirm CDN/WAF forwarding headers and TLS behavior.

Cutover:

- move `signalops.syncratic.io` to the k3s Gateway address;
- keep Docker Compose services running as rollback target;
- keep systemd scheduler authority unchanged unless a separate scheduler-authority cutover is approved;
- do not change provider-polling schedules during ingress movement.

Post-change validation window:

- `/readyz` returns success through public DNS;
- browser login and token refresh work;
- Dashboard, Assets, Watchlists, Pricing, Profile, Admin Subscriptions, and Syncratic Intelligence smoke pass;
- Stripe checkout start succeeds;
- webhook delivery succeeds through K3s route after Stripe webhook parity is closed;
- p95/p99 latency and error-rate remain within cutover thresholds.

## Rollback path

Rollback must remain a DNS/edge rollback, not a database rollback.

Rollback steps:

1. restore `signalops.syncratic.io` DNS to the Docker Traefik front door;
2. keep k3s app replicas available for diagnosis or scale them to zero if they are unhealthy;
3. do not delete OpenBao paths, MarketOps databases, Signal-Connect staging broker, or migration evidence;
4. keep Docker Compose/systemd as production authority until post-rollback health is verified;
5. run production browser/API smoke through the restored Compose route;
6. record rollback reason, timestamps, and whether user-facing errors occurred.

## Cutover stop conditions

Abort or rollback immediately if any of the following occur:

- login/callback fails for an existing subscriber;
- token refresh fails during active use;
- `/v1/*` returns elevated 5xx or auth mismatch responses;
- Stripe checkout or webhook verification fails after the Stripe K3s parity gate is in scope;
- Dashboard/Assets/Watchlists core views cannot load for tenant-local or tenant-pilot-b;
- latency or saturation exceeds the approved threshold;
- OpenBao, Keycloak, or database connectivity becomes unstable.

## Explicit non-authorizations

This plan does not authorize:

- production DNS changes;
- public traffic movement;
- disabling Docker Traefik or Compose rollback;
- unsuspending K3s production CronJobs;
- moving scheduler authority from systemd to K3s;
- changing provider polling;
- deleting Docker volumes or databases.

## Evidence state

The plan is considered verified when `scripts/verify_k8s_mesh4_ingress_dns_cutover_plan.sh` passes and the production-readiness report shows `pending_service_mesh_ingress_dns_cutover_plan=false` while retaining `production_cutover_allowed=false`.
