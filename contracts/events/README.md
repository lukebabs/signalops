# Event Contracts

Event contracts define the broker payloads exchanged between SignalOps
adapters, Go platform workers, and Python processing workers.

Initial events:

- `RawSignalEvent`
- `NormalizedSignalEvent`
- `Signal`
- `EventArtifact`
- `GraphMutationProposal`
- `InsightCandidate`
- `RetryEvent`
- `DLQEvent`

Versioned schemas:

- `common.defs.v1.schema.json`
- `raw_signal_event.v1.schema.json`
- `normalized_signal_event.v1.schema.json`
- `signal.v1.schema.json`
- `retry_event.v1.schema.json`
- `dlq_event.v1.schema.json`

The v1 schemas establish the shared boundary between the Go core platform and
Python processing workers. They include source-domain, adapter, ingestion-mode,
dataset, time, correlation, and idempotency fields required for replayable
multi-domain signal processing.

Raw and normalized event metadata may include the shared `quality` envelope. When
present it requires `quality_state`, `quality_policy_id`, and semantic
`quality_policy_version`. The state is one of the platform standard quality
states. The Massive opt-in registry enforcement path uses this envelope for the
active `signalops.normalized_event_quality` policy.

`DLQEvent` captures failed durable processing attempts with source topic,
partition, offset, headers, and base64-encoded original payload so failures can
be audited and replayed without losing the original broker value.

`RetryEvent` captures retryable durable processing failures with retry attempt,
source topic, partition, offset, headers, and base64-encoded original payload.
