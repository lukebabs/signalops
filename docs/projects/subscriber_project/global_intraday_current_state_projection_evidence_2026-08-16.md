# Global Intraday Current-State Projection Evidence — 2026-08-16

Status: applied at the dedicated MarketOps database boundary; not yet exposed through a MarketOps API, UI view, or scheduler.

## Contract

Migration `000143_subscriber_global_intraday_current_state_projection` adds the restricted `subscriber_gateway_global_intraday_current_states` reader over platform-owned `intraday_snapshot` evidence only.

The reader is intentionally a **latest current-state projection**, not an intraday history endpoint:

- only `legacy_materialization` records with `payload.current_only_source=true` qualify;
- it selects one record per canonical global asset;
- it orders and reports freshness from immutable `payload.as_of_time`;
- it preserves status, stale marker, conditions, source payload, algorithm/version, fingerprint, validation contract, immutable baseline, and provenance;
- it does not derive freshness from `evidence_records.observed_at`, because legacy current-snapshot rows retain their original creation time.

## Applied proof

The migration was applied to the dedicated primary at `2026-08-16T22:09:17Z`.

| Projection check | Result |
| --- | ---: |
| rows | 132 |
| distinct canonical assets | 132 |
| earliest/latest payload `as_of_time` | 2026-08-14 04:00 UTC / 2026-08-14 04:00 UTC |
| `current_only_source` | true for every record |
| gateway role direct raw-evidence access | denied |

The restricted `signalops_subscriber_gateway` role can select the projection; `PUBLIC` cannot. The gateway application has not been changed to query it, so existing browser/API behavior remains unchanged.

## Operational boundary

This is a one-time imported current-state baseline. It is intentionally stale after the source’s last completed update until the later dual-run/writer phase is approved. The current legacy scheduler is still authoritative for ongoing intraday writes; this migration neither starts it nor redirects it.

## Central shadow-capture gate

Migration `000146_subscriber_global_intraday_shadow_capture` was applied to the dedicated MarketOps primary at `2026-08-17T00:07:26Z`; it adds an unscheduled, append-only central provider-capture worker. It resolves only the aggregate hot selector through the constrained selector role, records no tenant, user, or watchlist identity, and records per-symbol freshness parity only against the existing legacy projection. It is deliberately not a reader switch: the existing legacy scheduler and current-state view remain authoritative.

The worker is safe to invoke with `--dry-run` on a weekend: it makes no provider request and writes nothing. A live `--execute` capture remains a separately approved market-session action; it must produce zero provider failures and a documented freshness result before any projection or scheduler cutover is considered.

## Weekend dry-run evidence

On `2026-08-17`, the managed `subscriber-global-intraday-shadow-dry-run` completed against the dedicated MarketOps primary: **125** aggregate hot assets resolved; `market_open=false`; no session value was returned. The worker made no Massive request and the append-only `subscriber_global_intraday_shadow_capture_runs` ledger remained at **0** rows after the run.

## Approved one-shot release

Luke approved one no-retry Massive central intraday shadow capture for the aggregate hot selector. The managed release schedules exactly one execution for `2026-08-17 09:50 America/New_York`, five minutes after the legacy `09:45` interval and within the same 15-minute bucket. The one-shot does not enable any recurring central schedule and fails closed if the legacy intraday timer is inactive.

## Next gate

Define the temporary 132-symbol grandfathered global hot cohort and run it in shadow against the watchlist-derived global hot selector. The dual-run must prove identical membership and state freshness before a controlled central writer or any API/UI reader switch is enabled.
