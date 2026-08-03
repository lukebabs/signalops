# CyberOps Detection and Evidence Reference

## Operating principles

CyberOps processes security telemetry as durable evidence before it creates human work. A parser/detector publishes a Signal; lifecycle policy then determines whether the evidence is record-only, supports an Insight, or warrants an Alert. This ordering preserves valid evidence even if a policy is changed, disabled, or misconfigured.

All scope is tenant-isolated. `tenant_id` is an authorization and data-isolation boundary; CyberOps uses `app_id=cyberops`, `domain=security`, and `use_case=cyberops` within the tenant.

## Current source and normalized evidence

The current workflow consumes OPNsense filter-log traffic. A successful parse provides the firewall action, source IP, destination IP, protocol, destination port, and observation time. The primary normalized ledger is the source of truth for the live dashboard; temporal replication supports historical/backfill workflows and is not required for the live display.

The parser distinguishes `pass`/`allow` and `block`/`deny` events. Malformed/unparsed messages remain visible through ingestion-quality measures rather than being silently converted to a security conclusion.

## Allowed-traffic and service-exposure evidence

### Allowed traffic

The traffic dashboard aggregates successfully parsed allowed events by time, source, destination, protocol, destination port, and flow. These are visibility and investigation features, not a service inventory or an approval decision.

### New public service exposure

This condition applies only when a successfully parsed allowed/pass event comes from a public-routable source. Private, loopback, link-local, multicast, and malformed sources do not qualify.

An approved service is an explicit tenant-owned tuple of destination IP, protocol, and port. The canonical exposure fingerprint is:

`tenant_id|destination_ip|protocol|destination_port`

Absence of that configuration means unapproved for the policy; it does not, on its own, mean vulnerable, compromised, malicious, or prohibited.

## Deny and port-scan evidence

### External deny

The canonical deny aggregation fingerprint is:

`tenant_id|source_ip|destination_ip|protocol|destination_port`

External denies are durable evidence. The default policy does not interpret a denied packet as proof of an attack or create an Insight per packet.

### Public-source port scan

The canonical port-scan fingerprint is:

`tenant_id|source_ip|destination_ip`

The current policy threshold is at least ten **distinct destination ports** from one public source to one destination in five minutes. Repeated packets to one port do not advance the distinct-port threshold. At threshold, policy projects a high-severity Alert with a linked Insight for a bounded 60-minute episode.

## IoT/network behavior and deterministic anomaly method

**Algorithm ID:** `signalops.algorithms.zscore_anomaly_v1`  
**Mode:** hourly IoT anomaly observation

Internal-network scope is configured by tenant CIDRs. For every configured internal device, CyberOps evaluates the preceding completed hour only; it does not score the partially open current hour.

| Metric | Observed value | Baseline | Detection rule |
|---|---|---|---|
| `allowed_log_count` | Allowed firewall events for the device in the completed hour | Hourly values across the preceding seven days | Emit when absolute z-score is at least 3 |
| `distinct_peers` | Unique communicating peers in the completed hour | Hourly peer-count values across the preceding seven days | Emit when absolute z-score is at least 3 |

The worker requires a device to have at least 24 non-zero active baseline hours and an observation in the target hour. If standard deviation is zero, the result is withheld rather than treated as an infinite or arbitrary score. Severity is medium at 3σ and high at 5σ. Confidence is bounded from the z-score and no emitted result creates a Signal, Insight, or Alert by itself.

Each emitted result preserves the device, metric, observed value, baseline mean, baseline standard deviation, baseline-hour count, completed hour, algorithm version, and source-event references.

**Interpretation boundary:** a z-score is a statistical difference from the observed recent baseline. It does not infer malware, command-and-control, compromise, asset criticality, or business intent.

## Lifecycle policy and noise control

| Condition | Policy disposition | Aggregation behavior |
|---|---|---|
| External deny | `record_only` | Counted by public source/destination/protocol/port; no work item per packet |
| Approved public service exposure | `record_only` | Evidence retained and service episode updated |
| Unapproved public service exposure | `create_or_update_insight` | One low-severity Insight per exposure fingerprint for 30 days or until archived |
| Port scan | `create_or_update_alert` | One high-severity Alert and linked Insight per fingerprint for 60 minutes |

The local CyberOps seed is presently `shadow` mode. It records immutable policy decisions and episodes, but does not create new policy-driven Insights or Alerts. Moving to enforcement requires an approved operator decision after projected-volume review; it is not an automatic product behavior.

Signal IDs are immutable evidence identities. Insights and Alerts are derived from policy version plus canonical fingerprint, not raw event IDs. Evidence is bounded in work items: counts, first/last references, and at most 100 recent Signal IDs are stored there; broader evidence remains reachable through the Signal ledger and lifecycle decisions.

## Explainability and safeguards

Every lifecycle decision records the policy ID/version/hash, disposition, reason, observed time, episode, and linked work-item identity where applicable. Evaluation and persistence occur transactionally; broker acknowledgement follows successful persistence so retry/redelivery does not multiply the episode or work item.

CyberOps does not currently provide automatic blocking, firewall changes, ticket creation, paging, reputation feeds, ML classification, or automated remediation. It also does not claim that its current OPNsense-derived telemetry is comprehensive security coverage.

## Data lifecycle

- Raw and normalized firewall detail: 30 days.
- High-resolution hourly flow features: 30 days.
- Daily device/service aggregates, anomaly outputs, lifecycle results, and compact evidence receipts: 12 months.
- Idempotency records: 35 days.

Retention governance runs daily in dry-run mode. Before referenced high-resolution firewall evidence is eligible for expiry, the governor preserves a compact receipt containing identity, hash, source/dataset, time, parser/version, and quality metadata—not the original firewall message.
