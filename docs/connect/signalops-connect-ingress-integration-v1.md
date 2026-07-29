# SignalOps–Connect Ingress Integration v1

## Purpose

This specification defines how SignalOps integrates with Syncratic Connect. Connect is the authoritative ingress boundary for external operational sources. SignalOps consumes Connect's durable `RawSignalEvent v1` handoff; it does not receive UDP, Syslog, webhooks, or source credentials directly.

CyberOps RFC 5424 Syslog is the reference implementation. It proves the integration with OPNsense-originated UDP Syslog while preserving raw evidence and allowing later vendor-specific enrichment.

## Scope

In scope:

- Consume Connect-produced `RawSignalEvent v1` records from Redpanda/Kafka.
- Validate, persist, query, and normalize CyberOps raw events idempotently.
- Preserve Connect lineage on all SignalOps records and derived artifacts.
- Provide a security-domain model and initial queries over raw CyberOps event attributes.

Out of scope:

- Direct Syslog, UDP, TCP/TLS Syslog, webhook, or credential handling in SignalOps.
- Reimplementation of Connect source CIDR filtering, RFC 5424 parsing, admission, replay, or payload retention.
- Claims of OPNsense firewall semantics beyond explicitly versioned parsers.

## Architectural Boundary

```text
External source → Syncratic Connect → Redpanda topic → SignalOps consumer → SignalOps storage/query/derivation
```

Connect owns source trust, channel configuration, producer resolution, protocol decoding, durable ingress, and ingress idempotency. SignalOps owns its consumer offset, validation, durable business persistence, domain queries, normalization, findings, alerts, and user-facing workflows.

SignalOps MUST NOT access Connect PostgreSQL, OpenBao secrets, or raw network traffic. Connect metadata is carried only in the event and broker headers.

## Transport and Contract

For local development, Connect publishes to `signalops.local.raw.v1`. Production topic selection follows the existing `signalops.<environment>.raw.v1` convention. The record value conforms to the established SignalOps `RawSignalEvent v1` schema.

SignalOps MUST validate the event schema before persistence. It MUST reject or route invalid records using its existing retry/DLQ policy and MUST NOT acknowledge the consumer message until the durable idempotent write succeeds.

Required CyberOps values:

- `source_domain`: `security`
- `event_type`: `cyberops.syslog.raw`
- `source_adapter`: `connect:<connector_id>`
- `ingestion_mode`: `push_event`
- `metadata.connect.protocol_key`: `syslog-rfc5424`
- `metadata.connect.protocol_version`: `1.0.0`

## CyberOps Reference Payload

Connect maps RFC 5424 into `payload` without vendor semantic normalization:

| Payload path | Meaning |
| --- | --- |
| `source.hostname` | RFC 5424 hostname |
| `source.application` | RFC 5424 app name |
| `message` | Raw Syslog message text |
| `occurred_at` | RFC 5424 timestamp |
| `syslog.severity` | RFC 5424 severity integer |
| `syslog.facility` | RFC 5424 facility integer |

SignalOps MUST preserve these values verbatim in raw-event storage. Derived fields may be added only by an explicit, versioned parser and must retain a pointer to the raw event and parser version.

## Lineage and Idempotency

SignalOps MUST store the complete `metadata.connect` object. At minimum it includes ingress event, connector, connector version, channel, producer, protocol, mapping, dataset binding, payload hash, processing run, and delivery identifiers.

The durable business idempotency key is `(tenant_id, metadata.connect.ingress_event_id)`. If the same identity arrives with identical immutable content, the write is a no-op success. If it arrives with conflicting content or lineage, SignalOps MUST record an integrity failure and prevent silent overwrite.

Broker delivery identity and consumer offsets are operational metadata; they do not replace Connect ingress identity.

## SignalOps Data Model

The implementation MUST create or extend a tenant-scoped raw-event projection supporting:


- time-range queries using `occurred_at` and ingestion time;
- hostname, application, severity, and facility filters;
- message full-text search;
- `event_type`, producer, connector, and channel filters;
- direct lookup by Connect ingress event ID.

Raw event, normalized security observation, finding, correlation, and alert are distinct entities. A finding or alert MUST reference the raw event(s) and the rule/parser version that produced it.

## Data Model and Persistence

The implementation SHOULD extend the existing SignalOps raw-event persistence path before creating a CyberOps-only storage system. If the existing raw-event ledger already persists the complete event document and supports tenant-scoped indexing, add the required CyberOps indexes/projections there. Otherwise create a migration-owned projection with the following minimum fields:

