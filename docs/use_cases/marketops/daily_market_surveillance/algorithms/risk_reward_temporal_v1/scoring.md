# Scoring

Technical score is bounded to `-100..100`. It combines range position (25), RSI (20), volume/price divergence (15), trend regime (25), and five-session price trend (15). ATR sets low, medium, or high risk without changing direction.

- Bearish: range position ≥90%, RSI ≥60, or positive five-session price move on volume <0.8× its 10-session mean; a declining 50/200 regime is bearish.
- Bullish counterparts: range position ≤10%, RSI ≤40, or negative five-session price move on volume >1.2× mean; an improving 50/200 regime is bullish.
- Put/call corroboration is bearish only when ratio >1.0 and deviation ≥10%; bullish only when ratio <1.0 and deviation ≤-10%.

Put/call is canonical (puts divided by calls). It is speculative corroboration only and cannot set technical direction.
