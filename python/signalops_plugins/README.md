# SignalOps Python Plugins

This package contains the Python-first plugin SDK, reference detectors, and
plugin tests.

The Go core platform communicates with Python workers through broker events or
explicitly versioned internal APIs. Go services must not import, embed, or
directly execute Python libraries.

## Detector Contract

Detector plugins implement:

- `initialize(config, model_registry, runtime_context)`
- `detect(normalized_events, feature_context)`
- `explain(detection_result)`
- `emit_signal(detection_result, explanation)`

The shared detector contract lives in:

```text
python/signalops_plugins/detectors/base.py
```

## Reference Detector

`signalops.noop` is the first reference detector. It is deterministic, emits no
signals, and exists to prove worker/plugin lifecycle wiring before real
algorithm behavior is introduced.

The runnable worker runtime lives in `python/signalops_workers` and loads the
configured detector through `SIGNALOPS_WORKER_DETECTOR_ID`.
