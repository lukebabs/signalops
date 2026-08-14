# MarketOps Dedicated Database Boundary

Status: bootstrap/parity evidence, corrected gateway-read acceptance, and the continuous-writer cutover are complete. Scheduled-job routing/resume, dedicated pgBackRest schedules, and restore rehearsal remain separate gates.

## Decision

MarketOps recovery must not include CyberOps operational data. The approved
target is two distinct PostgreSQL clusters/volumes, not an additional database
inside the existing shared cluster:

| Store | Owns | Excludes |
| --- | --- | --- |
| `marketops-postgres` | MarketOps operational tables, SAF/SRI, algorithms, and filtered MarketOps rows in the primary ledgers | CyberOps event/outbox/lifecycle/IoT data; non-MarketOps ledger rows; the live subscriber control plane |
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
`compose.marketops-writer-cutover.yaml` attaches it to the continuous MarketOps writers (normalizer, signal persister, SAF registrar, and SAF outbox). It does not enable or reroute any paused scheduled batch job; that is a separate schedule-resume gate.

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

## Corrected gateway read-cutover acceptance evidence — 2026-08-14

Release `3132b75` redeployed the gateway using the protected production Compose
environment file. The gateway was verified with authentication enabled and
subscriber lists enabled for `tenant-pilot-b`. The fresh
`signalops.syncratic.io-testsignal-01.har` contains 94 requests and zero failed
requests. Its pilot-tenant MarketOps, watchlist-context, subscriber-list, and
subscriber-list-items calls all returned `200`; the tenant context was
consistently `tenant-pilot-b`.

The normalizer and signal persister retain their pre-cutover start times, and
all seven MarketOps timers remain `inactive`. Accordingly, the corrected
acceptance is limited to gateway reads; the writer cutover and schedule resume
are still pending their separate gates.

## Writer preflight reconciliation — 2026-08-14

The read-only `scripts/preflight_marketops_writer_cutover.sh` compared every scoped primary and temporal MarketOps table while excluding `subscriber_*`. The subscriber catalog, watchlists, tenant grants, and their audit/RLS controls remain on the central shared subscriber gateway database; their bootstrap copy in the boundary is not a live source of truth.

The first preflight found one bounded post-bootstrap delta: 12 State Street ETF holdings snapshots (effective 2026-08-12, retrieved 2026-08-14) and their 700 child holdings. Those exact parent/child rows were reconciled to the dedicated primary store in a single target transaction. The rerun passed all scoped counts, including 24,422 normalized events and 139,442 signals in the temporal store, and confirmed zero non-MarketOps ledger rows in both dedicated stores.

This authorizes only the next continuous-writer gate. The paused scheduled batch jobs still require explicit launch-environment routing before any timer can be resumed.

## Continuous-writer cutover evidence — 2026-08-14

Release `353a541` was deployed from the clean cutover worktree after the final parity gate. Normalizer, signal persister, and the SAF registrar were stopped briefly and rebuilt with the protected dedicated MarketOps URLs. Each replacement container is running with `SIGNALOPS_MARKETOPS_DATABASE_URL` present; the registrar logged that its writes are routed to the dedicated MarketOps boundary.

The immediate post-deployment preflight passed every scoped primary and temporal count, including the reconciled 2,800 SRI holdings and 48 snapshots. It again found zero non-MarketOps normalized-event or signal-ledger rows in either dedicated store. The gateway remains on its already accepted dedicated read path. The seven scheduler units were found enabled in the user-level systemd scope despite being inactive; they were explicitly stopped and disabled. The writer-cutover guard now verifies both system and user scopes. No batch job was enabled or rerouted by this release.

## Scheduled-job routing preparation — 2026-08-14

`compose.marketops-scheduled-cutover.yaml` is a dormant, validated overlay for MarketOps batch runners. The system-managed `signalops-marketops-boundary-schedule@.service` template receives the protected environment from systemd, runs as the existing Docker-capable service user, and uses the overlay to replace runner standard primary and temporal URLs with the dedicated MarketOps URLs. Direct completion queries in the daily, intraday, retry, recovery, and universal-completion scripts use explicit dedicated database helpers. It does not alter normalizer, signal persister, the gateway, subscriber control-plane services, or CyberOps.

The existing user-level timers were explicitly disabled after discovery that system-level checks did not cover them. `scripts/install_marketops_boundary_scheduler.sh` installs only the disabled dispatcher unit; it does not enable a timer. No timer is to be re-enabled until a separately approved one-job smoke test passes. The SRI refresh and holdings jobs remain outside this dispatcher because their in-progress implementation is not part of this clean release.

