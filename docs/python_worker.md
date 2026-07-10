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
  `marketops.dsm.taxonomy_v1`. Set this to `signalops.noop` or
  `signalops.static_test` for lifecycle-only validation runs.
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
env PYTHONPATH=python pytest python/tests
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

`marketops.dsm.taxonomy_v1` is the default detector. It evaluates normalized
Massive equity EOD and option contract daily events scoped to `app_id=marketops`,
`domain=market_data`, `source_adapter=market_data.massive`, and
`use_case=daily_market_surveillance`. Equity records produce price-derived DSM
features and quality checks; option records use option-interest features such as
open interest, volume/open-interest ratio, days to expiration, and optional
moneyness percent. Volatility expansion thresholds remain 3.0% absolute
open/close move, 5.0% intraday range, or 4.0% absolute daily return.
Price-quality exceptions take precedence over equity price signals for a single
event.

`signalops.noop` is deterministic, emits no signals, and remains available for
lifecycle-only validation runs.

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


## G047 lifecycle persistence

After a Python detector signal is published, the Go `signal-persister` validates and persists the
signal, then derives durable lifecycle records: one active insight for every signal and one open alert
for `medium`, `high`, or `critical` severities. This keeps Python responsible for algorithms while Go
owns infrastructure durability, idempotent persistence, and operator-facing lifecycle APIs.

### MarketOps DSM Artifacts And Graph Proposals

`marketops.dsm.taxonomy_v1` emits deterministic DSM artifact proposal metadata inside the existing
`signal.v1` payload. The worker persists these through the existing signal ledger fields; no
standalone artifact or graph service is required for G074.

- `artifact_ids`: one stable `artifact_marketops_dsm_v1_*` ID per emitted signal.
- `semantic_evidence[0].artifact`: a `marketops.dsm.signal_artifact.v1` proposal with source event,
  ticker subject, severity, confidence, computed feature summary, and quality issues.
- `graph_targets`: node candidates for ticker, DSM signal type, artifact, plus relationship
  candidates `EXHIBITS_SIGNAL` and `SUPPORTED_BY_ARTIFACT`.
- `recommendation`: includes the artifact IDs and graph target count for operator review.

### MarketOps DSM Taxonomy Detector

`marketops.dsm.taxonomy_v1` is the default always-on detector pack. It preserves the prior EOD
price signals and adds deterministic DSM taxonomy signals for normalized Massive market data:

- Equity EOD: `marketops.dsm.accumulation`, `marketops.dsm.divergence`,
  `marketops.dsm.volatility_expansion`, and `marketops.dsm.price_quality_exception`.
- Option contract daily: `marketops.dsm.hedging_pressure`,
  `marketops.dsm.speculative_call_pressure`, `marketops.dsm.speculative_put_pressure`, and
  `marketops.dsm.pinning_risk`.

The legacy `marketops.dsm.eod_price_v1` detector remains loadable through
`SIGNALOPS_WORKER_DETECTOR_ID` for targeted validation or rollback.
