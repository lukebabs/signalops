# S4 Provider Revision Evidence — 2026-08-13

Status: immutable revision observations and field-level deltas recorded; legacy canonical selection remains held pending review.

## Deployment

- Commit: `65de08e`.
- Migration: `000104_subscriber_global_eod_provider_revisions`.
- Policy: `subeodrevpolicy_massive_v1` / `s4-provider-revision-v1`.
- Provider policy: initial capture is immutable; revised capture is an immutable version; canonical selection is `hold_initial_pending_review`.

No provider request, normalized-ledger update, coverage activation, scheduler change, or watchlist mutation occurred during this backfill.

## Preserved observations

| Role | Count | Source |
|---|---:|---|
| Initial tenant-local capture | 2 | `tenant-local` / `src-massive` normalized EOD ledger, processing time 2026-08-12T22:01:57.893087Z |
| Global re-observation | 2 | Live run `subeodlive_3fe919b63640ecce8d10ceb2` |

Each observation retains asset, session, provider, source event/run, canonical payload, fingerprint, algorithm/observation version, quality state, provenance, and observation time.

## Field-level result

| Field class | AAPL | NVDA | Treatment |
|---|---|---|---|
| Open, high, low, close | unchanged | unchanged | informational immutable delta |
| Volume | provider revision | provider revision | review required |
| VWAP | provider revision | provider revision | review required |

The ledger contains twelve deltas: eight unchanged/informational OHLC rows and four provider-revision/review-required volume/VWAP rows. It confirms the live canary’s parity failure is attributable to a post-capture provider revision, not a changed price bar or an algorithm calculation mismatch.

## Gate status

The existing tenant-local result remains authoritative. The global re-observation is retained for evaluation only. No retry, provider correction propagation, EOD coverage activation, or cohort expansion is allowed until the canonical-selection policy is deliberately chosen and the parity contract is recalibrated.