## Scheduler dispatcher preflight evidence — 2026-08-14

The disabled system-managed dispatcher was installed from the clean release and started as the `preflight` instance. It completed successfully and logged `Dedicated scheduler preflight passed: primary=marketops temporal=marketops_temporal`. The host emitted a pre-existing `Too many open files` directory-watch warning, but systemd recorded exit code 0 and result `success`; the warning did not prevent execution.

Immediately afterward, the dedicated tactical retry query returned zero due records and the legacy user timer inventory contained no scheduled MarketOps timer. A one-shot retry-dispatcher smoke is therefore provider-free while that queue remains empty; it remains a separate execution step and does not re-enable any timer.

## Scheduler no-op retry smoke evidence — 2026-08-14

The dedicated `marketops-task-retry` dispatcher instance ran after the preflight. It completed with systemd result `success` and exit code 0, logging `no due tactical retries for 2026-08-12`. This proves the installed system-managed unit can invoke a real scheduled script, reach the dedicated query path, and terminate cleanly without a provider call or a MarketOps data mutation when no retry is eligible. Legacy user timers remained unscheduled.

## SRI gateway and scheduler release evidence — 2026-08-14

Release `8da1855` rebuilt and restarted only the gateway and web containers from a fresh worktree. Authentication, subscriber lists, and the dedicated MarketOps gateway URL remained configured; the gateway logged its dedicated-read route. The new ETF-makeup route responds with the expected authentication challenge before a session is supplied. The continuous writers were not restarted and all scheduled timers remained disabled.

After active dedicated writers produced 261 new MarketOps temporal signals, strict source/target equality is no longer the correct post-cutover criterion. The reconciliation gate now has an explicit `--dedicated-authoritative` mode: every former shared count must remain present or be exceeded in the dedicated store, and cross-workload ledger rows must remain zero. That gate passed, recording 139,703 dedicated MarketOps temporal signals versus the frozen shared baseline of 139,442.

## Cutover gates

1. Bootstrap evidence passes, including table counts and database sizes.
2. Add dedicated MarketOps repository wiring to the gateway and continuous MarketOps writers; retain shared-store reads during parity validation. Route paused scheduled jobs separately before any timer resume.
3. Reconcile source versus target checksums/counts and API responses for the
   approved tenant/watchlist cohort.
4. Freeze MarketOps writers briefly, run final delta copy, route MarketOps
   readers/writers to the dedicated stores, then validate no CyberOps workload
   reaches either target.
5. Configure a distinct pgBackRest stanza and S3 prefix for **each** dedicated
   cluster, enable their timers, and complete an isolated restore rehearsal.
6. Retain the existing shared-cluster backup as a separate platform safety
   baseline; do not use it as MarketOps recovery evidence.

## Approved State Street issuer-holdings refresh evidence — 2026-08-14

With the named approval of `luke@strategiclabs.io`, the disabled dedicated scheduler ran exactly one `marketops-sri-holdings-refresh` for `tenant-local`. It completed successfully with no retry (`systemd` result `success`, exit code `0`): 12 supported State Street primary ETFs were evaluated, producing 12 snapshot candidates and 700 holding candidates; four non-State-Street primary ETFs were reported unsupported.

State Street returned the same effective-date/content hashes already stored for all 12 supported ETFs. The immutable snapshot and holding upserts therefore correctly resolved as no-ops: the dedicated store remains at 48 State Street snapshots, latest retrieved timestamp `2026-08-14 00:20:03 UTC`. No duplicate or mutable historical holding rows were created.

The run log showed the legacy `signalops-postgres-1` only because Compose inherited its base dependency health check. The scheduler overlay supplied the dedicated primary URL to the runner, and neither the shared nor dedicated snapshot row set changed because the source content was unchanged. The runner is now invoked with `--no-deps` so future dedicated runs do not start or log that legacy dependency; this improves boundary evidence without changing provider or data behavior. No timer was enabled.

## Controlled recurring SRI schedule — approved 2026-08-14

The dedicated scheduler installer now supports an explicit `--enable-sri` action, used only with recorded approval. It installs two system-managed, persistent timers that invoke the dedicated dispatcher: the SRI ETF price/reconciliation job at 20:07 America/New_York every weekday, followed by the State Street issuer-holdings job at 20:20 America/New_York. Both jobs inherit the protected dedicated MarketOps URLs; they do not reactivate legacy user timers or alter CyberOps or subscriber control-plane processing.

The one-minute scheduling accuracy intentionally avoids a fragile exact-second dependency while preserving ordering. Each service remains a separately observable one-shot unit, and no retry loop is added by these timers.
