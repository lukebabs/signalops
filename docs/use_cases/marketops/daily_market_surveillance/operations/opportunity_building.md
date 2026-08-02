# Opportunity Building

Use `signalops-marketops-opportunity-builder` to group compatible triggered G138 evaluations into research-only opportunities. The command reads persisted hypothesis definitions and evaluations, makes no provider calls, and writes no production signal state.

## Preflight

- Apply migration `000030_marketops_opportunities`.
- Set `SIGNALOPS_DATABASE_URL`.
- Confirm G138 evaluations exist for the target AAPL window.
- Use an inclusive session range and `--max-sessions` from 1 to 50.

Run dry-run first:

```bash
signalops-marketops-opportunity-builder \
  --tenant-id tenant-local \
  --symbol AAPL \
  --session-start 2026-07-01 \
  --session-end 2026-07-20 \
  --max-sessions 50 \
  --run-id operator-preflight \
  --dry-run
```

Remove `--dry-run` to upsert opportunities. G139 rejects symbols other than AAPL. The deterministic identity excludes run ID, so repeating the same logical inputs updates build lineage without duplicating an opportunity.

## Verification

- `evaluations` matches the bounded G138 rows read.
- Only eligible, triggered, non-invalidated evaluations contribute.
- `overlap_suppressed` reports weaker contributions from the same hypothesis family/domain.
- `conflict_links` reports opposing-direction contribution links.
- Every persisted row is `research_only=true`.
- `active` requires at least two independent domains and non-dominant conflict; otherwise lifecycle is `emerging`.
- Current AAPL data produces 24 `ineligible_evaluation` skips and zero opportunities.
- Repeated writes leave logical row counts unchanged.
- No signal, alert, insight, proposal, artifact, graph, trade, or outcome row is written.

Inspect results through `GET /v1/marketops/opportunities`. The detail endpoint is `GET /v1/marketops/opportunities/{opportunity_id}?tenant_id=...`.

## V2 convergence queue

V2 replaces the AAPL-only trigger gate for the analyst queue. It is a research-only, deterministic convergence record and requires at least two independent persisted sources that agree on the same asset, direction, and session date. Current sources are Risk/Reward, Exhaustive Reversal, tactical posture, and extreme options flow.

- The v2 record uses version `2` and source/evidence lineage, not hypothesis-evaluation IDs.
- `000068_marketops_convergence_opportunities` preserves the v1 hypothesis-lineage constraint while permitting valid v2 source/evidence lineage.
- A single-source condition or data from a different session never creates a candidate. Material cross-direction disagreement instead creates a separate non-directional `mixed-conviction review` when each source strength is at least `0.20`.
- The post-close workflow executes its normal cohorts, then runs tactical posture and Exhaustive Reversal, then performs a final `opportunity_build` pass for every ten-symbol cohort. This sequencing lets final daily algorithms participate without weakening the exact-session rule.
- On a successful non-dry-run build, older active v2 records for that symbol are marked `expired` before current-session convergence records are upserted. Historical records remain available through the lifecycle filter.
- V2 remains `research_only=true`; it does not create an alert, recommendation, trade, or operational action.

A healthy queue is selective, including zero candidates when no two independent sources agree. Queue cards display the agreeing source names; the detail view records their alignment strength and evidence identifiers.


## Outcome maturity sweep

The post-close workflow reruns `outcome_materialization` across a rolling 45-calendar-day window after the final convergence refresh. It upserts deterministic opportunity outcome rows, moving them from `pending` to `matured` as later canonical closes become available. Standard horizons are 1, 5, 10, and 20 trading sessions; analyst reporting should emphasize 1/5/10 until an adequate 20-session sample exists. Expired opportunities remain measurable because expiry changes queue lifecycle, not their historical outcome source.
