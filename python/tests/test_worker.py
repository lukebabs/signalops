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

        count = run_worker(
            consumer,
            RawEventHandler(),
            poll_timeout_seconds=0.01,
            max_messages=1,
        )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertTrue(consumer.closed)

    def test_run_worker_commits_invalid_messages(self) -> None:
        message = BrokerMessage(
            topic="signalops.local.raw.v1",
            partition=0,
            offset=1,
            key="bad-1",
            value=b'{"payload":{}}',
            headers={},
        )
        consumer = FakeConsumer([message])

        with self.assertLogs("signalops_workers.worker", level="WARNING"):
            count = run_worker(
                consumer,
                RawEventHandler(),
                poll_timeout_seconds=0.01,
                max_messages=1,
            )

        self.assertEqual(count, 1)
        self.assertEqual(consumer.committed, [message])
        self.assertTrue(consumer.closed)


if __name__ == "__main__":
    unittest.main()

