# MarketOps Dedicated Database Boundary

Status: bootstrap and parity verification completed 2026-08-14; application cutover, dedicated pgBackRest schedules, and restore rehearsal remain separate approvals.

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
