# Research Hypothesis Evaluation

Use `signalops-marketops-hypothesis-evaluator` to evaluate the G138 research pack over existing G137 AAPL states. The command reads persisted states, features, transitions, and evidence. It makes no provider calls and writes no production signals.

## Preflight

- Apply migration `000029_marketops_hypothesis_research` after G136 migration `000028`.
- Set `SIGNALOPS_DATABASE_URL` for the relational MarketOps ledgers.
- Confirm the target AAPL states exist.
- Choose an inclusive session range and bounded `--max-sessions` value.

Run a dry-run first:

```bash
signalops-marketops-hypothesis-evaluator \
  --tenant-id tenant-local \
  --symbol AAPL \
  --session-start 2026-07-01 \
  --session-end 2026-07-20 \
  --max-sessions 100 \
  --run-id operator-preflight \
  --dry-run
```

Remove `--dry-run` to register the four research definitions and upsert their evaluations. G138 rejects symbols other than AAPL. A different run ID updates evaluation-run lineage but does not change deterministic IDs or create duplicate logical evaluations.

## Verification

- Definition count is four: H001, H004, H006, and H007 v1.
- Evaluation count equals state count multiplied by four.
- `triggered=true` always implies `eligible=true`.
- Rejected evaluations contain reason codes and no trigger/confidence score.
- Current sparse AAPL states produce zero eligible and zero triggered rows.
- Repeating a write leaves definition and evaluation counts unchanged.
- No signal, alert, insight, DSM artifact, graph proposal, algorithm proposal, or opportunity row is produced by this command.

Inspect results through `GET /v1/marketops/hypotheses` and `GET /v1/marketops/hypothesis-evaluations` after deploying the current gateway.
