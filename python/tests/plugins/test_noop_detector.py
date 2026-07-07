import unittest

from signalops_plugins.detectors.base import FeatureContext, RuntimeContext
from signalops_plugins.detectors.noop import NoopDetector


class NoopDetectorTests(unittest.TestCase):
    def test_noop_detector_contract(self) -> None:
        detector = NoopDetector()
        detector.initialize({}, None, RuntimeContext(environment="test", worker_id="worker-1"))

        result = detector.detect([{"event_id": "evt-1"}], FeatureContext())
        explanation = detector.explain(result)
        signal = detector.emit_signal(result, explanation)

        self.assertTrue(detector.initialized)
        self.assertEqual(result.detector_id, "signalops.noop")
        self.assertFalse(result.matched)
        self.assertEqual(result.score, 0.0)
        self.assertEqual(result.metadata["event_count"], 1)
        self.assertIsNone(signal)


if __name__ == "__main__":
    unittest.main()
