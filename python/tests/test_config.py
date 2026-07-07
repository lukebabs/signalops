import os
import unittest
from unittest import mock

from signalops_workers.config import dlq_topic, load_config, raw_topic


class ConfigTests(unittest.TestCase):
    def test_raw_topic_defaults_environment(self) -> None:
        self.assertEqual(raw_topic(""), "signalops.local.raw.v1")

    def test_dlq_topic_defaults_environment(self) -> None:
        self.assertEqual(dlq_topic(""), "signalops.local.dlq.algorithm.v1")

    def test_load_config_defaults(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            config = load_config()

        self.assertEqual(config.brokers, "redpanda:9092")
        self.assertEqual(config.environment, "local")
        self.assertEqual(config.input_topic, "signalops.local.raw.v1")
        self.assertEqual(config.dlq_topic, "signalops.local.dlq.algorithm.v1")
        self.assertEqual(config.group_id, "signalops.raw-worker.v1")
        self.assertEqual(config.max_messages, 0)

    def test_load_config_overrides(self) -> None:
        with mock.patch.dict(
            os.environ,
            {
                "SIGNALOPS_BROKER_BROKERS": "localhost:19092",
                "SIGNALOPS_ENV": "test",
                "SIGNALOPS_WORKER_INPUT_TOPIC": "custom.topic",
                "SIGNALOPS_WORKER_DLQ_TOPIC": "custom.dlq",
                "SIGNALOPS_WORKER_GROUP_ID": "custom-group",
                "SIGNALOPS_WORKER_POLL_TIMEOUT_SECONDS": "2.5",
                "SIGNALOPS_WORKER_MAX_MESSAGES": "3",
                "SIGNALOPS_WORKER_LOG_LEVEL": "DEBUG",
            },
            clear=True,
        ):
            config = load_config()

        self.assertEqual(config.brokers, "localhost:19092")
        self.assertEqual(config.environment, "test")
        self.assertEqual(config.input_topic, "custom.topic")
        self.assertEqual(config.dlq_topic, "custom.dlq")
        self.assertEqual(config.group_id, "custom-group")
        self.assertEqual(config.poll_timeout_seconds, 2.5)
        self.assertEqual(config.max_messages, 3)
        self.assertEqual(config.log_level, "DEBUG")


if __name__ == "__main__":
    unittest.main()
