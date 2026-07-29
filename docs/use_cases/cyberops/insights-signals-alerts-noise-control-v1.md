# CyberOps Signal, Insight, and Alert Lifecycle Noise-Control Specification v1

**Status:** Ready for code-agent implementation and review
**Audience:** SignalOps backend, API, frontend, and operations implementers
**Scope:** CyberOps first; the lifecycle mechanism must remain reusable by other SignalOps use cases.

## 1. Problem and desired outcome

The current signal processor persists every Signal and unconditionally creates one active Insight for it. It creates an Alert for every Signal whose severity is medium, high, or critical. This makes operational traffic indistinguishable from operator-worthy work and is the source of the current Insight noise.

The implementation must separate durable detection evidence from human work. A valid Signal must never be silently discarded, but it must not automatically become an active Insight or Alert.

The required outcome is:

- Signals are durable, queryable detector facts and audit evidence.
- Insights are deduplicated, explainable investigation work items.
- Alerts are exceptional, policy-driven interruptions with a stable dedupe key.
- Every automatic lifecycle outcome is attributable to a versioned policy and can be explained to an operator.
- CyberOps starts with conservative defaults that prefer evidence and aggregation over interruption.

## 2. Normative model

| Layer | Meaning | Operator expectation |
| --- | --- | --- |
| Signal | A valid detector observation or finding. | Retained evidence; not necessarily actionable. |
| Insight | A grouped, reviewable hypothesis or condition. | One work item per entity/fingerprint and episode. |
| Alert | An active condition requiring timely attention. | Interrupting only when an explicit escalation policy permits it. |

Required rules:

1. Persist the Signal before evaluating its lifecycle policy. A bad policy must not lose a valid Signal.
2. A policy decides `record_only`, `create_or_update_insight`, or `create_or_update_alert`; alerts may also retain or update an Insight.
3. A repeated Signal updates a bounded aggregation episode and existing work item; it must not create a new Insight or Alert merely because it was redelivered.
4. Signal IDs remain immutable evidence identities. Insight and Alert IDs are derived from a policy version plus a canonical fingerprint, never from a single raw event ID.
5. A policy decision, including `record_only`, is durably audited with its reason and policy version.

## 3. CyberOps v1 policy contract

### 3.1 Default dispositions

All values below are tenant-scoped defaults, versioned as `cyberops-lifecycle-v1`, and must be configurable without a code redeploy.

| Signal type / condition | Default decision | Aggregation and escalation |
| --- | --- | --- |
| `cyberops.firewall.external_deny` | `record_only` | Count by public source, destination, protocol, and port; no Insight per denied packet. |
| `cyberops.firewall.new_public_service_exposure` for an approved service | `record_only` | Retain evidence and increment the service episode. |
| `cyberops.firewall.new_public_service_exposure` for an unapproved service | `create_or_update_insight` | One low-severity Insight per tenant, destination IP, protocol, and destination port for 30 days or until archived. No automatic Alert. |
| `cyberops.network.port_scan` | `create_or_update_alert` | Require at least 10 distinct destination ports from one public source to one destination in 5 minutes. Create/update one high-severity Alert and linked Insight for 60 minutes. |

### 3.2 Policy semantics

- An approved service is an explicit tenant-owned allow-list entry for destination IP, protocol, and port. Absence means unapproved; there is no implicit approval based on prior traffic.
- `new_public_service_exposure` applies only to successfully parsed OPNsense allow/pass traffic from a public-routable source. Private, loopback, link-local, multicast, and malformed source addresses do not qualify.
- A detector must never treat a firewall deny as proof of maliciousness. The default denies policy is evidence-only while baseline data is collected.
- Port-scan distinctness is based on canonical destination ports, not packet count. Repeated packets to one port cannot meet the threshold.

### 3.3 Canonical fingerprints and bounded evidence

- Exposure fingerprint: `tenant_id|destination_ip|protocol|destination_port`.
- Deny aggregation fingerprint: `tenant_id|source_ip|destination_ip|protocol|destination_port`.
- Port-scan fingerprint: `tenant_id|source_ip|destination_ip`.
- Canonicalize IP addresses, lower-case protocol values, and use integer ports before hashing. Store the canonical components as queryable columns as well as the SHA-256 fingerprint.

## 4. Required processing architecture

1. The signal consumer validates and persists the incoming Signal exactly as it does today.
2. In the same database transaction, it loads the applicable enabled policy, updates the aggregation episode, writes an immutable decision record, and creates or updates the permitted Insight/Alert.
3. If no policy applies, write a `record_only` decision with reason `no_matching_policy`; do not fall back to automatic Insight creation.
4. Use database uniqueness constraints and upserts for the policy/fingerprint/window identity. Do not rely on in-memory detector state, broker retention, or process uptime for deduplication.
5. Commit the broker offset only after that transaction succeeds. Retry-safe duplicate delivery must leave a single episode, decision identity, Insight, and Alert.

**No direct detector write:** detectors publish Signals only. They do not create Insights or Alerts and do not use the lifecycle tables directly.

### 4.1 Persistent lifecycle records

Add migrations and repository contracts for the following records. Typed columns are required for fields used in filtering or uniqueness; JSON may hold bounded contextual evidence only.

- `signal_lifecycle_policies`: policy ID, tenant scope, app/domain/use-case/signal-type/detector selectors, version, enabled state, disposition, severity, typed threshold/window/cooldown settings, canonical fingerprint strategy, created/updated audit fields, and policy document hash.
- `signal_lifecycle_episodes`: tenant, policy ID/version, fingerprint hash and components, window start/end, first/last observed timestamps, occurrence count, distinct-count state, current disposition, linked Insight/Alert IDs, and last Signal ID.
- `signal_lifecycle_decisions`: immutable decision ID, Signal ID, episode ID, policy ID/version/hash, outcome, reason code, observed timestamp, and linked work-item IDs.

