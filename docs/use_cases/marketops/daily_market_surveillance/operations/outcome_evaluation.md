# Forward Outcome Evaluation Operations

Use `signalops-marketops-outcome-materializer` to calculate bounded G140 forward outcomes from persisted hypothesis evaluations, opportunities, and normalized equity EOD events. The command makes no provider calls and does not mutate source records.

## Prerequisites

- Migration `000031_marketops_signal_outcomes` is applied.
- `SIGNALOPS_DATABASE_URL` points to the relational SignalOps database.
- `SIGNALOPS_TEMPORAL_DATABASE_URL` points to the normalized-event ledger.
- Source evaluations/opportunities and their origin-session equity EOD rows already exist.

## Dry Run

```bash
signalops-marketops-outcome-materializer \
  --tenant-id tenant-local \
  --symbol AAPL \
  --session-start 2026-07-01 \
  --session-end 2026-07-20 \
  --as-of 2026-07-20 \
  --max-sessions 50 \
  --threshold 0.02 \
  --run-id g140-operator-dry \
  --dry-run
```

Review `outcome_sources`, `matured`, `pending`, `missing_price`, and `skipped_reasons` before removing `--dry-run`.

## Write And Rerun

Run the same command without `--dry-run`. Repeat it with a different run ID to confirm the row count and deterministic identities remain stable. A pending row may advance when later persisted EOD sessions become available. Status progression is monotonic: matured rows do not regress, and missing-price rows do not return to pending.

## Database Checks

```sql
SELECT outcome_status, horizon_sessions, count(*)
FROM marketops_signal_outcomes
WHERE tenant_id = 'tenant-local' AND symbol = 'AAPL'
GROUP BY outcome_status, horizon_sessions
ORDER BY horizon_sessions, outcome_status;

SELECT source_type, source_id, origin_session_date, horizon_sessions,
       matured_session_date, forward_return, directional_hit,
       threshold_hit, days_to_threshold, outcome_event_ids
FROM marketops_signal_outcomes
WHERE tenant_id = 'tenant-local' AND symbol = 'AAPL'
ORDER BY origin_session_date, source_type, source_id, horizon_sessions;
```

## API Checks

Unauthenticated requests should return `401` when gateway auth is enabled.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:18000/v1/marketops/outcomes?tenant_id=tenant-local&symbol=AAPL&limit=50"
```

Use `source_type`, `source_id`, `hypothesis_key`, `hypothesis_version`, `direction`, `outcome_status`, `horizon_sessions`, `session_start`, and `session_end` to narrow the list.

## Interpretation Controls

- Zero rows are valid when no eligible triggered source exists.
- Do not treat `pending` as a failed prediction.
- Do not treat `missing_price` as a zero return.
- Directional hit is unavailable for non-directional sources.
- Realized-volatility change is unavailable when either comparison window has insufficient valid returns.
- Outcome confidence is not forecast probability; G140 records observations, not promotion decisions.
