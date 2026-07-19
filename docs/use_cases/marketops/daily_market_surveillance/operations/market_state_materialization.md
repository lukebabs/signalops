# Market State Materialization

Use `signalops-marketops-state-materializer` to rebuild the bounded G137 AAPL state path from existing persisted evidence. The command does not call Massive and does not write signals.

## Preflight

- Confirm migration `000028_marketops_market_state_foundation` is applied.
- Set `SIGNALOPS_DATABASE_URL` for the relational MarketOps ledgers.
- Set `SIGNALOPS_TEMPORAL_DATABASE_URL` for normalized equity EOD reads.
- Choose an inclusive start date, exclusive end date, and bounded `--max-sessions` value. The command reads up to 90 calendar days of pre-window equity and option warm-up but does not materialize warm-up sessions.

Run dry-run first:

```bash
signalops-marketops-state-materializer \
  --tenant-id tenant-local \
  --symbol AAPL \
  --window-start 2026-07-01 \
  --window-end 2026-07-20 \
  --max-sessions 100 \
  --run-id operator-preflight \
  --dry-run
```

Remove `--dry-run` to upsert. Repeating the same evidence window is idempotent by deterministic feature, state, transition, and evidence keys. A new run ID updates calculation/build lineage but does not create duplicate logical rows.

## Verification

- State count matches the bounded union of equity and option session dates.
- Every state has `feature_count=25`, `required_feature_count=25`, and 25 lineage IDs.
- Missing and invalid feature observations have no numeric value.
- AAPL OI evidence is absent while `put_call_oi_ratio` source quality is unusable.
- `mid_premium`, `extrinsic_premium`, and `premium_pct_spot` remain missing until positive persisted bid/ask inputs exist.
- A second run leaves aggregate ledger row counts unchanged.

Read results through the G136 `/v1/marketops/features/observations`, `/v1/marketops/states`, `/v1/marketops/states/{id}/lineage`, `/v1/marketops/transitions`, and `/v1/marketops/evidence` endpoints.
