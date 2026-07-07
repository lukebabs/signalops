# SignalOps Python Plugins

This directory will contain the Python-first algorithm plugin SDK, runners,
reference algorithms, and tests.

The Go core platform must communicate with Python workers through broker
events or explicitly versioned internal APIs. Go services must not import,
embed, or directly execute Python libraries.


The runnable worker runtime currently lives in `python/signalops_workers`.
Detector plugin contracts will be layered on top of that runtime in later
gates.
