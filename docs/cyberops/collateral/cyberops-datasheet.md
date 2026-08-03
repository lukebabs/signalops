# SignalOps CyberOps

## Deterministic firewall and IoT-flow intelligence for evidence-led security operations

SignalOps CyberOps helps security operators understand what their firewall is actually allowing, identify unusual device behavior as a baseline matures, and keep durable detection evidence separate from human work. It is a specialized SignalOps domain that ingests provider-specific telemetry, normalizes it into reusable contracts, persists provenance, and applies explicit lifecycle policy.

CyberOps is not an automated response tool. It does not change firewall rules, block traffic, page responders, create tickets, or make attribution claims.

## The challenge

Firewall logs are high-volume but hard to use as an analyst surface. A deny is often operational noise; an allow is usually more useful for understanding exposure, device behavior, and service relationships. Without context, a security team can be overwhelmed by raw events, duplicate alerts, or unexplained anomaly scores while still missing a meaningful change in its environment.

CyberOps converts that stream into a bounded, explainable workflow: traffic visibility first, configuration-aware service exposure, learning-progress visibility, durable Signals, and policy-governed Insights and Alerts.

## What CyberOps delivers

| Capability | What it provides | Operator value |
|---|---|---|
| Durable firewall evidence | Raw ingestion and normalized OPNsense filter-log evidence with lineage | A traceable record for investigating observed network behavior |
| Allowed-traffic dashboard | 1-hour, 24-hour, and 7-day trends; live inflow; sources, destinations, protocols, ports, and flows | A direct view of the traffic that is reaching or leaving the environment |
| Live ingestion quality | A streamed chart of received, parsed allowed, and unparsed log volume | Faster recognition of a feed interruption or parser-quality issue |
| IoT/network behavior | Internal-CIDR-scoped device/peer flows, baseline progress, novel services, and threshold observations | A way to understand normal device communication before interpreting a deviation |
| Deterministic anomaly detection | Hourly z-score checks for allowed-log volume and distinct peer count | Explainable deviation detection rather than an opaque anomaly label |
| Service-exposure control | Tenant-owned approved destination IP/protocol/port configuration | Explicit distinction between known approved services and unapproved public exposure |
| Lifecycle noise control | Signals, Insights, and Alerts separated by policy, aggregation episode, and canonical fingerprint | Durable evidence without one work item per repeated firewall event |
| Governance | Tenant isolation, role-aware settings, immutable lifecycle decisions, schedules, and storage visibility | A controlled and inspectable operational model |

## Evidence workflow

1. **Ingest and normalize:** firewall traffic enters SignalOps and is parsed into canonical action, source, destination, protocol, port, and observation-time evidence.
2. **Reveal operational traffic:** the dashboard shows allowed traffic and live stream quality from the primary normalized ledger.
3. **Define the environment:** an authorized operator configures internal CIDRs and approved public services.
4. **Learn normal behavior:** per-device flow evidence builds a visible baseline; devices that lack completed-hour history remain labeled as learning or waiting.
5. **Detect bounded deviations:** a completed hourly observation is compared with the recent device baseline for activity and peer diversity.
6. **Apply lifecycle policy:** valid detection Signals are persisted before policy evaluation; a policy may record evidence only, group an Insight, or create/update an Alert.
7. **Investigate with context:** operators pivot among dashboard, anomaly, Signals, Insights, and Alerts views to see why an item exists.

## Core detection and policy capabilities

- **Allowed-traffic analysis:** focuses on successfully parsed `allow`/`pass` activity and exposes common sources, destinations, protocols, destination ports, and source-to-destination flows.
- **Public-service exposure:** a successfully parsed public-source allowed connection to an explicit destination IP/protocol/port is checked against the tenant’s approved-service register.
- **Port-scan detection:** a public source reaching ten distinct denied destination ports on one destination in five minutes meets the high-severity port-scan policy threshold.
- **IoT behavioral anomaly:** an internal device’s completed-hour allowed-log count or distinct-peer count may be emitted for review when it materially exceeds its recent baseline.

## Signals, Insights, and Alerts are intentionally different

| Layer | Meaning | Why it matters |
|---|---|---|
| Signal | Immutable detector observation and evidence | Valid findings are retained even when no human action is appropriate |
| Insight | Deduplicated, reviewable investigation condition | Repeated evidence updates one bounded episode instead of creating noise |
| Alert | Policy-approved, timely condition | Interruption is reserved for explicit escalation criteria |

The current CyberOps lifecycle policies are deployed in **shadow mode** for the local environment. Shadow mode persists decisions and projected outcomes but does not create new policy-driven Insight or Alert work items. This enables volume and quality review before enforcement.

## Data, cadence, and retention

- The dashboard refreshes traffic views on a short polling interval and maintains a live event stream from the primary normalized ledger.
- The anomaly worker evaluates completed hourly observations. It uses a seven-day lookback, requires at least 24 active baseline hours, and emits review-only observations at or above an absolute 3σ threshold.
- Raw and normalized CyberOps firewall detail and high-resolution hourly features are retained for 30 days. Daily device/service aggregates, anomaly outputs, and lifecycle evidence are retained for 12 months. A compact evidence receipt is preserved before any referenced high-resolution evidence may expire.
- Retention governance currently runs daily in dry-run mode. Deletion requires a separately approved enforcement change and explicit execution.

## Current boundaries

- CyberOps currently supports the implemented OPNsense filter-log workflow; it is not a generalized SIEM ingestion catalog.
- The IoT anomaly method is deterministic z-score detection, not ML classification, threat scoring, or behavioral attribution.
- A baseline that is not mature produces an explicit learning/withheld status rather than a fabricated anomaly result.
- Aggregate firewall observations do not establish business intent, device ownership, vulnerability, compromise, or maliciousness.
- No automatic blocking, rule changes, ticketing, paging, reputation enrichment, or remediation is active.

> **Common reusable disclaimer:** SignalOps CyberOps provides deterministic, evidence-driven security-operations analytics. It does not independently verify maliciousness, provide threat attribution, change security controls, or perform automated remediation. Operators remain responsible for investigation, risk decisions, and response.
