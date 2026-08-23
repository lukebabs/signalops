# SignalOps Retention Governance

SignalOps uses metadata-first retention. It is not a SIEM archive: provider and firewall payloads are short-lived operational data, while deterministic inputs, provenance, compact finding evidence, and derived outputs remain available for algorithm integrity.

## Policy baseline

| Scope | Raw/high-resolution | Preserved metadata |
|---|---:|---:|
| MarketOps provider payloads | 30 days | Canonical EOD, features, states, scores, and outcomes: 12 months |
| MarketOps option-chain detail | 92 days | Derived option distributions and results: 12 months |
| MarketOps financial facts/snapshots | n/a | 4 years |
| CyberOps raw/normalized firewall detail | 30 days | Daily device/service aggregates, anomaly/lifecycle outputs: 12 months |
| CyberOps hourly flow features | 30 days | Daily flow aggregates: 12 months |
| Platform idempotency | 35 days | n/a |

The existing 400-calendar-day MarketOps evaluator must be constrained to a 12-month metadata horizon before the corresponding expiry policy may be enforced.

## Operation

`signalops-retention-governance.timer` runs daily at 02:30 ET for the shared SignalOps database. It first materializes CyberOps daily features, then runs `retention-governor` in dry-run mode. The run records candidate counts, temporal bounds, policy version, and preservation rule in `retention_runs`; no record is deleted by this schedule.

After the MarketOps database boundary cutover, subscriber and MarketOps-specific retention policies must run through the dedicated MarketOps path, not the shared SignalOps retention job. The allowlisted job is `marketops-retention-governance`, backed by `scripts/run_marketops_retention_governance.sh` and `signalops-marketops-retention-governance.service`. It currently dry-runs `subscriber.user_activity_180d` for `tenant-local` and `tenant-pilot-b` against the dedicated MarketOps primary database.

An operator can inspect every policy and its latest dry-run result in **Administration → Storage → Retention governance**. Enforcement requires a separate approved change to set a policy's `mode` to `enforced`, followed by an explicit `retention-governor --execute` run. There is no bulk/global purge command.

Before CyberOps normalized firewall detail can expire, the governor preserves compact receipts for events referenced by a derived signal. Receipts contain event identity, hash, source/dataset, time, parser/version metadata, and quality metadata; they never retain the original firewall message.

Broker topic retention remains disabled in this release because it requires a broker-administration adapter that can verify terminal consumer/replay state before changing topic configuration.
