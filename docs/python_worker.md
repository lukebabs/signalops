# SignalOps Python Worker

The G010 worker is the first runnable Python component in SignalOps. It proves
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
- logs and commits invalid raw events so a poison record does not crash-loop
  the skeleton worker;
- manually commits explicit topic/partition offsets after each handled message.

Detector execution, retry topics, DLQ publishing, and result emission are not
implemented in G010. Those are separate gates.

## Configuration

- `SIGNALOPS_BROKER_BROKERS`: broker bootstrap servers.
- `SIGNALOPS_ENV`: environment segment used in default topic naming.
- `SIGNALOPS_WORKER_INPUT_TOPIC`: topic override.
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

