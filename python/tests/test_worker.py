import unittest

from datetime import datetime, timezone

from signalops_plugins.detectors.base import (
    DetectionResult,
    EmittedSignal,
    Explanation,
    FeatureContext,
)
from signalops_workers.worker import (
    BrokerMessage,
    InvalidRawEventError,
    InvalidSignalEventError,
    RawEventHandler,
    RetryableWorkerError,
    build_signal_event,
    run_worker,
)


class FakeConsumer:
    def __init__(self, messages: list[BrokerMessage | None]) -> None:
        self.messages = list(messages)
        self.committed: list[BrokerMessage] = []
        self.closed = False

    def poll(self, timeout_seconds: float) -> BrokerMessage | None:
        if not self.messages:
            return None
        return self.messages.pop(0)

    def commit(self, message: BrokerMessage) -> None:
        self.committed.append(message)

    def close(self) -> None:
        self.closed = True


class FakeFailurePublisher:
    def __init__(self, err: Exception | None = None) -> None:
        self.err = err
        self.published: list[tuple[BrokerMessage, Exception]] = []
        self.closed = False

    def publish(self, message: BrokerMessage, error: Exception) -> None:
        if self.err is not None:
            raise self.err
        self.published.append((message, error))

    def close(self) -> None:
        self.closed = True


class FakeSignalPublisher:
    def __init__(self, err: Exception | None = None) -> None:
        self.err = err
        self.published: list[tuple[dict[str, object], BrokerMessage]] = []
        self.closed = False

    def publish(self, signal_event, source_message: BrokerMessage) -> None:
        if self.err is not None:
            raise self.err
        self.published.append((dict(signal_event), source_message))

    def close(self) -> None:
        self.closed = True


class RetryableHandler:
    def handle(self, message: BrokerMessage):
        raise RetryableWorkerError("transient dependency unavailable")


class FakeDetector:
    detector_id = "fake.detector"
    detector_version = "1.0.0"
    model_version = "none"

    def __init__(self) -> None:
        self.detected_events = []

    def initialize(self, config, model_registry, runtime_context) -> None:
        return None

    def detect(self, normalized_events, feature_context: FeatureContext) -> DetectionResult:
        self.detected_events.append(normalized_events)
        return DetectionResult(
            detector_id=self.detector_id,
            detector_version=self.detector_version,
            matched=False,
            score=0.0,
            reason="test detector",
        )

    def explain(self, detection_result: DetectionResult) -> Explanation:
        return Explanation(summary=detection_result.reason)

    def emit_signal(self, detection_result: DetectionResult, explanation: Explanation):
        return None


class SignalDetector(FakeDetector):
    detector_id = "fake.signal"
    detector_version = "1.0.0"
    model_version = "model-1"

    def detect(self, normalized_events, feature_context: FeatureContext) -> DetectionResult:
        self.detected_events.append(normalized_events)
        return DetectionResult(
            detector_id=self.detector_id,
            detector_version=self.detector_version,
            matched=True,
            score=0.82,
            reason="test signal",
        )

    def emit_signal(
        self, detection_result: DetectionResult, explanation: Explanation
    ) -> EmittedSignal | None:
        return EmittedSignal(
            signal_id="sig-1",
            signal_type="test_signal",
            confidence=detection_result.score,
            severity="medium",
            payload={
                "supporting_metrics": {"score": detection_result.score},
                "entities": [{"type": "ticker", "id": "SPY"}],
            },
        )


class InvalidSignalDetector(SignalDetector):
    def emit_signal(
        self, detection_result: DetectionResult, explanation: Explanation
    ) -> EmittedSignal | None:
        return EmittedSignal(
            signal_id="sig-invalid",
            signal_type="test_signal",
            confidence=1.5,
            severity="medium",
            payload={},
        )


class RetryableDetector(FakeDetector):
    def detect(self, normalized_events, feature_context: FeatureContext) -> DetectionResult:
        raise RetryableWorkerError("detector dependency unavailable")


def raw_signal_message() -> BrokerMessage:
    return BrokerMessage(
        topic="signalops.local.raw.v1",
        partition=0,
        offset=1,
        key="idem-signal",
        value=(
            b'{"tenant_id":"tenant-1","source_id":"source-1",'
            b'"source_domain":"market_data","source_adapter":"market_data.massive",'
            b'"ingestion_mode":"scheduled_pull","dataset":"daily_options",'
            b'"event_id":"evt-signal","observation_time":"2026-07-07T00:00:00Z",'
            b'"effective_time":"2026-07-07T00:00:00Z",'
            b'"processing_time":"2026-07-07T00:01:00Z",'
            b'"correlation_id":"corr-signal","idempotency_key":"idem-signal"}'
        ),
        headers={"correlation_id": "corr-signal"},
    )


