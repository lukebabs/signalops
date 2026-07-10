import unittest

from signalops_plugins.detectors.base import FeatureContext, RuntimeContext
from signalops_plugins.detectors.marketops import MarketOpsDSMTaxonomyDetector, MarketOpsEODPriceDetector


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


def marketops_option_event(**payload_overrides):
    payload = {
        "option_ticker": "O:AAPL270116C00200000",
        "underlying_symbol": "AAPL",
        "contract_type": "call",
        "strike_price": 200.0,
        "expiration_date": "2027-01-16",
        "observation_date": "2026-12-20",
        "open": 5.0,
        "high": 6.0,
        "low": 4.5,
        "close": 5.5,
        "volume": 1500,
        "open_interest": 2000,
        "vwap": 5.25,
        "features": {
            "volume": 1500,
            "open_interest": 2000,
            "volume_open_interest_ratio": 0.75,
            "days_to_expiration": 27,
            "moneyness_pct": 5.0,
        },
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
        "dataset": "options_contracts_daily",
        "event_id": "evt-aapl-option-20261220",
        "normalized_payload": payload,
        "entities": [
            {"type": "option_contract", "id": "option_contract:O:AAPL270116C00200000", "external_id": "O:AAPL270116C00200000"},
            {"type": "ticker", "id": "ticker:AAPL", "external_id": "AAPL"},
        ],
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
        artifact_ids = signal.payload["artifact_ids"]
        self.assertEqual(len(artifact_ids), 1)
        self.assertTrue(artifact_ids[0].startswith("artifact_marketops_dsm_v1_"))
        graph_targets = signal.payload["graph_targets"]
        self.assertEqual(graph_targets[0]["type"], "node_candidate")
        self.assertTrue(
            any(
                target.get("type") == "relationship_candidate"
                and target.get("relationship") == "EXHIBITS_SIGNAL"
                and target.get("to") == f"signal_type:{signal.signal_type}"
                for target in graph_targets
            )
        )
        artifact = signal.payload["semantic_evidence"][0]["artifact"]
        self.assertEqual(artifact["artifact_id"], artifact_ids[0])
        self.assertEqual(artifact["artifact_type"], "marketops.dsm.signal_artifact.v1")
        self.assertEqual(artifact["subject"]["symbol"], "AAPL")
        self.assertEqual(signal.payload["recommendation"]["artifact_ids"], artifact_ids)

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
        semantic = signal.payload["semantic_evidence"][0]
        self.assertIn("non_positive_close", semantic["quality_issues"])
        self.assertIn("non_positive_close", semantic["artifact"]["quality_issues"])
        self.assertEqual(signal.payload["artifact_ids"][0], semantic["artifact_id"])

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
        self.assertEqual(first.payload["artifact_ids"], second.payload["artifact_ids"])


class MarketOpsDSMTaxonomyDetectorTests(unittest.TestCase):
    def detector(self) -> MarketOpsDSMTaxonomyDetector:
        detector = MarketOpsDSMTaxonomyDetector()
        detector.initialize({}, None, RuntimeContext(environment="local", worker_id="test"))
        return detector

    def signal_for(self, event):
        detector = self.detector()
        result = detector.detect([event], FeatureContext())
        signal = detector.emit_signal(result, detector.explain(result))
        self.assertTrue(result.matched)
        self.assertIsNotNone(signal)
        assert signal is not None
        return signal

    def test_emits_accumulation(self) -> None:
        signal = self.signal_for(
            marketops_event(open=100.0, high=104.0, low=99.0, close=103.1, previous_close=100.0, vwap=101.0, volume=2500000)
        )

        self.assertEqual(signal.signal_type, "marketops.dsm.accumulation")
        self.assertEqual(signal.severity, "medium")
        self.assertTrue(signal.signal_id.startswith("sig_marketops_dsm_taxonomy_v1_"))

    def test_emits_divergence(self) -> None:
        signal = self.signal_for(
            marketops_event(open=106.0, high=107.0, low=95.0, close=102.0, previous_close=98.0, vwap=104.0, volume=1500000)
        )

        self.assertEqual(signal.signal_type, "marketops.dsm.divergence")

    def test_emits_hedging_pressure(self) -> None:
        signal = self.signal_for(
            marketops_option_event(features={"volume": 1200, "open_interest": 4000, "volume_open_interest_ratio": 0.3, "days_to_expiration": 60, "moneyness_pct": 4.0})
        )

        self.assertEqual(signal.signal_type, "marketops.dsm.hedging_pressure")
        self.assertEqual(signal.severity, "high")

    def test_emits_speculative_call_pressure(self) -> None:
        signal = self.signal_for(marketops_option_event())

        self.assertEqual(signal.signal_type, "marketops.dsm.speculative_call_pressure")
        self.assertEqual(signal.severity, "medium")

    def test_emits_speculative_put_pressure(self) -> None:
        signal = self.signal_for(
            marketops_option_event(
                option_ticker="O:AAPL270116P00180000",
                contract_type="put",
                features={"volume": 1800, "open_interest": 3000, "volume_open_interest_ratio": 0.6, "days_to_expiration": 27, "moneyness_pct": 5.0},
            )
        )

        self.assertEqual(signal.signal_type, "marketops.dsm.speculative_put_pressure")

    def test_emits_pinning_risk(self) -> None:
        signal = self.signal_for(
            marketops_option_event(
                expiration_date="2026-12-24",
                features={"volume": 800, "open_interest": 2000, "volume_open_interest_ratio": 0.4, "days_to_expiration": 4, "moneyness_pct": 0.5},
            )
        )

        self.assertEqual(signal.signal_type, "marketops.dsm.pinning_risk")
        self.assertEqual(signal.severity, "high")
        self.assertEqual(signal.payload["supporting_metrics"]["open_interest"], 2000)
        self.assertEqual(signal.payload["supporting_metrics"]["volume_open_interest_ratio"], 0.4)
        self.assertEqual(signal.payload["supporting_metrics"]["days_to_expiration"], 4)

    def test_no_match_for_low_intensity_option(self) -> None:
        detector = self.detector()
        event = marketops_option_event(features={"volume": 100, "open_interest": 500, "volume_open_interest_ratio": 0.05, "days_to_expiration": 90, "moneyness_pct": 8.0})

        result = detector.detect([event], FeatureContext())

        self.assertFalse(result.matched)


if __name__ == "__main__":
    unittest.main()
