import os
import unittest
from unittest import mock

from signalops_workers.config import (
    dlq_topic,
    load_config,
    load_retry_replay_config,
    normalized_topic,
    raw_topic,
    retry_topic,
    signal_topic,
)


class ConfigTests(unittest.TestCase):
    def test_raw_topic_defaults_environment(self) -> None:
        self.assertEqual(raw_topic(""), "signalops.local.raw.v1")

    def test_normalized_topic_defaults_environment(self) -> None:
        self.assertEqual(normalized_topic(""), "signalops.local.normalized.v1")

    def test_retry_topic_defaults_environment(self) -> None:
        self.assertEqual(retry_topic(""), "signalops.local.retry.algorithm.v1")

    def test_signal_topic_defaults_environment(self) -> None:
        self.assertEqual(signal_topic(""), "signalops.local.signal.v1")

    def test_dlq_topic_defaults_environment(self) -> None:
        self.assertEqual(dlq_topic(""), "signalops.local.dlq.algorithm.v1")

    def test_load_config_defaults(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            config = load_config()

        self.assertEqual(config.brokers, "redpanda:9092")
        self.assertEqual(config.environment, "local")
        self.assertEqual(config.input_topic, "signalops.local.normalized.v1")
        self.assertEqual(config.retry_topic, "signalops.local.retry.algorithm.v1")
        self.assertEqual(config.dlq_topic, "signalops.local.dlq.algorithm.v1")
        self.assertEqual(config.group_id, "signalops.normalized-worker.v1")
        self.assertEqual(config.max_messages, 0)
        self.assertEqual(config.detector_id, "marketops.dsm.taxonomy_v1")

    def test_load_config_overrides(self) -> None:
        with mock.patch.dict(
            os.environ,
            {
                "SIGNALOPS_BROKER_BROKERS": "localhost:19092",
                "SIGNALOPS_ENV": "test",
                "SIGNALOPS_WORKER_INPUT_TOPIC": "custom.topic",
                "SIGNALOPS_WORKER_RETRY_TOPIC": "custom.retry",
                "SIGNALOPS_WORKER_DLQ_TOPIC": "custom.dlq",
                "SIGNALOPS_WORKER_GROUP_ID": "custom-group",
                "SIGNALOPS_WORKER_POLL_TIMEOUT_SECONDS": "2.5",
                "SIGNALOPS_WORKER_MAX_MESSAGES": "3",
                "SIGNALOPS_WORKER_DETECTOR_ID": "signalops.static_test",
                "SIGNALOPS_WORKER_LOG_LEVEL": "DEBUG",
            },
            clear=True,
        ):
            config = load_config()

        self.assertEqual(config.brokers, "localhost:19092")
        self.assertEqual(config.environment, "test")
        self.assertEqual(config.input_topic, "custom.topic")
        self.assertEqual(config.retry_topic, "custom.retry")
        self.assertEqual(config.dlq_topic, "custom.dlq")
        self.assertEqual(config.group_id, "custom-group")
        self.assertEqual(config.poll_timeout_seconds, 2.5)
        self.assertEqual(config.max_messages, 3)
        self.assertEqual(config.detector_id, "signalops.static_test")
        self.assertEqual(config.log_level, "DEBUG")


    def test_load_retry_replay_config_defaults(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            config = load_retry_replay_config()

        self.assertEqual(config.brokers, "redpanda:9092")
        self.assertEqual(config.raw_topic, "signalops.local.raw.v1")
        self.assertEqual(config.retry_topic, "signalops.local.retry.algorithm.v1")
        self.assertEqual(config.dlq_topic, "signalops.local.dlq.algorithm.v1")
        self.assertEqual(config.group_id, "signalops.retry-replayer.v1")
        self.assertEqual(config.max_messages, 0)
        self.assertEqual(config.max_retry_attempts, 3)

    def test_load_retry_replay_config_overrides(self) -> None:
        with mock.patch.dict(
            os.environ,
            {
                "SIGNALOPS_BROKER_BROKERS": "localhost:19092",
                "SIGNALOPS_ENV": "test",
                "SIGNALOPS_RETRY_REPLAY_RAW_TOPIC": "custom.raw",
                "SIGNALOPS_RETRY_REPLAY_INPUT_TOPIC": "custom.retry",
                "SIGNALOPS_RETRY_REPLAY_DLQ_TOPIC": "custom.dlq",
                "SIGNALOPS_RETRY_REPLAY_GROUP_ID": "custom-replayer",
                "SIGNALOPS_RETRY_REPLAY_POLL_TIMEOUT_SECONDS": "2.5",
                "SIGNALOPS_RETRY_REPLAY_MAX_MESSAGES": "5",
                "SIGNALOPS_RETRY_REPLAY_MAX_ATTEMPTS": "7",
                "SIGNALOPS_RETRY_REPLAY_LOG_LEVEL": "DEBUG",
            },
            clear=True,
        ):
            config = load_retry_replay_config()

        self.assertEqual(config.brokers, "localhost:19092")
        self.assertEqual(config.environment, "test")
        self.assertEqual(config.raw_topic, "custom.raw")
        self.assertEqual(config.retry_topic, "custom.retry")
        self.assertEqual(config.dlq_topic, "custom.dlq")
        self.assertEqual(config.group_id, "custom-replayer")
        self.assertEqual(config.poll_timeout_seconds, 2.5)
        self.assertEqual(config.max_messages, 5)
        self.assertEqual(config.max_retry_attempts, 7)
        self.assertEqual(config.log_level, "DEBUG")


if __name__ == "__main__":
    unittest.main()
