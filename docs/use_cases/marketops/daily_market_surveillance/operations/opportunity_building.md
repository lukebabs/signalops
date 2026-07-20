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
