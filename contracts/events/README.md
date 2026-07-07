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
- `DLQEvent`

Versioned schemas:

- `common.defs.v1.schema.json`
- `raw_signal_event.v1.schema.json`
- `normalized_signal_event.v1.schema.json`
- `signal.v1.schema.json`
- `dlq_event.v1.schema.json`

The v1 schemas establish the shared boundary between the Go core platform and
Python processing workers. They include source-domain, adapter, ingestion-mode,
dataset, time, correlation, and idempotency fields required for replayable
multi-domain signal processing.

`DLQEvent` captures failed durable processing attempts with source topic,
partition, offset, headers, and base64-encoded original payload so failures can
be audited and replayed without losing the original broker value.