| Field | Requirement |
| --- | --- |
| `tenant_id` | Required; part of every key and query predicate. |
| `connect_ingress_event_id` | Required; immutable business idempotency identity. |
| `event_id` | SignalOps event identity; retain the Connect ingress value when compatible. |
| `event_type` | Required; initially `cyberops.syslog.raw`. |
| `occurred_at`, `ingested_at` | Required timestamps for event-time and processing-time queries. |
| `hostname`, `application` | Nullable extracted query columns copied from raw payload. |
| `severity`, `facility` | Nullable integer query columns copied from raw payload. |
| `message` | Required raw message text; index for bounded full-text search. |
| `raw_event` | Required immutable RawSignalEvent v1 JSON document. |
| `connect_metadata` | Required immutable `metadata.connect` JSON document. |
| `payload_hash` | Required integrity value from Connect metadata. |
| `created_at` | Required persistence timestamp. |

Create a unique constraint on `(tenant_id, connect_ingress_event_id)`. Add tenant-leading indexes for event-time queries and the initial filter set. Do not mutate the raw JSON document after insertion. Parser or detector output belongs in separately versioned derived tables.

## Consumer Design

Use the established SignalOps Kafka/Redpanda client and canonical raw-event consumer path. Do not introduce a second broker abstraction or a consumer that bypasses existing retry/DLQ controls.

For each broker message:

1. Decode the message value as RawSignalEvent v1 and validate it against the existing schema.
2. Require `source_domain=security`, `event_type=cyberops.syslog.raw`, and complete Connect metadata for the CyberOps route.
3. Check the protocol binding is exactly `syslog-rfc5424@1.0.0`.
4. Extract the allowed query projection from `payload`; preserve the original document even when optional fields are absent.
5. In one database transaction, insert the raw event or resolve the idempotent existing row.
6. Commit the broker offset only after the transaction commits or an identical duplicate is confirmed.
7. Route schema, lineage, or integrity failures through the established non-retryable failure/DLQ mechanism. Retry only failures classified as transient.

The consumer group name must be stable, environment-aware, and documented. It may be a CyberOps-specific group only if that does not duplicate the canonical raw-event persistence responsibility.

## APIs and Query Behavior

Expose CyberOps only through the existing authenticated, tenant-scoped SignalOps API conventions. At minimum provide:

- `GET /v1/cyberops/events` with bounded `from` and `to` range, pagination, hostname, application, severity, facility, producer, connector, and event-type filters.
- `GET /v1/cyberops/events/{connect_ingress_event_id}` returning the immutable raw record, Connect metadata, and links/counts for derived artifacts.
- A bounded message-search parameter with explicit maximum length and time-range requirement.

Every list and detail query MUST derive tenant scope from the authenticated principal, never from a caller-supplied tenant identifier. Responses must preserve evidence: raw payload, source metadata, timestamps, and Connect lineage must remain available to authorized operators.

## Failure and Integrity Behavior

- Duplicate with same tenant, ingress identity, payload hash, and immutable lineage: acknowledge as idempotent success.
- Duplicate with conflicting hash, event body, or lineage: persist/audit an integrity failure and do not overwrite the accepted record.
- Invalid schema, missing required Connect metadata, unsupported protocol binding, or invalid CyberOps projection: non-retryable failure and DLQ/audit evidence.
- Database/network outage: retryable failure; do not commit the offset.
- Unknown source domain or event type: use the existing routing policy; do not silently coerce it into CyberOps.
- Parser/enrichment failure after raw persistence: preserve raw event and record the derived-stage failure separately; it must not destroy ingestion evidence.

## Security and Retention

SignalOps must treat `message` and structured raw values as potentially sensitive security telemetry. Reuse existing tenant authorization, audit logging, encryption, retention, and redaction controls. Any display redaction must not mutate the evidence record. Access to raw message text should follow the same authorization level as other sensitive operational evidence.

## Tests

The code agent MUST add focused tests for:

- RawSignalEvent v1 schema acceptance with a sanitized CyberOps RFC5424 fixture.
- Rejection of missing or malformed Connect metadata.
- Rejection of unsupported Syslog protocol key/version.
- Durable idempotency for duplicate delivery and integrity failure for conflicting duplicate delivery.
- Tenant isolation for list and detail APIs.
- Projection/query behavior for time, hostname, application, severity, facility, and message search.
- Offset commit only after durable success; retry/DLQ behavior for transient and non-retryable failures.
- Preservation of raw payload and metadata through any normalization or derived-artifact path.

Use unit tests for validation and idempotency, repository/integration tests for migration constraints and queries, and one live Compose/Redpanda acceptance test when the local environment supports it.

## Rollout Plan

