import base64
import json
import unittest

from signalops_workers.retry_replay import (
    InvalidRetryEventError,
    RetryAttemptsExhausted,
    retry_replay_decision,
    run_retry_replayer,
)
from signalops_workers.worker import BrokerMessage


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


class FakeRawPublisher:
    def __init__(self, err: Exception | None = None) -> None:
        self.err = err
        self.published: list[BrokerMessage] = []
        self.closed = False

    def publish(self, message: BrokerMessage) -> None:
        if self.err is not None:
            raise self.err
        self.published.append(message)

    def close(self) -> None:
        self.closed = True


class FakeDeadLetterPublisher:
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


def retry_message(*, retry_attempt: int = 1, value: bytes = b'{"event_id":"evt-1"}') -> BrokerMessage:
    payload = {
        "schema_id": "signalops.retry.raw_event.v1",
        "failed_at": "2026-07-07T00:00:00Z",
        "error_type": "RetryableWorkerError",
        "error_message": "transient dependency unavailable",
        "retry_attempt": retry_attempt,
        "source": {
            "topic": "signalops.local.raw.v1",
            "partition": 2,
            "offset": 99,
            "key": "idem-1",
            "headers": {"correlation_id": "corr-1"},
            "value_base64": base64.b64encode(value).decode("ascii"),
        },
    }
    return BrokerMessage(
        topic="signalops.local.retry.algorithm.v1",
        partition=0,
        offset=4,
        key="idem-1",
        value=json.dumps(payload).encode("utf-8"),
        headers={"correlation_id": "corr-1"},
    )


class RetryReplayTests(unittest.TestCase):
    def test_retry_replay_decision_reconstructs_source_message(self) -> None:
        decision = retry_replay_decision(retry_message(retry_attempt=2), 3)

        self.assertEqual(decision.action, "replay")
        self.assertEqual(decision.retry_attempt, 2)
        assert decision.source_message is not None
        self.assertEqual(decision.source_message.topic, "signalops.local.raw.v1")
        self.assertEqual(decision.source_message.partition, 2)
        self.assertEqual(decision.source_message.offset, 99)
        self.assertEqual(decision.source_message.key, "idem-1")
        self.assertEqual(decision.source_message.value, b'{"event_id":"evt-1"}')
        self.assertEqual(decision.source_message.headers["correlation_id"], "corr-1")
        self.assertEqual(decision.source_message.headers["signalops_retry_attempt"], "2")
        self.assertEqual(decision.source_message.headers["signalops_replayed_from_retry"], "true")

    def test_retry_replay_decision_exhausts_at_limit(self) -> None:
        decision = retry_replay_decision(retry_message(retry_attempt=3), 3)

        self.assertEqual(decision.action, "dlq")
        self.assertEqual(decision.retry_attempt, 3)
        self.assertIsNotNone(decision.source_message)

    def test_retry_replay_decision_rejects_bad_schema(self) -> None:
        bad = retry_message()
        payload = json.loads(bad.value.decode("utf-8"))
        payload["schema_id"] = "wrong"
        bad = BrokerMessage(
            topic=bad.topic,
            partition=bad.partition,
            offset=bad.offset,
            key=bad.key,
            value=json.dumps(payload).encode("utf-8"),
            headers=bad.headers,
        )

        with self.assertRaises(InvalidRetryEventError):
            retry_replay_decision(bad, 3)

    def test_run_retry_replayer_republishes_source_and_commits_retry_record(self) -> None:
        message = retry_message(retry_attempt=1)
        consumer = FakeConsumer([message])
        raw = FakeRawPublisher()
        dlq = FakeDeadLetterPublisher()

        count = run_retry_replayer(
            consumer,
            raw,
            dlq,
            poll_timeout_seconds=0.01,
            max_retry_attempts=3,
            max_messages=1,
        )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(len(raw.published), 1)
        self.assertEqual(raw.published[0].headers["signalops_retry_attempt"], "1")
        self.assertEqual(dlq.published, [])
        self.assertTrue(consumer.closed)
        self.assertTrue(raw.closed)
        self.assertTrue(dlq.closed)

    def test_run_retry_replayer_routes_exhausted_retry_to_dlq(self) -> None:
        message = retry_message(retry_attempt=3)
        consumer = FakeConsumer([message])
        raw = FakeRawPublisher()
        dlq = FakeDeadLetterPublisher()

        with self.assertLogs("signalops_workers.retry_replay", level="WARNING"):
            count = run_retry_replayer(
                consumer,
                raw,
                dlq,
                poll_timeout_seconds=0.01,
                max_retry_attempts=3,
                max_messages=1,
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(raw.published, [])
        self.assertEqual(len(dlq.published), 1)
        self.assertIsInstance(dlq.published[0][1], RetryAttemptsExhausted)
        self.assertEqual(dlq.published[0][0].topic, "signalops.local.raw.v1")

    def test_run_retry_replayer_routes_invalid_retry_record_to_dlq(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.retry.algorithm.v1",
            partition=0,
            offset=1,
            key="bad-retry",
            value=b"{}",
            headers={},
        )
        consumer = FakeConsumer([message])
        raw = FakeRawPublisher()
        dlq = FakeDeadLetterPublisher()

        with self.assertLogs("signalops_workers.retry_replay", level="WARNING"):
            count = run_retry_replayer(
                consumer,
                raw,
                dlq,
                poll_timeout_seconds=0.01,
                max_retry_attempts=3,
                max_messages=1,
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(raw.published, [])
        self.assertEqual(len(dlq.published), 1)
        self.assertIsInstance(dlq.published[0][1], InvalidRetryEventError)

    def test_run_retry_replayer_does_not_commit_when_republish_fails(self) -> None:
        message = retry_message(retry_attempt=1)
        consumer = FakeConsumer([message])
        raw = FakeRawPublisher(err=RuntimeError("raw unavailable"))
        dlq = FakeDeadLetterPublisher()

        with self.assertRaises(RuntimeError):
            run_retry_replayer(
                consumer,
                raw,
                dlq,
                poll_timeout_seconds=0.01,
                max_retry_attempts=3,
                max_messages=1,
            )

        self.assertEqual(consumer.committed, [])
        self.assertEqual(dlq.published, [])
        self.assertTrue(consumer.closed)
        self.assertTrue(raw.closed)
        self.assertTrue(dlq.closed)


if __name__ == "__main__":
    unittest.main()
