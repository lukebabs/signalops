import unittest

from signalops_workers.detectors import UnknownDetectorError, load_detector


class DetectorLoaderTests(unittest.TestCase):
    def test_loads_noop_detector(self) -> None:
        detector = load_detector("signalops.noop", environment="local", worker_id="worker-1")

        self.assertEqual(detector.detector_id, "signalops.noop")

    def test_rejects_unknown_detector(self) -> None:
        with self.assertRaises(UnknownDetectorError):
            load_detector("missing.detector", environment="local", worker_id="worker-1")


if __name__ == "__main__":
    unittest.main()