1. Create a dedicated branch or worktree; do not mix CyberOps work with unrelated MarketOps changes.
2. Implement migrations, repository methods, consumer routing, API handlers, and tests behind the existing authentication and broker abstractions.
3. Deploy with CyberOps consumer enabled but without parser-derived alerts.
4. Publish one sanitized Connect-produced `cyberops.syslog.raw` event into the local raw topic.
5. Verify consumer group health, persisted row, query API result, Connect lineage, duplicate behavior, and zero unintended side effects.
6. Enable any parser/enrichment only after raw-event acceptance is signed off and parser provenance is versioned.

## Acceptance Criteria

The implementation is accepted only when all of the following are demonstrated:

- A Connect-produced CyberOps event is consumed from `signalops.<environment>.raw.v1` and schema-validated.
- It is durably stored once logically using `(tenant_id, connect_ingress_event_id)`.
- A duplicate delivery is a no-op; a conflicting duplicate is visible as an integrity failure.
- Authorized tenant-scoped APIs retrieve and filter the event using the required raw attributes.
- Raw payload and complete `metadata.connect` lineage are retained and visible to authorized users.
- Invalid messages follow the established DLQ/audit path without corrupting valid-event processing.
- No SignalOps component opens a Syslog socket, accesses Connect data stores, or requires Connect secrets.
- Automated tests pass, and the implementation documentation includes commands/results for the local acceptance run.

## Implementation Handoff

The code agent should first identify the existing RawSignalEvent consumer and storage path, then extend it rather than creating a parallel broker abstraction. It must preserve unrelated in-progress MarketOps work, use a dedicated branch or worktree, add migrations only where durable query requirements require them, and commit only CyberOps-scoped changes.


## Normative Addendum: Exact Connect CyberOps Contract

This addendum supersedes conflicting earlier language.

### Required envelope

Every CyberOps record MUST be a RawSignalEvent v1 with these values: `app_id=cyberops`, `domain=security`, `use_case=cyberops`, `source_domain=security`, `source_id=metadata.connect.producer_id`, `source_adapter=connect:<connector_id>`, `ingestion_mode=push_event`, `dataset=cyberops.syslog.raw`, `event_type=cyberops.syslog.raw`, `schema_id=signalops.raw_signal_event.v1`, and `schema_version=1.0.0`.

`event_id`, `correlation_id`, and `causation_id` MUST equal `metadata.connect.ingress_event_id`. `idempotency_key` MUST equal `syslog-sha256:<metadata.connect.payload_hash>`. `trace_id` MUST equal `trc_<metadata.connect.ingress_event_id>` unless Connect supplies an immutable trace value. `entity_hints` is an empty array in v1.

`occurred_at`, `effective_time`, and `payload.occurred_at` MUST be exactly equal UTC RFC3339Nano timestamps from RFC5424. `observation_time` and `observed_at` MUST equal Connect ingress receipt time. `processing_time` is Connect completed-processing time. A mismatch is a semantic-validation failure.

### metadata.connect.v1

`metadata.connect` MUST contain `contract_version=1.0.0`, `tenant_id`, `ingress_event_id`, `connector_id`, `connector_version`, `channel_id`, `producer_id`, `protocol_key`, `protocol_version`, `mapping_key`, `mapping_version`, `dataset_binding_id`, `dataset_key`, `dataset_version`, `destination`, `payload_hash_algorithm`, `payload_hash`, `processing_run_id`, and `delivery_id`.

All fields are required non-empty strings. `tenant_id` equals root `tenant_id`; `ingress_event_id` equals root event, correlation, and causation IDs; protocol is exactly `syslog-rfc5424@1.0.0`; `payload_hash_algorithm=sha256`; and `payload_hash` is a lowercase 64-character hexadecimal SHA-256 digest of the exact original RFC5424 datagram bytes, without reserialization or normalization.

Connect MUST add `contract_version`, `tenant_id`, and `payload_hash_algorithm` before acceptance. SignalOps MUST add an explicit semantic validator; shared JSON-Schema validation alone cannot enforce this contract. Immutable lineage comparison canonically serializes all fields above except `processing_run_id` and `delivery_id`, with lexicographically sorted keys, UTF-8, and no insignificant whitespace. It then compares that result, raw event JSON, and payload hash independently.

### Chosen ingestion topology

SignalOps SHALL add dedicated consumer group `signalops.<environment>.connect-raw-persistence.v1` on `signalops.<environment>.raw.v1`. It owns full JSON-Schema validation, Connect/CyberOps semantic validation, raw projection persistence, and integrity-failure persistence. The existing normalizer remains a separate group and does not own Connect raw persistence.

