import unittest

from signalops_plugins.detectors.base import FeatureContext, RuntimeContext
from signalops_plugins.detectors.marketops import MarketOpsEODPriceDetector


def marketops_event(**payload_overrides):
    payload = {
        "symbol": "AAPL",
        "observation_date": "2026-07-09",
        "open": 100.0,
        "high": 108.0,
        "low": 99.0,
        "close": 106.0,
        "volume": 1200000,
        "vwap": 103.0,
        "previous_close": 101.0,
    }
    payload.update(payload_overrides)
    return {
        "tenant_id": "tenant-local",
        "source_id": "src-massive",
        "app_id": "marketops",
        "domain": "market_data",
        "use_case": "daily_market_surveillance",
        "source_domain": "market_data",
        "source_adapter": "market_data.massive",
        "ingestion_mode": "scheduled_pull",
        "dataset": "equity_eod_prices",
        "event_id": "evt-aapl-20260709",
        "normalized_payload": payload,
        "entities": [{"type": "ticker", "id": "ticker:AAPL", "external_id": "AAPL"}],
    }


class MarketOpsEODPriceDetectorTests(unittest.TestCase):
    def detector(self) -> MarketOpsEODPriceDetector:
        detector = MarketOpsEODPriceDetector()
        detector.initialize({}, None, RuntimeContext(environment="local", worker_id="test"))
        return detector

    def test_no_match_for_non_marketops_event(self) -> None:
        event = marketops_event()
        event["app_id"] = "console"
        detector = self.detector()

        result = detector.detect([event], FeatureContext())

        self.assertFalse(result.matched)
        self.assertIsNone(detector.emit_signal(result, detector.explain(result)))

    def test_no_match_for_non_equity_dataset(self) -> None:
        event = marketops_event()
        event["dataset"] = "option_contracts_daily"
        detector = self.detector()

        result = detector.detect([event], FeatureContext())

        self.assertFalse(result.matched)

    def test_emits_volatility_expansion_signal(self) -> None:
        detector = self.detector()
        result = detector.detect([marketops_event()], FeatureContext())
        explanation = detector.explain(result)
        signal = detector.emit_signal(result, explanation)

        self.assertTrue(result.matched)
        self.assertIsNotNone(signal)
        assert signal is not None
        self.assertEqual(signal.signal_type, "marketops.dsm.volatility_expansion")
        self.assertEqual(signal.severity, "high")
        self.assertEqual(signal.payload["entities"][0]["external_id"], "AAPL")
        metrics = signal.payload["supporting_metrics"]
        self.assertEqual(metrics["open_close_move_pct"], 6.0)
        self.assertEqual(metrics["intraday_range_pct"], 9.0)
        self.assertEqual(metrics["vwap_distance_pct"], 2.9126)
        self.assertEqual(metrics["daily_return_pct"], 4.9505)
        self.assertEqual(signal.payload["evidence"][0]["type"], "normalized_event")
        self.assertEqual(signal.payload["graph_targets"][0]["to"], signal.signal_type)

    def test_emits_price_quality_exception(self) -> None:
        detector = self.detector()
        result = detector.detect([marketops_event(close=0.0)], FeatureContext())
        explanation = detector.explain(result)
        signal = detector.emit_signal(result, explanation)

        self.assertTrue(result.matched)
        self.assertIsNotNone(signal)
        assert signal is not None
        self.assertEqual(signal.signal_type, "marketops.dsm.price_quality_exception")
        self.assertEqual(signal.severity, "medium")
        self.assertIn("non_positive_close", signal.payload["semantic_evidence"][0]["quality_issues"])

    def test_previous_close_is_optional(self) -> None:
        detector = self.detector()
        event = marketops_event(open=100.0, high=106.0, low=100.0, close=103.5)
        del event["normalized_payload"]["previous_close"]

        result = detector.detect([event], FeatureContext())
        signal = detector.emit_signal(result, detector.explain(result))

        self.assertTrue(result.matched)
        self.assertIsNotNone(signal)
        assert signal is not None
        self.assertIsNone(signal.payload["supporting_metrics"]["daily_return_pct"])

    def test_signal_id_is_stable_for_same_input(self) -> None:
        detector = self.detector()
        event = marketops_event()

        first = detector.emit_signal(
            detector.detect([event], FeatureContext()),
            detector.explain(detector.detect([event], FeatureContext())),
        )
        second_result = detector.detect([event], FeatureContext())
        second = detector.emit_signal(second_result, detector.explain(second_result))

        assert first is not None
        assert second is not None
        self.assertEqual(first.signal_id, second.signal_id)


if __name__ == "__main__":
    unittest.main()
