from __future__ import annotations

import logging

from signalops_workers.broker import (
    RedpandaDeadLetterPublisher,
    RedpandaRawEventConsumer,
    RedpandaRawEventPublisher,
)
from signalops_workers.config import load_retry_replay_config
from signalops_workers.retry_replay import run_retry_replayer


def main() -> int:
    config = load_retry_replay_config()
    logging.basicConfig(
        level=config.log_level,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )

    consumer = RedpandaRawEventConsumer(
        brokers=config.brokers,
        group_id=config.group_id,
        input_topic=config.retry_topic,
    )
    raw_publisher = RedpandaRawEventPublisher(
        brokers=config.brokers,
        raw_topic=config.raw_topic,
    )
    dead_letter_publisher = RedpandaDeadLetterPublisher(
        brokers=config.brokers,
        dlq_topic=config.dlq_topic,
    )
    handled = run_retry_replayer(
        consumer,
        raw_publisher,
        dead_letter_publisher,
        poll_timeout_seconds=config.poll_timeout_seconds,
        max_retry_attempts=config.max_retry_attempts,
        max_messages=config.max_messages,
    )
    logging.getLogger(__name__).info("retry replayer stopped", extra={"handled": handled})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