Store no unbounded event-ID array in an Insight or Alert. Retain aggregate counts, first/last evidence references, and at most the latest 100 Signal IDs; older evidence remains reachable through the Signal ledger and lifecycle decision query.

## 5. API and operator behavior

### 5.1 Required read surfaces

Each Insight and Alert response must expose enough information to answer “why did this exist?” without reading logs:

- policy ID, version, disposition, and decision reason;
- canonical fingerprint/entity fields, first/last observed timestamps, occurrence count, and threshold/window;
- linked Signal ID(s) and a tenant-scoped link/query route to the underlying Signal evidence;
- approval state for service-exposure Insights, without exposing policy mutation to unauthorised users.

### 5.2 Existing lifecycle API compatibility

Existing Insight and Alert list/get/mutation routes remain supported and tenant-isolated. Add filtered decision and episode read routes, or equivalent fields on the existing detail routes, before changing the frontend.

- Insight mutations retain `active`, `reviewed`, `dismissed`, and `archived` semantics.
- Alert mutations retain `open`, `acknowledged`, `resolved`, and `suppressed` semantics.
- A dismissed/suppressed work item remains closed during its policy cooldown. A material fingerprint change or a later new episode may create a new item.

### 5.3 Authorization and audit

All list, detail, and lifecycle mutation operations require tenant scope and the existing SignalOps authorization model. Policy writes are admin-only and must audit actor, before/after values, reason, and policy hash. Initial policy seeding may be migration/config driven; do not build a broad policy editor in this increment.

## 6. Metrics and observability

- Counters by tenant, detector, signal type, policy version, and decision outcome: received, record-only, insight created/updated, alert created/updated, and policy-miss.
- Gauges for active Insights, open Alerts, episode cardinality, and age of the oldest active item.
- Histograms for Signal-to-decision and Signal-to-work-item latency.
- Error metrics for validation failure, policy evaluation failure, transaction rollback, and duplicate redelivery.
- Metrics and logs must use IDs/fingerprints, never raw Syslog payloads, credentials, or full unredacted event bodies.

## 7. Delivery and migration plan

1. Add schema, repository interfaces, unit tests, and a deterministic policy evaluator behind a feature flag. Preserve the current generic lifecycle behavior until the flag is enabled for CyberOps.
2. Seed the CyberOps v1 defaults in disabled/shadow mode. In shadow mode, persist decisions and metrics but create no new Insights or Alerts; compare projected volume with current volume.
3. Enable the policy path for one test tenant, then `tenant-local`, after the shadow review is accepted.
4. Stop unconditional Insight creation for enabled tenants. Preserve every Signal and decision regardless of outcome.
5. Do not delete or silently rewrite historical Insights/Alerts. Provide a dry-run, auditable archival/suppression migration or operator command for approved historical noise cleanup.

## 8. Acceptance criteria

1. A valid low-severity CyberOps Signal with no matching policy is persisted with a `record_only/no_matching_policy` decision and creates neither Insight nor Alert.
2. One hundred redeliveries of the same Signal result in one evidence identity and one lifecycle episode/work item; counts may advance only according to the configured aggregation semantics.
3. An approved exposure produces no active Insight; an unapproved exposure produces exactly one low Insight for its fingerprint within the 30-day cooldown and never automatically opens an Alert.
4. Nine distinct public-source ports in five minutes do not alert; the tenth creates one high Alert and linked Insight. Further matching traffic updates that episode rather than multiplying work items.
5. Restarting the detector and signal consumer, including loss of in-memory state, does not bypass the database cooldown or create duplicate work.
6. A database transaction failure creates neither a partial episode nor a work item and does not acknowledge the broker record.
7. Tenant A cannot read, mutate, deduplicate against, or approve service exposure state belonging to Tenant B.

## 9. Non-goals and guardrails

- No automatic blocking, firewall changes, ticket creation, paging integration, reputation feeds, or ML classification in this increment.
- No assumption that all denied traffic is hostile or that all public traffic is a vulnerability.
- No frontend redesign before the API exposes policy explanations and bounded evidence.
- No production-wide enablement without shadow-volume evidence and explicit operator acceptance.

## 10. Likely implementation areas

- `internal/signals/processor.go`: replace unconditional `lifecycleRecords` behavior with policy evaluation and transactional persistence.
- `internal/cyberops/detection`: emit stable typed Signals and preserve parser/source evidence; do not own lifecycle state.
- `internal/storage` and `internal/storage/postgres`: lifecycle types, migrations, transactional upserts, query filters, and tenant-safe uniqueness.
- `internal/api`: detail/list decision evidence, authorization, and compatibility tests for existing Insight/Alert routes.
- Migrations and operational docs: policy seed, shadow enablement, rollback, and auditable historical cleanup.

## 11. Required test coverage

- Table-driven policy evaluator tests for selectors, thresholds, cooldowns, canonical fingerprints, and material changes.
- Postgres integration tests for concurrent consumers, duplicate delivery, window boundaries, transaction rollback, and tenant isolation.
- API tests for explanation fields, pagination/filtering, authorization, and lifecycle mutations.
- End-to-end CyberOps fixtures for parsed allow, parsed deny, malformed records, private sources, approved service, unapproved service, and port-scan threshold crossing.

## 12. Definition of done

The code agent must deliver migrations, implementation, tests, runbook updates, and an acceptance report showing before/after Insight and Alert volume for the CyberOps test tenant. It must state any deliberate deviation from this specification and must not enable policy enforcement outside the approved tenant without explicit approval.