class WorkerTests(unittest.TestCase):

    def test_build_signal_event_maps_detector_output_to_contract(self) -> None:
        processed = RawEventHandler().handle(raw_signal_message())
        detector = SignalDetector()
        detection = detector.detect([processed.payload], FeatureContext())
        explanation = detector.explain(detection)
        signal = detector.emit_signal(detection, explanation)
        assert signal is not None

        event = build_signal_event(
            processed,
            detector,
            signal,
            explanation,
            now=datetime(2026, 7, 7, 2, 0, 0, tzinfo=timezone.utc),
        )

        self.assertEqual(event["signal_id"], "sig-1")
        self.assertEqual(event["tenant_id"], "tenant-1")
        self.assertEqual(event["source_domain"], "market_data")
        self.assertEqual(event["event_ids"], ["evt-signal"])
        self.assertEqual(event["detector_id"], "fake.signal")
        self.assertEqual(event["model_version"], "model-1")
        self.assertEqual(event["timestamp"], "2026-07-07T02:00:00Z")
        self.assertEqual(event["confidence"], 0.82)
        self.assertEqual(event["severity"], "medium")
        self.assertEqual(event["correlation_id"], "corr-signal")
        self.assertEqual(event["evidence"][0]["type"], "normalized_event")

    def test_run_worker_publishes_emitted_signals(self) -> None:
        message = raw_signal_message()
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher()
        dlq = FakeFailurePublisher()
        signals = FakeSignalPublisher()

        count = run_worker(
            consumer,
            RawEventHandler(),
            poll_timeout_seconds=0.01,
            max_messages=1,
            retry_publisher=retry,
            dead_letter_publisher=dlq,
            signal_publisher=signals,
            detector=SignalDetector(),
        )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(retry.published, [])
        self.assertEqual(dlq.published, [])
        self.assertEqual(len(signals.published), 1)
        self.assertEqual(signals.published[0][0]["signal_id"], "sig-1")
        self.assertTrue(signals.closed)

    def test_run_worker_routes_invalid_signal_events_to_dlq(self) -> None:
        message = raw_signal_message()
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher()
        dlq = FakeFailurePublisher()
        signals = FakeSignalPublisher()

        with self.assertLogs("signalops_workers.worker", level="WARNING"):
            count = run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                retry_publisher=retry,
                dead_letter_publisher=dlq,
                signal_publisher=signals,
                detector=InvalidSignalDetector(),
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(retry.published, [])
        self.assertEqual(signals.published, [])
        self.assertEqual(len(dlq.published), 1)
        self.assertIsInstance(dlq.published[0][1], InvalidSignalEventError)

    def test_run_worker_routes_signal_publish_failures_to_retry(self) -> None:
        message = raw_signal_message()
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher()
        dlq = FakeFailurePublisher()
        signals = FakeSignalPublisher(err=RuntimeError("signal topic unavailable"))

        with self.assertLogs("signalops_workers.worker", level="WARNING"):
            count = run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                retry_publisher=retry,
                dead_letter_publisher=dlq,
                signal_publisher=signals,
                detector=SignalDetector(),
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(len(retry.published), 1)
        self.assertIsInstance(retry.published[0][1], RetryableWorkerError)
        self.assertEqual(dlq.published, [])
        self.assertEqual(signals.published, [])

    def test_run_worker_does_not_commit_when_signal_retry_publish_fails(self) -> None:
        message = raw_signal_message()
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher(err=RuntimeError("retry unavailable"))
        dlq = FakeFailurePublisher()
        signals = FakeSignalPublisher(err=RuntimeError("signal topic unavailable"))

        with self.assertRaises(RuntimeError):
            run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                retry_publisher=retry,
                dead_letter_publisher=dlq,
                signal_publisher=signals,
                detector=SignalDetector(),
            )

        self.assertEqual(consumer.committed, [])
        self.assertEqual(dlq.published, [])
        self.assertEqual(signals.published, [])
        self.assertTrue(signals.closed)

    def test_handler_parses_raw_event(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="idem-1",
            value=b'{"event_id":"evt-1","payload":{"symbol":"SPY"}}',
            headers={"correlation_id": "corr-1"},
        )

        processed = RawEventHandler().handle(message)

        self.assertEqual(processed.event_id, "evt-1")
        self.assertEqual(processed.idempotency_key, "idem-1")
        self.assertEqual(processed.correlation_id, "corr-1")
        self.assertEqual(processed.payload["payload"], {"symbol": "SPY"})

    def test_handler_rejects_non_object_json(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="idem-1",
            value=b"[]",
            headers={},
        )

        with self.assertRaises(InvalidRawEventError):
            RawEventHandler().handle(message)

    def test_run_worker_commits_processed_messages(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="idem-1",
            value=b'{"event_id":"evt-1"}',
            headers={},
        )
        consumer = FakeConsumer([message])
        dlq = FakeFailurePublisher()

        retry = FakeFailurePublisher()

        detector = FakeDetector()

        count = run_worker(
            consumer,
            RawEventHandler(),
            poll_timeout_seconds=0.01,
            max_messages=1,
            retry_publisher=retry,
            dead_letter_publisher=dlq,
            detector=detector,
        )

        self.assertEqual(count, 1)
        self.assertEqual(detector.detected_events, [[{"event_id": "evt-1"}]])
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(retry.published, [])
        self.assertEqual(dlq.published, [])
        self.assertTrue(consumer.closed)
        self.assertTrue(retry.closed)
        self.assertTrue(dlq.closed)

    def test_run_worker_routes_retryable_detector_failures_to_retry(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="retry-2",
            value=b'{"event_id":"evt-2"}',
            headers={},
        )
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher()
        dlq = FakeFailurePublisher()

        with self.assertLogs("signalops_workers.worker", level="WARNING"):
            count = run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                retry_publisher=retry,
                dead_letter_publisher=dlq,
                detector=RetryableDetector(),
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(len(retry.published), 1)
        self.assertIsInstance(retry.published[0][1], RetryableWorkerError)
        self.assertEqual(dlq.published, [])

    def test_run_worker_publishes_retryable_messages_to_retry_topic(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="retry-1",
            value=b'{"event_id":"evt-1"}',
            headers={},
        )
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher()
        dlq = FakeFailurePublisher()

        with self.assertLogs("signalops_workers.worker", level="WARNING"):
            count = run_worker(
                consumer,
                RetryableHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                retry_publisher=retry,
                dead_letter_publisher=dlq,
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(len(retry.published), 1)
        self.assertIsInstance(retry.published[0][1], RetryableWorkerError)
        self.assertEqual(dlq.published, [])
        self.assertTrue(consumer.closed)
        self.assertTrue(retry.closed)
        self.assertTrue(dlq.closed)

    def test_run_worker_does_not_commit_when_retry_publish_fails(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="retry-1",
            value=b'{"event_id":"evt-1"}',
            headers={},
        )
        consumer = FakeConsumer([message])
        retry = FakeFailurePublisher(err=RuntimeError("retry unavailable"))
        dlq = FakeFailurePublisher()

        with self.assertRaises(RuntimeError):
            run_worker(
                consumer,
                RetryableHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                retry_publisher=retry,
                dead_letter_publisher=dlq,
            )

        self.assertEqual(consumer.committed, [])
        self.assertEqual(dlq.published, [])
        self.assertTrue(consumer.closed)
        self.assertTrue(retry.closed)
        self.assertTrue(dlq.closed)

    def test_run_worker_publishes_invalid_messages_to_dlq(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="bad-1",
            value=b'{"payload":{}}',
            headers={},
        )
        consumer = FakeConsumer([message])
        dlq = FakeFailurePublisher()

        with self.assertLogs("signalops_workers.worker", level="WARNING"):
            count = run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                dead_letter_publisher=dlq,
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(len(dlq.published), 1)
        self.assertIsInstance(dlq.published[0][1], InvalidRawEventError)
        self.assertTrue(consumer.closed)
        self.assertTrue(dlq.closed)

    def test_run_worker_does_not_commit_when_dlq_publish_fails(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="bad-1",
            value=b'{"payload":{}}',
            headers={},
        )
        consumer = FakeConsumer([message])
        dlq = FakeFailurePublisher(err=RuntimeError("dlq unavailable"))

        with self.assertRaises(RuntimeError):
            run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
                dead_letter_publisher=dlq,
            )

        self.assertEqual(consumer.committed, [])
        self.assertTrue(consumer.closed)
        self.assertTrue(dlq.closed)

    def test_run_worker_without_dlq_raises_invalid_message(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="bad-1",
            value=b'{"payload":{}}',
            headers={},
        )
        consumer = FakeConsumer([message])

        with self.assertRaises(InvalidRawEventError):
            run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
            )

        self.assertEqual(consumer.committed, [])
        self.assertTrue(consumer.closed)


if __name__ == "__main__":
    unittest.main()
