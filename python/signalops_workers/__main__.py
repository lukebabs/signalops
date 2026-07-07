from __future__ import annotations

import logging

from signalops_workers.broker import RedpandaRawEventConsumer
from signalops_workers.config import load_config
from signalops_workers.worker import RawEventHandler, run_worker


def main() -> int:
    config = load_config()
    logging.basicConfig(
        level=config.log_level,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )

    consumer = RedpandaRawEventConsumer(
        brokers=config.brokers,
        group_id=config.group_id,
        input_topic=config.input_topic,
    )
    processed = run_worker(
        consumer,
        RawEventHandler(),
        poll_timeout_seconds=config.poll_timeout_seconds,
        max_messages=config.max_messages,
    )
    logging.getLogger(__name__).info("worker stopped", extra={"processed": processed})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

