# SignalOps Python Worker

The Python worker is the first runnable Python component in SignalOps. It proves
the durable Go-to-Python boundary without introducing detector-specific
algorithm behavior.

## Runtime

Package:

```text
python/signalops_workers
```

Docker image:

```text
deploy/docker/python-worker/Dockerfile
```

Compose service:

```text
raw-worker
```

## Behavior

The worker:

- consumes `signalops.<environment>.raw.v1`;
- decodes Kafka/Redpanda message headers into worker metadata;
- parses the broker value as a JSON object;
- resolves `event_id`, `idempotency_key`, and `correlation_id`;
- logs processed valid raw events;
- publishes retryable processing failures to the configured retry topic;
- publishes invalid raw events and non-retryable processing failures to the
  configured DLQ topic;
- commits source offsets only after the event is processed, the retry publish is
  acknowledged, or the DLQ publish is acknowledged;
- manually commits explicit topic/partition offsets after each handled message.

Detector execution and result emission are not implemented yet. Retry and
DLQ publishing are implemented for worker failure routing.

## Configuration

- `SIGNALOPS_BROKER_BROKERS`: broker bootstrap servers.
- `SIGNALOPS_ENV`: environment segment used in default topic naming.
- `SIGNALOPS_WORKER_INPUT_TOPIC`: topic override.
- `SIGNALOPS_WORKER_RETRY_TOPIC`: retry topic for retryable processing failures.
  The default is `signalops.<environment>.retry.algorithm.v1`.
- `SIGNALOPS_WORKER_DLQ_TOPIC`: DLQ topic for invalid raw events and processing failures.
  The default is `signalops.<environment>.dlq.algorithm.v1`.
- `SIGNALOPS_WORKER_GROUP_ID`: consumer group ID.
- `SIGNALOPS_WORKER_POLL_TIMEOUT_SECONDS`: broker poll timeout.
- `SIGNALOPS_WORKER_MAX_MESSAGES`: optional finite-run count for validation.
- `SIGNALOPS_WORKER_LOG_LEVEL`: Python logging level.

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
