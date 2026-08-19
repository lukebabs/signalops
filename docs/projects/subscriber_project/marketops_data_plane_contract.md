# MarketOps Data-Plane Contract

## Purpose

MarketOps runs as a dedicated operational data plane while SignalOps continues
to host shared platform and CyberOps data. Every MarketOps result must follow
one route:

```text
scheduled acquisition → raw broker topic → continuous MarketOps-aware writer
→ dedicated MarketOps primary / temporal stores → algorithms and UI
```

The scheduler is not an alternate data path. It only acquires or materializes
work; continuous broker consumers own normalized persistence.

## Boundary contract

The protected `/etc/signalops/marketops-cutover.env` renders the complete
dedicated pair:

- `SIGNALOPS_MARKETOPS_DATABASE_URL`
- `SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL`
- `SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED=true`

The normalizer and signal persister retain their shared URLs for non-MarketOps
envelopes, while selecting the dedicated pair for `app_id=marketops`. The SAF
registrar, outbox, and worker select the dedicated pair for all SAF work.

When `SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED=true`, processes fail at
startup if either dedicated URL is absent. A primary-only or temporal-only
configuration is invalid in every environment.

## Deployment and job guardrails

`scripts/deploy_marketops_writer_cutover.sh` now restarts and verifies all
continuous MarketOps writers:

- `normalizer`
- `signal-persister`
- `marketops-signal-assurance-registrar`
- `marketops-signal-assurance-outbox`

It inspects the running containers and refuses completion unless each has the
boundary-required flag and both dedicated routes. A successful Compose command
alone is not deployment evidence.

The root-owned boundary dispatcher sets
`SIGNALOPS_MARKETOPS_DATA_PLANE_PREFLIGHT_REQUIRED=true`. Before any scheduled
job command runs, `scripts/preflight_marketops_data_plane.sh` verifies that the
same four containers are running, have the complete route, and that the two
dedicated databases answer as `marketops` and `marketops_temporal`.

If the contract is broken, the scheduled-job wrapper records a failed job and
raises its normal administrator notification **before any provider call**. It
does not wait for a downstream normalization timeout or attempt a misleading
post-close recovery.

The continuous writers are Compose-managed daemons and must use
`restart: unless-stopped`. A transient PostgreSQL restart or network reset must
not permanently remove a required data-plane writer. Backup and restore
rehearsal controls must also treat the live MarketOps database containers as
pre-existing services: they may verify that the live containers are pgBackRest
capable, but they must not run `docker compose up -d --build` against the live
database services as part of routine backup or isolated restore rehearsal.

The operations monitor actively probes WAL archiving with a bounded
`pg_switch_wal()` before evaluating archive freshness. This prevents low-write
periods, especially in the temporal store, from appearing stale when archive
transport is functional. Non-success systemd results such as `exit-code` are
actionable failures.


## August 19, 2026 recovery-control incident

A controlled `restore-rehearsal-run` restarted the live dedicated MarketOps
PostgreSQL container at `2026-08-19T03:24:37Z`. The
`marketops-signal-assurance-outbox` process had a live database connection,
received a connection reset, exited with code 1, and stayed down because the
container had no restart policy. The first market-session intraday run at
`2026-08-19T13:30:00Z` then failed the data-plane preflight with
`marketops_data_plane_service_missing=marketops-signal-assurance-outbox` before
any provider polling.

Permanent controls added after the incident:

- continuous MarketOps writers use `restart: unless-stopped`;
- backup and restore rehearsal scripts verify live pgBackRest-capable services
  instead of rebuilding/recreating live database containers;
- operations monitor WAL checks actively switch WAL before measuring archive
  freshness;
- operations monitor treats non-success systemd results, including `exit-code`,
  as actionable failures.

## August 17, 2026 incident

The initial dedicated scheduler configuration was correct, but the running
normalizer had only the shared `SIGNALOPS_DATABASE_URL` and
`SIGNALOPS_TEMPORAL_DATABASE_URL`. It consumed 1,034 EOD events successfully
into the shared temporal store while the dedicated temporal store received
none. Warm EOD, post-close/risk-reward recovery, and SRI then failed their
dedicated-normalization barriers. This contract prevents recurrence of that
split route.

## Global Dashboard projection completeness

The subscriber Dashboard reads restricted, platform-global projections rather
than tenant-local algorithm tables. Therefore a completed post-close run is not
complete merely because the primary MarketOps tables contain the new session.

`scripts/marketops_global_dashboard_projection.sh` materializes and verifies,
for its exact completed session date:

- options distributions;
- Risk/Reward snapshots; and
- Market State evidence.

The parity-manifest worker accepts `--session-date YYYY-MM-DD`. The post-close
projection passes that exact date, preventing a bounded newest-first run from
silently consuming an older backlog. It fails when the restricted global
projection has fewer rows than its authoritative tenant-local source.

On 2026-08-18, the catch-up for the Aug 17 completed session appended the
missing immutable Market State evidence and restored 132 of 132 global Dashboard
Market State records. This is append-only global evidence; it does not mutate
the tenant-local source records.
