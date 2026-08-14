# MarketOps Dedicated Database Boundary

Status: bootstrap/parity evidence and the read-cutover release are complete. Production gateway deployment, writer cutover, dedicated pgBackRest schedules, and restore rehearsal remain separate approvals.

## Decision

MarketOps recovery must not include CyberOps operational data. The approved
target is two distinct PostgreSQL clusters/volumes, not an additional database
inside the existing shared cluster:

| Store | Owns | Excludes |
| --- | --- | --- |
| `marketops-postgres` | MarketOps operational tables, SAF/SRI, algorithms, MarketOps subscriber catalog/control data, and filtered MarketOps rows in the primary ledgers | CyberOps event/outbox/lifecycle/IoT data; non-MarketOps ledger rows |
| `marketops-timescaledb` | MarketOps EOD prices, options history, and filtered MarketOps temporal ledgers | CyberOps and console temporal-ledger rows |

The shared `signalops` and `signalops_temporal` stores remain authoritative
until a separately approved dual-read/write cutover. No source rows are
deleted by boundary bootstrap.

## Inventory captured 2026-08-13

The shared primary database is 55.8 GB. Its largest CyberOps tables are
`normalized_event_ledger` (18.0 GB), `signal_ledger` (13.1 GB),
`cyberops_connect_raw_events` (12.5 GB), and
`cyberops_connect_outbox` (8.4 GB). The large shared primary ledgers contain
only 14 MarketOps normalized events and 2,778 MarketOps signals.

The shared temporal store is 2.56 GB and is predominantly MarketOps: it has
24,398 MarketOps normalized events and 139,441 MarketOps signals. The boundary
therefore copies its MarketOps data plus MarketOps EOD/options tables, while
filtering both shared temporal ledgers by `app_id='marketops'`.

## Bootstrap

The additive Compose definition is [compose.marketops-boundary.yaml](../../../compose.marketops-boundary.yaml).
It requires independently injected secrets; do not add these to `.env`. The root-only provisioner installs the handoff into `/etc/signalops/marketops-boundary.env` mode `0600`, which is the only persistent location for the two database passwords.

```bash
sudo ./scripts/provision_marketops_database_boundary.sh \
  /tmp/signalops-marketops-boundary-secrets.env
```

The script creates only the new target services, applies the current schema,
copies the explicit MarketOps tables, binary-copies only `app_id='marketops'`
rows from shared ledgers, then proves target exclusion of CyberOps and
non-MarketOps ledger data. It does not change any application environment,
scheduled job, S3 backup, or source data.

## Bootstrap evidence — 2026-08-14

The additive bootstrap completed without changing the shared source stores or application routing. The source and dedicated primary stores each exposed 123 scoped tables with zero per-table row-count mismatches. The dedicated temporal store matched 24,422 MarketOps normalized events and 139,442 MarketOps signals. Both source and target had zero rows in the current `marketdata_equity_eod_prices` and `marketdata_option_contracts_daily` hypertable roots.

The resulting dedicated physical stores are 914 MB (`marketops-postgres`) and 1007 MB (`marketops-timescaledb`). Boundary verification proved zero CyberOps rows and zero non-MarketOps ledger rows in either target. Both the shared and dedicated primary databases have migrations `000116_platform_primitive_audit_schema_qualification` and `000117_platform_primitive_policy_schema_qualification` applied.

## Phased application cutover

Release `d606d06` adds an optional dedicated MarketOps repository to the
gateway. When `SIGNALOPS_MARKETOPS_DATABASE_URL` is present, MarketOps-only
routes use it; access control, CyberOps, platform administration, generic
alerts/algorithms, Syncratic, and publishing remain on the shared repository.
This prevents a broad repository swap from exposing CyberOps to the MarketOps
store. The configuration is backwards compatible: without the variables every
route remains on the existing shared stores.

The root-only renderer derives the two URLs from the existing protected
boundary secret and writes `/etc/signalops/marketops-cutover.env` mode `0600`.
Neither the passwords nor the rendered file belong in Git or `.env`.

```bash
sudo ./scripts/render_marketops_cutover_env.sh
```

`compose.marketops-read-cutover.yaml` attaches that file only to `gateway`.
`compose.marketops-writer-cutover.yaml` attaches it only to `normalizer` and
`signal-persister`; it is a later, separate phase. Do not apply both overrides
in the same first deployment.

Because the active working tree contains unrelated SRI work, build the gateway
from the pushed release in a clean worktree. This avoids accidentally coupling
the database cutover to unreviewed changes:

```bash
git fetch origin subscribers
git worktree add --detach /tmp/signalops-marketops-read-cutover origin/subscribers
cd /tmp/signalops-marketops-read-cutover
sudo ./scripts/deploy_marketops_read_cutover.sh /home/adminalien/docker/syncratic-core/subsystems/signalops/.env
```

The wrapper requires the protected production Compose environment file so the replacement gateway preserves its authentication, subscriber-list, and other existing runtime settings. It regenerates the root-owned cutover environment and loads the boundary passwords only while Compose resolves its service definitions. This is necessary because Compose interpolates those definitions even when it is starting only the gateway.

Validate the gateway health endpoint, the protected MarketOps asset and SAF
views for the approved tenant/watchlist cohort, and a non-MarketOps/CyberOps
route. Keep all MarketOps timers stopped until those checks pass. Rollback is
one gateway redeploy with the read override omitted; the shared databases have
not been mutated by the read phase.

## Initial gateway read-cutover observability — 2026-08-14

Do not treat this initial deployment as acceptance evidence. The clean worktree lacked the protected production Compose environment file, so its replacement gateway started with authentication and subscriber-list features disabled. The supplied tenant-pilot-b HAR caught the regression through `404` responses for both subscriber watchlist endpoints. Redeploy using `deploy_marketops_read_cutover.sh` with the protected production environment file before further validation.

The initial clean release `e913ce0` was built and deployed to `signalops-gateway-1`
with the read-only override. Its startup log states that MarketOps gateway reads
are routed to the dedicated data boundary. `GET /healthz` and `GET /readyz`
returned `200`. The dedicated MarketOps asset endpoint returned 132 assets for
`tenant-local`; the SAF effectiveness endpoint returned `200` (925 bytes).

The shared-plane controls remained available: catalog sources returned `200`
(553 bytes) and CyberOps events returned `200` (446 bytes). The normalizer and
signal persister containers retained their pre-cutover start times, and all
seven paused MarketOps timers were confirmed `inactive`. This is gateway-read
evidence only; no MarketOps writer has been cut over or resumed.

## Cutover gates

1. Bootstrap evidence passes, including table counts and database sizes.
2. Add dedicated MarketOps repository wiring to the gateway and every
   MarketOps worker; retain shared-store reads during parity validation.
3. Reconcile source versus target checksums/counts and API responses for the
   approved tenant/watchlist cohort.
4. Freeze MarketOps writers briefly, run final delta copy, route MarketOps
   readers/writers to the dedicated stores, then validate no CyberOps workload
   reaches either target.
5. Configure a distinct pgBackRest stanza and S3 prefix for **each** dedicated
   cluster, enable their timers, and complete an isolated restore rehearsal.
6. Retain the existing shared-cluster backup as a separate platform safety
   baseline; do not use it as MarketOps recovery evidence.
