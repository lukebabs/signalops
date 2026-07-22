# Input features

- `range_position_252d`: close within the preceding 252-session high/low range.
- `rsi_14`, `return_5d`, `volume_ratio_10d`.
- `distance_sma_50_pct`, `distance_sma_200_pct`, `sma_50_slope_20d_pct`.
- `atr_14_pct` for risk level.
- `put_call_volume_ratio` and `put_call_volume_ratio_10d_deviation_pct` for speculative corroboration.

All inputs are persisted Market State observations. Missing, invalid, stale, or insufficient-history values remain unavailable; they are never coerced to zero.