The consumer examines every shared-topic record. A record without `source_adapter` beginning `connect:` or without `app_id=cyberops` is ignored, metric-counted with bounded labels, and offset-committed without a CyberOps write. A record claiming Connect/CyberOps identity that fails schema or semantic validation is non-retryable and follows the DLQ/audit path.

### Durable integrity failure

Create a migration-owned tenant-scoped integrity-failure record containing generated failure ID, tenant ID, Connect ingress ID, existing event identity, existing and incoming payload hashes, existing and incoming canonical immutable lineage, received event JSON, first/last seen timestamps, occurrence count, resolution status (`open`, `acknowledged`, `resolved`), resolution actor/time/note.

On conflicting duplicate, transactionally insert or update this record, retain incoming evidence under security retention, and commit the broker offset. It is an acknowledged non-retryable failure and MUST NOT overwrite the accepted raw event. Failure to persist this record is retryable and leaves the offset uncommitted.

### Mandatory validation and acceptance

Before persistence the dedicated consumer MUST run both full JSON-Schema validation against `raw_signal_event.v1.schema.json` and the semantic validator above. Existing selective normalizer parsing satisfies neither requirement.

Acceptance additionally requires tests for exact-envelope equality rules, metadata semantic validation, ignored non-CyberOps records, conflicting duplicate integrity record creation plus offset commit, and database-write failure without offset commit.


## Normative Addendum 2: Validated Topic and Routing Safety

This addendum supersedes conflicting topology and idempotency wording above.

### Accepted-raw topology

The dedicated `signalops.<environment>.connect-raw-persistence.v1` consumer is the sole consumer of Connect CyberOps records from the shared `signalops.<environment>.raw.v1` topic. After full JSON-Schema validation, semantic validation, and durable idempotent persistence succeed, it MUST publish the exact accepted RawSignalEvent value to `signalops.<environment>.connect-accepted-raw.v1` through the existing durable outbox or equivalent transactionally persisted publication mechanism.

The normalizer MUST consume CyberOps only from `signalops.<environment>.connect-accepted-raw.v1`; it MUST NOT consume or normalize `app_id=cyberops` Connect candidates from the shared raw topic. This topic boundary is mandatory because independent consumer groups cannot use another group validation outcome as a safety barrier. The accepted topic contains only validated Connect CyberOps records. Publication failure after persistence is retryable through the durable outbox; no broker offset is committed until the input persistence/outbox transaction commits.

The existing normalizer may continue consuming non-CyberOps records from the shared raw topic. It must route CyberOps candidates away from that shared path without normalizing them. A future generic accepted-raw architecture may replace this split only if it preserves the same validation barrier.

### Idempotency correction

For CyberOps, root `idempotency_key` MUST equal `metadata.connect.ingress_event_id`, not the payload hash. It therefore equals root `event_id`, `correlation_id`, and `causation_id`. This is compatible with existing raw-ledger uniqueness on `(tenant_id, source_id, idempotency_key)` and permits distinct ingress events containing identical Syslog bytes. `payload_hash` is retained only for source-byte integrity comparison and conflicting-duplicate evidence.

### Conservative candidate classification

Before full schema validation, the persistence consumer classifies a record as a Connect/CyberOps candidate if any one of the following is true: broker header `connect_ingress_event_id` is present; broker header `connect_delivery_id` is present; a bounded JSON pre-parse finds `app_id=cyberops`; a bounded JSON pre-parse finds `source_adapter` beginning `connect:`; or a bounded JSON pre-parse finds a `metadata.connect` object.

A candidate that cannot be decoded, fails RawSignalEvent schema validation, or fails semantic validation is non-retryable and MUST be written to the Connect/CyberOps invalid-message audit/DLQ path. It MUST NOT be silently ignored and it MUST NOT reach the accepted-raw topic or normalizer. Only a non-candidate record is ignored by this consumer and committed without a CyberOps write.

### Additional acceptance evidence

Acceptance must prove that an invalid or conflicting CyberOps candidate never appears on `connect-accepted-raw`, never reaches normalizer CyberOps processing, and has durable invalid-message or integrity evidence. It must also prove that two distinct ingress IDs with identical RFC5424 bytes persist as distinct raw events without a ledger uniqueness conflict.

## Allowed-traffic exposure detection

The initial CyberOps detector consumes normalized `cyberops.syslog.raw` OPNsense
filterlog records. It accepts explicit `pass` or `allow` actions, ignores
`block` and `deny`, and considers only public-source traffic. The first observed
`(tenant_id, destination IP, protocol, destination port)` is emitted as a
low-severity service-exposure signal and becomes a reviewable Insight; it does
not open an Alert. The detector persists this first-seen baseline in a compacted
allowed-service state topic.
