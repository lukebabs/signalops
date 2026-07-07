import unittest

from signalops_workers.worker import (
    BrokerMessage,
    InvalidRawEventError,
    RawEventHandler,
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


class WorkerTests(unittest.TestCase):
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
        dlq = FakeDeadLetterPublisher()

        count = run_worker(
            consumer,
            RawEventHandler(),
            poll_timeout_seconds=0.01,
            max_messages=1,
            dead_letter_publisher=dlq,
        )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertEqual(dlq.published, [])
        self.assertTrue(consumer.closed)
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
        dlq = FakeDeadLetterPublisher()

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
        dlq = FakeDeadLetterPublisher(err=RuntimeError("dlq unavailable"))

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
