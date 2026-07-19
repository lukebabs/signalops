# G137 AAPL Market State Vertical Slice

Status: implemented backend materialization and live validation

Date: 2026-07-19

## Purpose

G137 implements the first bounded Market State Intelligence calculation path over already persisted AAPL evidence. It turns normalized equity EOD events plus approved option chain/distribution rows into the G136 feature, state, transition, and evidence ledgers without provider calls or production-signal side effects.

## Implemented Scope

- Added `signalops-marketops-state-materializer` with explicit tenant, AAPL symbol, date-window, maximum-session, run-id, and dry-run controls.
- Registered 24 versioned feature definitions covering selected underlying momentum, implied-volatility surface, premium, positioning, liquidity, and quality domains.
- Materializes 25 feature slots per available session. The `iv` definition has separate 30-DTE 25-delta put and call dimensions.
- Calculates point-in-time underlying returns, Wilder RSI/ATR, SMA distance, trailing volume ratio, and opening gap when sufficient normalized equity history exists. The CLI reads a bounded pre-window warm-up while materializing only sessions inside the requested window.
- Selects deterministic 30/60/90-DTE ATM IV and 30-DTE 25-delta put/call IV cells from eligible persisted contracts.
- Persists premium cells as `missing` with `bid_ask_not_persisted`; the implementation does not substitute daily close for a bid/ask midpoint.
- Calculates put/call OI and volume ratios only under their declared source-quality rules. Unusable OI does not receive a numeric value.
- Persists chain quality and surface coverage, including valid numeric zero coverage where a chain exists but no eligible surface cell exists.
- Builds one canonical versioned state for every unioned equity/options session and links all 25 feature-observation IDs.
- Builds deterministic one-eligible-state-session absolute-difference transitions for numeric observations that are usable on both sides.
- Emits reusable return, usable OI ratio, and material ATM-IV-change evidence only after quality gates pass.
- Adds a first-class Docker image target named `marketops-state-materializer`.

## Operator Command

```bash
signalops-marketops-state-materializer \
  --tenant-id tenant-local \
  --symbol AAPL \
  --window-start 2026-07-01 \
  --window-end 2026-07-20 \
  --max-sessions 100 \
  --run-id g137-aapl
```

`SIGNALOPS_DATABASE_URL` and `SIGNALOPS_TEMPORAL_DATABASE_URL` are required. `--window-start` is inclusive and `--window-end` is exclusive. Use `--dry-run` to calculate counts and quality without writes.

## Live Validation

The bounded local AAPL window contained:

- 3 normalized equity EOD events;
- 3 approved option distributions;
- 5 persisted option contracts;
- 6 unioned sessions.

All repeated write runs produced the same logical result:

- 24 feature definitions;
- 150 feature observations;
- 6 market states;
- 11 state transitions;
- 2 reusable evidence records.

The persisted ledger remained at those exact counts after repeated runs. Every state has 25 feature IDs and a database check found zero unresolved feature references.

Live AAPL state quality is currently one `missing` state and five `partial` states. All three available AAPL OI-ratio observations are explicitly `invalid` because their persisted source quality is `denominator_zero` or `all_zero`; they have no numeric value and generated zero OI evidence. The two emitted evidence rows are valid one-session underlying-return observations.

The positive unit fixture uses 25 equity sessions and an eligible five-cell IV surface. It verifies a usable state, deterministic ATM/25-delta selection, usable put/call OI evidence, premium blocking, transitions, exact lineage, and stable IDs across different run IDs. A separate 65-session fixture verifies the bounded 60-session replay acceptance path and full lineage for every state.

## Safety Boundary

G137 does not add:

- Massive or other provider calls;
- scheduling or Top 50 fanout;
- bid/ask synthesis from option daily close;
- hypothesis definitions or evaluations;
- signal proposals or production signals;
- graph, Syncratic, alert, insight, or frontend changes.

## Next Gate

G138 should introduce the bounded research hypothesis registry/evaluator for H001, H004, H006, and H007 over these first-class states and evidence. It must persist trigger and non-trigger reason codes and must remain research-only until historical source coverage is materially broader.
