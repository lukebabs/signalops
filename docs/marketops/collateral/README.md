# SignalOps MarketOps Collateral

This folder contains externally safe, evidence-led MarketOps positioning. It describes the implemented MarketOps daily market-surveillance capability within SignalOps as of August 2026.

## Documents

- [MarketOps Datasheet](marketops-datasheet.md): concise product, workflow, data, and governance overview.
- [Executive Value Brief](marketops-executive-value-brief.md): business problems, differentiated value, and operating outcomes.
- [Analyst Capability Guide](marketops-analyst-capability-guide.md): how an analyst uses the views and evidence layers.
- [Algorithm and Evidence Reference](marketops-algorithm-evidence-reference.md): implementation-level algorithm semantics and limitations.

## Positioning guardrails

MarketOps is deterministic, research-only decision support. It identifies and prioritizes evidence for analyst review; it does not make investment recommendations, submit orders, manage holdings, or guarantee performance. Options-flow measures describe aggregate contract activity and cannot establish trader intent, purchase/sale direction, or hedging purpose.

The operational source of truth is the MarketOps algorithm catalog in `docs/use_cases/marketops/daily_market_surveillance/algorithms/marketops_algorithm_catalog_v1.md`. Retention claims are governed by `docs/support/retention-governance.md`.
