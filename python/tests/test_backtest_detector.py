import unittest

from datetime import datetime, timezone

from signalops_workers.backtest_detector import run_batch


class BacktestDetectorTests(unittest.TestCase):
    def test_run_batch_emits_signal_for_marketops_event(self) -> None:
        events = [
            {
                "event_id": "evt-1",
                "tenant_id": "tenant-1",
                "source_id": "src-massive",
                "app_id": "marketops",
                "domain": "market_data",
                "use_case": "daily_market_surveillance",
                "source_domain": "market_data",
                "source_adapter": "market_data.massive",
                "ingestion_mode": "scheduled_pull",
                "dataset": "equity_eod_prices",
                "observation_time": "2026-07-10T00:00:00Z",
                "effective_time": "2026-07-10T00:00:00Z",
                "processing_time": "2026-07-10T00:01:00Z",
                "normalized_payload": {
                    "symbol": "AAPL",
                    "observation_date": "2026-07-10",
                    "open": 100,
                    "high": 109,
                    "low": 99,
                    "close": 108,
                    "previous_close": 100,
                    "vwap": 105,
                    "volume": 2000000,
                },
            }
        ]

        signals = run_batch(
            events,
            detector_id="marketops.dsm.eod_price_v1",
            now=datetime(2026, 7, 10, 1, 0, 0, tzinfo=timezone.utc),
        )

        self.assertEqual(len(signals), 1)
        self.assertEqual(signals[0]["app_id"], "marketops")
        self.assertEqual(signals[0]["signal_type"], "marketops.dsm.volatility_expansion")
        self.assertEqual(signals[0]["event_ids"], ["evt-1"])

    def test_run_batch_suppresses_non_matching_event(self) -> None:
        signals = run_batch(
            [
                {
                    "event_id": "evt-2",
                    "tenant_id": "tenant-1",
                    "source_id": "src-massive",
                    "app_id": "console",
                    "domain": "market_data",
                    "use_case": "general",
                    "source_domain": "market_data",
                    "source_adapter": "market_data.massive",
                    "ingestion_mode": "scheduled_pull",
                    "dataset": "equity_eod_prices",
                    "observation_time": "2026-07-10T00:00:00Z",
                    "effective_time": "2026-07-10T00:00:00Z",
                    "processing_time": "2026-07-10T00:01:00Z",
                    "normalized_payload": {"symbol": "AAPL", "open": 100, "high": 101, "low": 99, "close": 100},
                }
            ],
            detector_id="marketops.dsm.eod_price_v1",
        )

        self.assertEqual(signals, [])


if __name__ == "__main__":
    unittest.main()
