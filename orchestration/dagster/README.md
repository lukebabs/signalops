# SignalOps Dagster definitions

This package is the first shadow control-plane slice for MarketOps. It observes the
current asset-list revision and post-close feature watermark without launching workers
or writing MarketOps data.

The package is intentionally read-only. Provider capture, algorithm execution, and
projection handoff will be added only after the shadow graph has reproduced the prior
partial-cohort incident as `blocked`.
