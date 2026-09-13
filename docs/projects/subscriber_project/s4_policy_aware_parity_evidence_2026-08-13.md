# S4 Policy-Aware Parity Evidence — 2026-08-13

Status: remediation complete; four of four policy-aware comparisons matched. No
provider request, scheduler enablement, coverage activation, or legacy
MarketOps mutation occurred.

## Why the original comparison failed

The original raw canary reporter compared the tenant-local initial capture
directly with Massive's later global re-observation. AAPL and NVDA have
unchanged OHLC values and provider revisions to VWAP and volume. Those are two
valid immutable versions, not an algorithm calculation discrepancy.

The raw 0/2 mismatch reports remain retained as provider-revision evidence.
They are not deleted, amended, or relabelled as matched.

## Corrected contract

Migration `000109_subscriber_global_eod_policy_parity` adds an append-only
policy-parity ledger. The dedicated
`subscriber-global-eod-canary-policy-parity` command requires the existing
authorization and exactly two completed baseline rows. It has no market-data
client and makes no provider request.

For every canary asset the command validates both fixed contexts:

| Usage context | Selected immutable version | Comparison baseline |
| --- | --- | --- |
| `historical_assurance` | `initial_tenant_local_capture` | Original tenant-local normalized EOD event |
| `current_market_context` | `global_reobservation` | Global canary baseline result |

The report retains the selection-policy version, selected observation role,
both canonical payload fingerprints, comparison source, correlation evidence,
and any review-required revision fields.

## Recorded result

Authorization: `subeodauth_20260813_aapl_nvda`
Live run: `subeodlive_3fe919b63640ecce8d10ceb2`
Policy version: `s4-as-of-selection-v1`

| Symbol | Historical assurance | Current market context | Review-required fields |
| --- | --- | --- | --- |
| AAPL | matched | matched | `volume`, `vwap` |
| NVDA | matched | matched | `volume`, `vwap` |

The policy-parity ledger therefore contains four matched rows. Historical SAF,
outcome, and backtest consumers remain bound to the original capture; current
MarketOps context remains bound to the latest usable global revision.

## Gate effect

This closes the S4 parity-contract remediation. It does **not** enable a
recurring global worker, expand the two-symbol cohort, activate queued assets,
or change the tenant-local scheduler. Any cohort expansion remains a separate
explicit release decision with the current kill switch, bounded request budget,
and rollback controls intact.
