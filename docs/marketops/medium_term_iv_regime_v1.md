# Medium-Term Implied-Volatility Regime v1

## Purpose

This deterministic MarketOps feature summarizes the existing 30, 60, and 90 DTE option surface without predicting price direction. It provides repeatable context for tactical research and earnings workflows.

## Inputs and bands

- Requires usable 30-DTE ATM IV, 20-session realized volatility, and at least two usable ATM cells across 30/60/90 DTE.
- `elevated_premium`: 30-DTE ATM IV / 20-session RV is at least 1.25x.
- `neutral`: IV/RV is 0.85x through 1.15x; `discounted` is below 0.85x; remaining usable values are `intermediate`.
- Existing 25-delta put/call IV, skew, and IV-change evidence remains available for direction-specific corroboration.

## Consumers and limits

Market State displays the regime and full feature lineage. H001 can increase corroboration when elevated IV aligns with its independent overbought, put-IV expansion, premium, and OI evidence. EEOM, Risk/Reward, and Exhaustive Reversal apply at most a 10-point equivalent adjustment only after their own direction is established. Opportunities continue to use the existing hypothesis and convergence controls.

The feature makes no directional claim from high IV alone. VC and DOSM remain strategic and do not consume IV/RV. IV rank, percentile, and z-score require more prospective history and are intentionally excluded.
