# MarketOps Use Cases

MarketOps is the first specialized SignalOps app profile. MarketOps documents should be organized by concrete use case, not as loose top-level files.

Current use cases:

- `daily_market_surveillance/`: deterministic market-data surveillance using Massive equity EOD and option contract daily data.

The implemented algorithm catalog, including VC/DOSM, daily tactical posture, Exhaustive Reversal, Risk/Reward, and convergent review semantics, is maintained at `daily_market_surveillance/algorithms/marketops_algorithm_catalog_v1.md`.

The implemented SRI Foundation is documented at `daily_market_surveillance/operations/sector_rotation_intelligence.md`; it provides research-only, price-led sector and industry context and remains separate from the signal and opportunity lifecycle.

Historical and target-architecture source documents still exist under `docs/marketops/`. New operational documentation should prefer the concrete use-case folder unless it applies to all MarketOps use cases.
