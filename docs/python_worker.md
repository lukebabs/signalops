# SignalOps Python Worker

The Python worker is the first runnable Python component in SignalOps. It proves
the durable Go-to-Python boundary while keeping algorithm behavior inside
Python detector plugins.

## Runtime

Package:

```text
python/signalops_workers
```

Docker image:

```text
deploy/docker/python-worker/Dockerfile
```

Compose services:

```text
raw-worker
retry-replayer
```

`retry-replayer` is attached to the optional `retry-replay` compose profile so
it does not replay retry records unless explicitly started.

## Behavior

The worker:

- consumes `signalops.<environment>.normalized.v1`;
- decodes Kafka/Redpanda message headers into worker metadata;
- parses the broker value as a JSON object;
- resolves `event_id`, `idempotency_key`, and `correlation_id`;
- invokes the configured detector plugin for valid normalized events;
- logs processed valid normalized events and detector outcomes;
- publishes emitted detector signals to `signalops.<environment>.signal.v1`;
- publishes retryable processing failures to the configured retry topic;
- publishes invalid normalized events and non-retryable processing failures to the
  configured DLQ topic;
- commits source offsets only after the event is processed, signal publish is
  acknowledged when a detector emits a signal, the retry publish is acknowledged,
  or the DLQ publish is acknowledged;
- manually commits explicit topic/partition offsets after each handled message.

Detector execution and signal result emission are implemented. Signal publish
failures are treated as retryable infrastructure failures and are routed through
the retry topic before the source offset is committed.

Published signals are consumed by the Go `signal-persister`, validated again at the infrastructure
boundary, persisted with broker coordinates and normalized-event lineage, and exposed through the
signal query API.

## Configuration

- `SIGNALOPS_BROKER_BROKERS`: broker bootstrap servers.
- `SIGNALOPS_ENV`: environment segment used in default topic naming.
- `SIGNALOPS_WORKER_INPUT_TOPIC`: topic override.
- `SIGNALOPS_WORKER_RETRY_TOPIC`: retry topic for retryable processing failures.
  The default is `signalops.<environment>.retry.algorithm.v1`.
- `SIGNALOPS_WORKER_DLQ_TOPIC`: DLQ topic for invalid normalized events and processing failures.
  The default is `signalops.<environment>.dlq.algorithm.v1`.
- `SIGNALOPS_WORKER_SIGNAL_TOPIC`: signal output topic for detector-emitted
  results. The default is `signalops.<environment>.signal.v1`.
- `SIGNALOPS_WORKER_GROUP_ID`: consumer group ID.
- `SIGNALOPS_WORKER_POLL_TIMEOUT_SECONDS`: broker poll timeout.
- `SIGNALOPS_WORKER_MAX_MESSAGES`: optional finite-run count for validation.
- `SIGNALOPS_WORKER_DETECTOR_ID`: detector plugin identifier. The default is
  `signalops.noop`.
- `SIGNALOPS_WORKER_LOG_LEVEL`: Python logging level.

Retry replayer configuration:

- `SIGNALOPS_RETRY_REPLAY_RAW_TOPIC`: target raw topic for replayed source
  messages. The default is `signalops.<environment>.raw.v1`.
- `SIGNALOPS_RETRY_REPLAY_INPUT_TOPIC`: retry topic to consume. The default is
  `signalops.<environment>.retry.algorithm.v1`.
- `SIGNALOPS_RETRY_REPLAY_DLQ_TOPIC`: DLQ topic for exhausted or invalid retry
  records. The default is `signalops.<environment>.dlq.algorithm.v1`.
- `SIGNALOPS_RETRY_REPLAY_GROUP_ID`: retry replayer consumer group ID.
- `SIGNALOPS_RETRY_REPLAY_POLL_TIMEOUT_SECONDS`: broker poll timeout.
- `SIGNALOPS_RETRY_REPLAY_MAX_MESSAGES`: optional finite-run count for
  validation.
- `SIGNALOPS_RETRY_REPLAY_MAX_ATTEMPTS`: attempt limit before routing the
  original source record to DLQ. The default is `3`.
- `SIGNALOPS_RETRY_REPLAY_LOG_LEVEL`: Python logging level.

## Local Validation

Run unit tests:

```bash
make docker-test-python
```

Build the worker image:

```bash
docker compose build raw-worker
```

Run a finite validation worker:

```bash
docker compose run --rm \
  -e SIGNALOPS_WORKER_GROUP_ID=signalops.g010.validation \
  -e SIGNALOPS_WORKER_MAX_MESSAGES=1 \
  raw-worker
```

Start the long-running worker:

```bash
docker compose up -d raw-worker
```

Run one retry replay validation message:

```bash
docker compose --profile retry-replay run --rm \
  -e SIGNALOPS_RETRY_REPLAY_MAX_MESSAGES=1 \
  retry-replayer
```


## DLQ Contract

DLQ records use `contracts/events/dlq_event.v1.schema.json`. The payload keeps
the source topic, partition, offset, key, headers, and base64-encoded source
value. The worker commits the source offset only after the DLQ publish is
acknowledged.

## Retry Contract

Retry records use `contracts/events/retry_event.v1.schema.json`. The payload
keeps the retry attempt, source topic, partition, offset, key, headers, and
base64-encoded source value. The worker commits the source offset only after
the retry publish is acknowledged.

The retry replayer consumes retry records, reconstructs the original source
message, adds retry replay headers, and republishes it to the configured raw
topic while attempts remain. When `retry_attempt` is greater than or equal to
`SIGNALOPS_RETRY_REPLAY_MAX_ATTEMPTS`, the replayer publishes the original
source message to DLQ with `RetryAttemptsExhausted`. Invalid retry records are
published to DLQ as the retry record itself. The retry topic offset is committed
only after replay or DLQ publication is acknowledged.

## Detector Plugins

Detector contracts live under `python/signalops_plugins/detectors/base.py`.
The default `signalops.noop` detector is deterministic, emits no signals, and
proves the worker/plugin lifecycle before real detector logic is added.

`signalops.static_test` is a deterministic reference detector that emits a low
severity test signal. It is intended for contract and deployment validation, not
production scoring.

## Signal Contract

Emitted detector signals use `contracts/events/signal.v1.schema.json`. The
worker enriches the detector output with source lineage, source-domain metadata,
timestamps, detector and model versions, evidence, and correlation fields before
publishing to the configured signal topic.

## Signal Schema Validation

Before publishing a detector-emitted signal, the worker validates the built
payload against `contracts/events/signal.v1.schema.json` using the checked-in
schema files packaged into the Python worker image. Invalid signal events are
routed to DLQ as `InvalidSignalEventError` and are not published to the signal
topic.
